package admin

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// ChatGPTAccountHandler manages ChatGPT Web OAuth accounts as a pooled resource.
type ChatGPTAccountHandler struct {
	accountRepo service.AccountRepository
}

// NewChatGPTAccountHandler creates a new ChatGPT account handler.
func NewChatGPTAccountHandler(accountRepo service.AccountRepository) *ChatGPTAccountHandler {
	return &ChatGPTAccountHandler{accountRepo: accountRepo}
}

// ListAccounts returns all ChatGPT Web accounts.
// GET /api/v1/admin/chatgpt/accounts
func (h *ChatGPTAccountHandler) ListAccounts(c *gin.Context) {
	ctx := c.Request.Context()
	accounts, err := h.chatgptAccounts(ctx)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, gin.H{"items": accounts})
}

// CreateAccounts adds one or more ChatGPT Web accounts.
// POST /api/v1/admin/chatgpt/accounts
func (h *ChatGPTAccountHandler) CreateAccounts(c *gin.Context) {
	ctx := c.Request.Context()
	var body struct {
		Tokens   []string         `json:"tokens"`
		Accounts []map[string]any `json:"accounts"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	existing, _ := h.chatgptAccountTokenSet(ctx)
	added, skipped := 0, 0

	// From tokens list
	for _, token := range body.Tokens {
		token = strings.TrimSpace(token)
		if token == "" || existing[token] {
			skipped++
			continue
		}
		if err := h.createChatGPTAccount(ctx, token, "oauth", nil); err != nil {
			skipped++
			continue
		}
		existing[token] = true
		added++
	}
	// From accounts list
	for _, acc := range body.Accounts {
		token := extractMapString(acc, "access_token")
		if token == "" {
			token = extractMapString(acc, "accessToken")
		}
		if token == "" || existing[token] {
			skipped++
			continue
		}
		accType := extractMapString(acc, "type")
		if accType == "" {
			accType = "oauth"
		}
		if err := h.createChatGPTAccount(ctx, token, accType, acc); err != nil {
			skipped++
			continue
		}
		existing[token] = true
		added++
	}
	response.Success(c, gin.H{"added": added, "skipped": skipped})
}

// DeleteAccounts removes accounts by token list.
// DELETE /api/v1/admin/chatgpt/accounts
func (h *ChatGPTAccountHandler) DeleteAccounts(c *gin.Context) {
	ctx := c.Request.Context()
	var body struct {
		Tokens []string `json:"tokens"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	tokenIDs, _ := h.chatgptAccountTokenIDs(ctx)
	removed := 0
	for _, token := range body.Tokens {
		token = strings.TrimSpace(token)
		if id, ok := tokenIDs[token]; ok {
			_ = h.accountRepo.Delete(ctx, id)
			removed++
		}
	}
	response.Success(c, gin.H{"removed": removed})
}

// RefreshAccounts refreshes ChatGPT Web user info for the given tokens.
// POST /api/v1/admin/chatgpt/accounts/refresh
func (h *ChatGPTAccountHandler) RefreshAccounts(c *gin.Context) {
	ctx := c.Request.Context()
	var body struct {
		Tokens []string `json:"access_tokens"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	tokenIDs, _ := h.chatgptAccountTokenIDs(ctx)
	refreshed := 0
	var errs []map[string]string
	for _, token := range body.Tokens {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		id, ok := tokenIDs[token]
		if !ok {
			errs = append(errs, map[string]string{"token": token, "error": "account not found"})
			continue
		}
		acc, err := h.accountRepo.GetByID(ctx, id)
		if err != nil || acc == nil {
			errs = append(errs, map[string]string{"token": token, "error": "account not found"})
			continue
		}
		// Use existing OpenAI token provider to refresh
		_ = acc // TODO: call refresh when ChatGPT Web user info fetch is implemented
		refreshed++
	}
	response.Success(c, gin.H{"refreshed": refreshed, "errors": errs})
}

// UpdateAccount status.
// POST /api/v1/admin/chatgpt/accounts/update
func (h *ChatGPTAccountHandler) UpdateAccount(c *gin.Context) {
	ctx := c.Request.Context()
	var body struct {
		AccessToken string `json:"access_token"`
		Status      string `json:"status"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	tokenIDs, _ := h.chatgptAccountTokenIDs(ctx)
	id, ok := tokenIDs[strings.TrimSpace(body.AccessToken)]
	if !ok {
		response.NotFound(c, "account not found")
		return
	}
	if body.Status != "" {
		acc, err := h.accountRepo.GetByID(ctx, id)
		if err == nil && acc != nil {
			acc.Status = body.Status
			_ = h.accountRepo.Update(ctx, acc)
		}
	}
	response.Success(c, gin.H{"ok": true})
}

// ExportAccounts exports accounts as JSON.
// POST /api/v1/admin/chatgpt/accounts/export
func (h *ChatGPTAccountHandler) ExportAccounts(c *gin.Context) {
	ctx := c.Request.Context()
	var body struct {
		Tokens []string `json:"access_tokens"`
	}
	_ = c.ShouldBindJSON(&body)
	accounts, err := h.chatgptAccounts(ctx)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	if len(body.Tokens) == 0 {
		response.Success(c, gin.H{"items": accounts})
		return
	}
	tokenSet := make(map[string]bool)
	for _, t := range body.Tokens {
		tokenSet[strings.TrimSpace(t)] = true
	}
	var exported []map[string]any
	for _, acc := range accounts {
		t, _ := acc["access_token"].(string)
		if tokenSet[t] {
			exported = append(exported, acc)
		}
	}
	response.Success(c, gin.H{"items": exported})
}

// RegisterConfig returns register machine configuration.
// GET /api/v1/admin/chatgpt/register
func (h *ChatGPTAccountHandler) RegisterConfig(c *gin.Context) {
	response.Success(c, gin.H{
		"register": map[string]any{
			"enabled": false,
			"mode":    "total",
			"total":   1,
			"threads": 1,
			"stats":   map[string]any{"success": 0, "fail": 0, "done": 0, "running": 0},
		},
	})
}

// ---- internal helpers ----

func (h *ChatGPTAccountHandler) chatgptAccounts(ctx context.Context) ([]map[string]any, error) {
	accounts, err := h.accountRepo.ListByPlatform(ctx, service.PlatformOpenAI)
	if err != nil {
		return nil, err
	}
	var result []map[string]any
	for i := range accounts {
		acc := &accounts[i]
		if !isChatGPTWebAccount(acc) {
			continue
		}
		token := acc.GetCredential("access_token")
		if token == "" {
			continue
		}
		result = append(result, map[string]any{
			"access_token": token,
			"type":         acc.Type,
			"status":       acc.Status,
			"name":         acc.Name,
			"email":        acc.GetCredential("email"),
			"user_id":      acc.GetCredential("user_id"),
			"plan_type":    acc.GetCredential("plan_type"),
			"created_at":   acc.CreatedAt.Format(time.RFC3339),
		})
	}
	return result, nil
}

func (h *ChatGPTAccountHandler) chatgptAccountTokenIDs(ctx context.Context) (map[string]int64, error) {
	accounts, err := h.accountRepo.ListByPlatform(ctx, service.PlatformOpenAI)
	if err != nil {
		return nil, err
	}
	result := make(map[string]int64)
	for i := range accounts {
		acc := &accounts[i]
		if !isChatGPTWebAccount(acc) {
			continue
		}
		token := acc.GetCredential("access_token")
		if token != "" {
			result[token] = acc.ID
		}
	}
	return result, nil
}

func (h *ChatGPTAccountHandler) chatgptAccountTokenSet(ctx context.Context) (map[string]bool, error) {
	accounts, err := h.accountRepo.ListByPlatform(ctx, service.PlatformOpenAI)
	if err != nil {
		return nil, err
	}
	result := make(map[string]bool)
	for i := range accounts {
		acc := &accounts[i]
		if !isChatGPTWebAccount(acc) {
			continue
		}
		token := acc.GetCredential("access_token")
		if token != "" {
			result[token] = true
		}
	}
	return result, nil
}

func (h *ChatGPTAccountHandler) createChatGPTAccount(ctx context.Context, token, accType string, extra map[string]any) error {
	name := "ChatGPT"
	if len(token) >= 8 {
		name = "ChatGPT-" + token[:8]
	}
	creds := map[string]any{"access_token": token}
	if extra != nil {
		if rt := extractMapString(extra, "refresh_token"); rt != "" {
			creds["refresh_token"] = rt
		}
		if idt := extractMapString(extra, "id_token"); idt != "" {
			creds["id_token"] = idt
		}
		if email := extractMapString(extra, "email"); email != "" {
			creds["email"] = email
			name = email
		}
	}
	return h.accountRepo.Create(ctx, &service.Account{
		Name:        name,
		Platform:    service.PlatformOpenAI,
		Type:        accType,
		Credentials: creds,
		Extra:       map[string]any{"openai_backend_mode": "chatgpt_web"},
		Status:      service.StatusActive,
		Concurrency: 3,
		Priority:    50,
	})
}

func isChatGPTWebAccount(acc *service.Account) bool {
	if acc == nil || acc.Extra == nil {
		return false
	}
	v, ok := acc.Extra["openai_backend_mode"]
	return ok && fmt.Sprint(v) == "chatgpt_web"
}

func extractMapString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}
