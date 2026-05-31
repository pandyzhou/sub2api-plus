package service

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"sync"
	"time"
)

var ErrNoAvailableImageToken = errors.New("no available image token")

// isChatGPTImageAccountAvailable checks whether the account has image generation quota remaining.
// An account is available if its status is not disabled/rate-limited/abnormal AND
// either its image quota is unknown (paid user) or its quota > 0.
func isChatGPTImageAccountAvailable(acc *Account) bool {
	if acc == nil {
		return false
	}
	switch acc.Status {
	case "禁用", "限流", "异常":
		return false
	}
	if acc.Extra != nil {
		if unknown, ok := acc.Extra["image_quota_unknown"].(bool); ok && unknown {
			return true
		}
	}
	return intValue(acc.Extra["quota"]) > 0
}

// tryAcquireImageSlot attempts to find an available image account and acquire a concurrency slot.
// Returns the account, its raw access_token, and whether a slot was acquired.
// Must be called with s.imageMu held.
func (s *ChatGPTAccountPoolService) tryAcquireImageSlot(ctx context.Context, maxConcurrency int) (*Account, string, bool) {
	accounts, err := s.listChatGPTAccounts(ctx)
	if err != nil {
		slog.Error("chatgpt_image_pool: failed to list accounts", "error", err)
		return nil, "", false
	}

	// Collect ready candidate accounts
	type candidate struct {
		acc   *Account
		token string
	}
	ready := make([]candidate, 0, len(accounts))
	for i := range accounts {
		acc := &accounts[i]
		if !isChatGPTImageAccountAvailable(acc) {
			continue
		}
		token := acc.GetCredential("access_token")
		if token == "" {
			continue
		}
		ready = append(ready, candidate{acc: acc, token: token})
	}

	if len(ready) == 0 {
		return nil, "", false
	}

	// Round-robin search for a token with an available slot
	for j := 0; j < len(ready); j++ {
		idx := (s.imageIndex + j) % len(ready)
		c := ready[idx]
		if s.imageInflight[c.token] < maxConcurrency {
			s.imageIndex = (idx + 1) % len(ready)
			s.imageInflight[c.token]++
			return c.acc, c.token, true
		}
	}

	return nil, "", false
}

// GetAvailableImageAccount returns an available image account, its raw access_token, and a release function.
// The caller should use OpenAITokenProvider.GetAccessToken(account) to get a refreshed token.
// The caller MUST call the returned release function when the image operation completes.
// Blocks up to 30 seconds waiting for a slot to become available.
func (s *ChatGPTAccountPoolService) GetAvailableImageAccount() (*Account, string, func(), error) {
	if s == nil {
		return nil, "", nil, fmt.Errorf("account pool service is not configured")
	}
	ctx := context.Background()
	cfg := s.GetConfig(ctx)
	maxConcurrency := cfg.ImageAccountConcurrency
	if maxConcurrency < 1 {
		maxConcurrency = 3
	}

	s.imageMu.Lock()
	defer s.imageMu.Unlock()

	deadline := time.Now().Add(30 * time.Second)

	for {
		acc, token, found := s.tryAcquireImageSlot(ctx, maxConcurrency)
		if found {
			capturedToken := token
			var once sync.Once
			release := func() {
				once.Do(func() {
					s.ReleaseImageSlot(capturedToken)
				})
			}
			return acc, token, release, nil
		}

		if time.Now().After(deadline) {
			return nil, "", nil, ErrNoAvailableImageToken
		}

		// Wait up to 1 second for a slot to be released, then retry.
		timer := time.AfterFunc(time.Second, func() {
			s.imageCond.Broadcast()
		})
		s.imageCond.Wait()
		timer.Stop()
	}
}

// MarkImageResult records the outcome of an image generation attempt.
// On success the account's success counter is incremented and quota is decremented (when known).
// On failure the fail counter is incremented.
// This also releases the concurrency slot for the given token.
func (s *ChatGPTAccountPoolService) MarkImageResult(token string, success bool) {
	if s == nil || token == "" {
		return
	}
	ctx := context.Background()

	acc, err := s.findByAccessToken(ctx, token)
	if err != nil || acc == nil {
		slog.Warn("chatgpt_image_pool: mark result for unknown token", "token", AnonymizeTokenForChatGPTPool(token))
		s.ReleaseImageSlot(token)
		return
	}

	if acc.Extra == nil {
		acc.Extra = map[string]any{}
	}

	if success {
		acc.Extra["success"] = intValue(acc.Extra["success"]) + 1
		// Decrement quota if it is a known value
		if !isBoolTrue(acc.Extra["image_quota_unknown"]) {
			q := intValue(acc.Extra["quota"])
			if q > 0 {
				acc.Extra["quota"] = q - 1
			}
			if q-1 <= 0 {
				acc.Status = "限流"
			}
		}
	} else {
		acc.Extra["fail"] = intValue(acc.Extra["fail"]) + 1
	}

	if err := s.accounts.Update(ctx, acc); err != nil {
		slog.Error("chatgpt_image_pool: failed to update account after mark result", "error", err)
	}

	s.ReleaseImageSlot(token)
}

// ReleaseImageSlot decrements the in-flight count for the given token and wakes up waiters.
func (s *ChatGPTAccountPoolService) ReleaseImageSlot(token string) {
	if s == nil || token == "" {
		return
	}
	s.imageMu.Lock()
	defer s.imageMu.Unlock()

	current := s.imageInflight[token]
	if current <= 1 {
		delete(s.imageInflight, token)
	} else {
		s.imageInflight[token] = current - 1
	}
	s.imageCond.Broadcast()
}

// GetImagePoolStats returns a snapshot of the image account pool statistics.
func (s *ChatGPTAccountPoolService) GetImagePoolStats() map[string]any {
	if s == nil {
		return map[string]any{"error": "service not configured"}
	}
	ctx := context.Background()
	cfg := s.GetConfig(ctx)

	accounts, err := s.listChatGPTAccounts(ctx)
	if err != nil {
		return map[string]any{"error": err.Error()}
	}

	total := 0
	available := 0
	totalQuota := 0
	unknownQuota := 0
	statusCounts := map[string]int{}

	for i := range accounts {
		acc := &accounts[i]
		total++
		if isChatGPTImageAccountAvailable(acc) {
			available++
		}
		if acc.Extra != nil {
			if isBoolTrue(acc.Extra["image_quota_unknown"]) {
				unknownQuota++
			} else {
				totalQuota += intValue(acc.Extra["quota"])
			}
		}
		statusCounts[acc.Status]++
	}

	s.imageMu.Lock()
	inflightCopy := make(map[string]int, len(s.imageInflight))
	totalInflight := 0
	for k, v := range s.imageInflight {
		inflightCopy[k] = v
		totalInflight += v
	}
	s.imageMu.Unlock()

	return map[string]any{
		"total_accounts":           total,
		"available_accounts":       available,
		"total_known_quota":        totalQuota,
		"unknown_quota_accounts":   unknownQuota,
		"image_account_concurrency": cfg.ImageAccountConcurrency,
		"total_inflight":           totalInflight,
		"inflight_per_token":       inflightCopy,
		"status_counts":            statusCounts,
	}
}

// isBoolTrue returns true if the value is a bool with value true.
func isBoolTrue(v any) bool {
	b, ok := v.(bool)
	return ok && b
}
