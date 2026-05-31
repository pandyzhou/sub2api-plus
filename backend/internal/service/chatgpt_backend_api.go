package service

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/imroc/req/v3"
	utls "github.com/refraction-networking/utls"
)

// ---------------------------------------------------------------------------
// Constants
// ---------------------------------------------------------------------------

const (
	defaultChatGPTBaseURL    = "https://chatgpt.com"
	defaultClientVersion     = "prod-5c63a9fb6f8bb7a7ef58a5c9afcab4914da46b6b"
	defaultClientBuildNumber = "14337326319"
	defaultPowScript         = "https://chatgpt.com/backend-api/sentinel/sdk.js"
	defaultUserAgent         = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/146.0.0.0 Safari/537.36 Edg/146.0.0.0"
	defaultImageModelSlug    = "gpt-image-2"
	defaultImagePollTimeout  = 120 * time.Second
	defaultImagePollInitial  = 10 * time.Second
	defaultImagePollInterval = 10 * time.Second
)

// ---------------------------------------------------------------------------
// Errors
// ---------------------------------------------------------------------------

var (
	ErrInvalidAccessToken = fmt.Errorf("invalid access token")
	ErrArkoseRequired     = fmt.Errorf("arkose token required, not implemented")
	ErrImagePollTimeout   = fmt.Errorf("image poll timeout")
)

// ChatRequirements holds sentinel / proof / turnstile tokens required by the ChatGPT web backend.
type ChatRequirements struct {
	Token          string
	ProofToken     string
	TurnstileToken string
	SOToken        string
}

// ---------------------------------------------------------------------------
// Chrome TLS dialer using utls
// ---------------------------------------------------------------------------

// ChatGPTBackendAPI is a standalone ChatGPT web backend-api client that
// implements the full image-generation pipeline:
//
//	bootstrap → sentinel token → conduit token → SSE image gen → extract file_id → download
type ChatGPTBackendAPI struct {
	accessToken string
	baseURL     string
	proxyURL    string

	clientVersion string
	clientBuild   string
	userAgent     string
	deviceID      string
	sessionID     string

	httpClient       *req.Client
	powScriptSources []string
	powDataBuild     string
}

func NewChatGPTBackendAPI(accessToken string, proxyURL string) *ChatGPTBackendAPI {
	client := req.C().
		ImpersonateChrome().
		SetTLSFingerprint(utls.HelloChrome_131).
		SetTimeout(300 * time.Second).
		SetRedirectPolicy(req.NoRedirectPolicy())

	if proxyURL != "" {
		client.SetProxyURL(proxyURL)
	}

	return &ChatGPTBackendAPI{
		accessToken:      accessToken,
		baseURL:          defaultChatGPTBaseURL,
		proxyURL:         proxyURL,
		clientVersion:    defaultClientVersion,
		clientBuild:      defaultClientBuildNumber,
		userAgent:        defaultUserAgent,
		deviceID:         uuid.NewString(),
		sessionID:        uuid.NewString(),
		httpClient:       client,
		powScriptSources: nil,
		powDataBuild:     "",
	}
}

// ---------------------------------------------------------------------------
// Common helpers
// ---------------------------------------------------------------------------

func (c *ChatGPTBackendAPI) baseHeaders() map[string]string {
	return map[string]string{
		"User-Agent":                  c.userAgent,
		"Origin":                      c.baseURL,
		"Referer":                     c.baseURL + "/",
		"Accept":                      "*/*",
		"Accept-Language":             "zh-CN,zh;q=0.9,en;q=0.8,en-US;q=0.7",
		"Cache-Control":               "no-cache",
		"Pragma":                      "no-cache",
		"Priority":                    "u=1, i",
		"Oai-Device-Id":               c.deviceID,
		"Oai-Language":                "en-US",
		"Chatgpt-App-Version":         c.clientVersion,
		"Sec-Ch-Ua":                   `"Microsoft Edge";v="143", "Chromium";v="143", "Not A(Brand";v="24"`,
		"Sec-Ch-Ua-Arch":              `"x86"`,
		"Sec-Ch-Ua-Bitness":           `"64"`,
		"Sec-Ch-Ua-Full-Version":      `"143.0.3650.96"`,
		"Sec-Ch-Ua-Full-Version-List": `"Microsoft Edge";v="143.0.3650.96", "Chromium";v="143.0.7499.147", "Not A(Brand";v="24.0.0.0"`,
		"Sec-Ch-Ua-Mobile":            "?0",
		"Sec-Ch-Ua-Model":             `""`,
		"Sec-Ch-Ua-Platform":          `"Windows"`,
		"Sec-Ch-Ua-Platform-Version":  `"19.0.0"`,
		"Sec-Fetch-Dest":              "empty",
		"Sec-Fetch-Mode":              "cors",
		"Sec-Fetch-Site":              "same-origin",
	}
}

func (c *ChatGPTBackendAPI) apiHeaders(path string, extra map[string]string) map[string]string {
	h := c.baseHeaders()
	h["Authorization"] = "Bearer " + c.accessToken
	h["Referer"] = c.baseURL + path
	h["X-OpenAI-Target-Path"] = path
	h["X-OpenAI-Target-Route"] = path
	for k, v := range extra {
		h[k] = v
	}
	return h
}

func (c *ChatGPTBackendAPI) applyHeaders(r *req.Request, headers map[string]string) {
	for k, v := range headers {
		r.SetHeader(k, v)
	}
}

// ---------------------------------------------------------------------------
// 1. Bootstrap — GET / and extract PoW script sources + data-build
// ---------------------------------------------------------------------------

// Bootstrap fetches the ChatGPT homepage and extracts PoW resources
// (script sources and data-build attribute).
func (c *ChatGPTBackendAPI) Bootstrap(ctx context.Context) error {
	bootstrapHeaders := map[string]string{
		"User-Agent":                c.userAgent,
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
		"Accept-Language":           "zh-CN,zh;q=0.9,en;q=0.8",
		"Sec-Ch-Ua":                 `"Microsoft Edge";v="143", "Chromium";v="143", "Not A(Brand";v="24"`,
		"Sec-Ch-Ua-Mobile":          "?0",
		"Sec-Ch-Ua-Platform":        `"Windows"`,
		"Sec-Fetch-Dest":            "document",
		"Sec-Fetch-Mode":            "navigate",
		"Sec-Fetch-Site":            "none",
		"Sec-Fetch-User":            "?1",
		"Upgrade-Insecure-Requests": "1",
	}

	r := c.httpClient.R().SetContext(ctx)
	c.applyHeaders(r, bootstrapHeaders)

	resp, err := r.Get(c.baseURL + "/")
	if err != nil {
		return fmt.Errorf("bootstrap fetch: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, err := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if err != nil {
		return fmt.Errorf("bootstrap read body: %w", err)
	}
	if resp.StatusCode >= 400 {
		// 403 from Cloudflare is expected for bot-like clients.
		// Fall back to default PoW resources instead of failing.
		slog.Warn("chatgpt_bootstrap_fallback", "status", resp.StatusCode, "using_defaults", true)
		c.powScriptSources = []string{defaultPowScript}
		c.powDataBuild = ""
		return nil
	}

	c.powScriptSources, c.powDataBuild = parseChatGPTBackendPowResources(string(body))
	if len(c.powScriptSources) == 0 {
		c.powScriptSources = []string{defaultPowScript}
	}
	return nil
}

// ---------------------------------------------------------------------------
// 2. GetSentinelToken — POST /backend-api/sentinel/chat-requirements
// ---------------------------------------------------------------------------

// GetSentinelToken gets a sentinel token using the proven register code approach.
// Uses sentinel.openai.com with text/plain content type (same as chatgpt_register_openai.go).
func (c *ChatGPTBackendAPI) GetSentinelToken(ctx context.Context) (*ChatRequirements, error) {
	// 使用注册机的 sentinel token 生成方式（已验证可工作）
	generator := newSentinelTokenGenerator(c.deviceID, c.userAgent)
	payload := map[string]any{
		"p":    generator.requirementsToken(),
		"id":   c.deviceID,
		"flow": "chat",
	}
	payloadBytes, _ := json.Marshal(payload)

	sentinelHeaders := map[string]string{
		"Content-Type":       "text/plain;charset=UTF-8",
		"Referer":            "https://sentinel.openai.com/backend-api/sentinel/frame.html",
		"Origin":             "https://sentinel.openai.com",
		"User-Agent":         c.userAgent,
		"Sec-Ch-Ua":          `"Microsoft Edge";v="143", "Chromium";v="143", "Not A(Brand";v="24"`,
		"Sec-Ch-Ua-Mobile":   "?0",
		"Sec-Ch-Ua-Platform": `"Windows"`,
	}

	r := c.httpClient.R().SetContext(ctx)
	c.applyHeaders(r, sentinelHeaders)
	r.SetBodyBytes(payloadBytes)

	resp, err := r.Post("https://sentinel.openai.com/backend-api/sentinel/req")
	if err != nil {
		return nil, fmt.Errorf("sentinel fetch: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode == 401 {
		return nil, ErrInvalidAccessToken
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("sentinel failed: HTTP %d: %s", resp.StatusCode, truncateBytes(body, 300))
	}

	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, fmt.Errorf("sentinel json: %w", err)
	}

	token := strings.TrimSpace(fmt.Sprint(data["token"]))
	if token == "" {
		return nil, fmt.Errorf("missing sentinel token in response")
	}

	return &ChatRequirements{
		Token:   token,
		SOToken: strings.TrimSpace(fmt.Sprint(data["so_token"])),
	}, nil
}

// ---------------------------------------------------------------------------
// 3. GetConduitToken — POST /backend-api/f/conversation/prepare
// ---------------------------------------------------------------------------

// GetConduitToken prepares a conduit token for image generation.
func (c *ChatGPTBackendAPI) GetConduitToken(ctx context.Context, prompt string, model string, reqs *ChatRequirements) (string, error) {
	path := "/backend-api/f/conversation/prepare"
	payload := map[string]any{
		"action":                "next",
		"fork_from_shared_post": false,
		"parent_message_id":     uuid.NewString(),
		"model":                 c.imageModelSlug(model),
		"client_prepare_state":  "success",
		"timezone_offset_min":   -480,
		"timezone":              "Asia/Shanghai",
		"conversation_mode":     map[string]any{"kind": "primary_assistant"},
		"system_hints":          []string{"picture_v2"},
		"partial_query": map[string]any{
			"id":      uuid.NewString(),
			"author":  map[string]any{"role": "user"},
			"content": map[string]any{"content_type": "text", "parts": []string{prompt}},
		},
		"supports_buffering":     true,
		"supported_encodings":    []string{"v1"},
		"client_contextual_info": map[string]any{"app_name": "chatgpt.com"},
	}

	body, _ := json.Marshal(payload)

	r := c.httpClient.R().SetContext(ctx)
	c.applyHeaders(r, c.imageHeaders(path, reqs, "", "application/json"))
	r.SetBodyBytes(body)

	resp, err := r.Post(c.baseURL + path)
	if err != nil {
		return "", fmt.Errorf("conduit fetch: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode == 401 {
		slog.Error("conduit_401", "status", resp.StatusCode, "body", truncateBytes(respBody, 500), "sentinel_token_len", len(reqs.Token), "auth_header_present", c.accessToken != "")
		return "", ErrInvalidAccessToken
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("conduit failed: HTTP %d: %s", resp.StatusCode, truncateBytes(respBody, 300))
	}

	var result map[string]any
	if err := json.Unmarshal(respBody, &result); err != nil {
		return "", fmt.Errorf("conduit json: %w", err)
	}
	return mapString(result, "conduit_token"), nil
}

// ---------------------------------------------------------------------------
// 4. StartImageGeneration — POST /backend-api/f/conversation (SSE)
// ---------------------------------------------------------------------------

// StartImageGeneration starts the SSE image generation request and returns the
// raw HTTP response (caller must parse the SSE stream and close the body).
func (c *ChatGPTBackendAPI) StartImageGeneration(ctx context.Context, prompt string, reqs *ChatRequirements, conduitToken string, model string) (*http.Response, error) {
	path := "/backend-api/f/conversation"

	payload := map[string]any{
		"action": "next",
		"messages": []any{
			map[string]any{
				"id":          uuid.NewString(),
				"author":      map[string]any{"role": "user"},
				"create_time": float64(time.Now().Unix()),
				"content":     map[string]any{"content_type": "text", "parts": []string{prompt}},
				"metadata": map[string]any{
					"developer_mode_connector_ids": []any{},
					"selected_github_repos":        []any{},
					"selected_all_github_repos":    false,
					"system_hints":                 []string{"picture_v2"},
					"serialization_metadata":       map[string]any{"custom_symbol_offsets": []any{}},
				},
			},
		},
		"parent_message_id":        uuid.NewString(),
		"model":                    c.imageModelSlug(model),
		"client_prepare_state":     "sent",
		"timezone_offset_min":      -480,
		"timezone":                 "Asia/Shanghai",
		"conversation_mode":        map[string]any{"kind": "primary_assistant"},
		"enable_message_followups": true,
		"system_hints":             []string{"picture_v2"},
		"supports_buffering":       true,
		"supported_encodings":      []string{"v1"},
		"client_contextual_info": map[string]any{
			"is_dark_mode":      false,
			"time_since_loaded": 1200,
			"page_height":       1072,
			"page_width":        1724,
			"pixel_ratio":       1.2,
			"screen_height":     1440,
			"screen_width":      2560,
			"app_name":          "chatgpt.com",
		},
		"paragen_cot_summary_display_override": "allow",
		"force_parallel_switch":                "auto",
	}

	body, _ := json.Marshal(payload)

	r := c.httpClient.R().SetContext(ctx)
	c.applyHeaders(r, c.imageHeaders(path, reqs, conduitToken, "text/event-stream"))
	r.SetBodyBytes(body)

	resp, err := r.Post(c.baseURL + path)
	if err != nil {
		return nil, fmt.Errorf("image gen fetch: %w", err)
	}
	if resp.StatusCode == 401 {
		_ = resp.Body.Close()
		return nil, ErrInvalidAccessToken
	}
	if resp.StatusCode >= 400 {
		respBody, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		_ = resp.Body.Close()
		return nil, fmt.Errorf("image gen failed: HTTP %d: %s", resp.StatusCode, truncateBytes(respBody, 300))
	}
	return resp.Response, nil
}

// ---------------------------------------------------------------------------
// 5. ParseSSEImageResult — parse SSE stream for conversation_id + file_ids
// ---------------------------------------------------------------------------

// ParseSSEImageResult reads the SSE stream from the image generation response
// and extracts the conversation ID and file IDs (from file-service:// and sediment:// pointers).
func (c *ChatGPTBackendAPI) ParseSSEImageResult(resp *http.Response) (conversationID string, fileIDs []string, sedimentIDs []string, err error) {
	defer func() { _ = resp.Body.Close() }()

	filePat := regexp.MustCompile(`file-service://([A-Za-z0-9_-]+)`)
	sedPat := regexp.MustCompile(`sediment://([A-Za-z0-9_-]+)`)
	convPat := regexp.MustCompile(`"conversation_id"\s*:\s*"([^"]+)"`)

	seen := make(map[string]bool)
	sedSeen := make(map[string]bool)

	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 64*1024), 8<<20)

	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" || payload == "[DONE]" {
			continue
		}

		// Try to extract conversation_id
		if conversationID == "" {
			if m := convPat.FindStringSubmatch(payload); len(m) > 1 {
				conversationID = m[1]
			}
		}

		// Parse JSON event
		var event map[string]any
		if jsonErr := json.Unmarshal([]byte(payload), &event); jsonErr != nil {
			continue
		}

		// Check if this is an image tool event
		if c.isImageToolEvent(event) {
			// Extract file-service:// IDs
			for _, hit := range filePat.FindAllString(payload, -1) {
				id := strings.TrimPrefix(hit, "file-service://")
				if id != "" && !seen[id] && id != "file_upload" {
					seen[id] = true
					fileIDs = append(fileIDs, id)
				}
			}
			// Extract sediment:// IDs
			for _, hit := range sedPat.FindAllString(payload, -1) {
				id := strings.TrimPrefix(hit, "sediment://")
				if id != "" && !sedSeen[id] {
					sedSeen[id] = true
					sedimentIDs = append(sedimentIDs, id)
				}
			}
		}

		// Also check conversation_id from event fields
		if cid, ok := event["conversation_id"].(string); ok && cid != "" && conversationID == "" {
			conversationID = cid
		}
		if v, ok := event["v"].(map[string]any); ok {
			if cid, ok := v["conversation_id"].(string); ok && cid != "" && conversationID == "" {
				conversationID = cid
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return conversationID, fileIDs, sedimentIDs, fmt.Errorf("SSE scan error: %w", err)
	}
	return conversationID, fileIDs, sedimentIDs, nil
}

// isImageToolEvent checks if the SSE event represents an image generation tool output.
func (c *ChatGPTBackendAPI) isImageToolEvent(event map[string]any) bool {
	// Direct event or nested in "v"
	for _, candidate := range []any{event, event["v"]} {
		m, ok := candidate.(map[string]any)
		if !ok {
			continue
		}
		message, _ := m["message"].(map[string]any)
		if message == nil {
			continue
		}
		author, _ := message["author"].(map[string]any)
		metadata, _ := message["metadata"].(map[string]any)
		content, _ := message["content"].(map[string]any)

		if author != nil && fmt.Sprint(author["role"]) == "tool" {
			if metadata != nil && fmt.Sprint(metadata["async_task_type"]) == "image_gen" {
				return true
			}
			if content != nil && fmt.Sprint(content["content_type"]) == "multimodal_text" {
				return true
			}
		}
	}
	return false
}

// ---------------------------------------------------------------------------
// 6. PollImageResult — poll conversation for file IDs when SSE didn't return them
// ---------------------------------------------------------------------------

// PollImageResult polls the conversation endpoint until image file IDs appear
// or the timeout is reached. Returns (fileIDs, sedimentIDs, error).
func (c *ChatGPTBackendAPI) PollImageResult(ctx context.Context, conversationID string, timeout time.Duration) ([]string, []string, error) {
	if timeout <= 0 {
		timeout = defaultImagePollTimeout
	}

	deadline := time.Now().Add(timeout)
	attempt := 0

	// Initial wait before first poll
	select {
	case <-ctx.Done():
		return nil, nil, ctx.Err()
	case <-time.After(defaultImagePollInitial + time.Duration(rand.Int63n(int64(2*time.Second)))):
	}

	for time.Now().Before(deadline) {
		attempt++
		fileIDs, sedimentIDs, err := c.pollOnce(ctx, conversationID)
		if err != nil {
			// Retry on transient errors
			backoff := time.Duration(1<<uint(minInt(attempt, 4))) * time.Second
			if backoff > 16*time.Second {
				backoff = 16 * time.Second
			}
			select {
			case <-ctx.Done():
				return nil, nil, ctx.Err()
			case <-time.After(backoff):
				continue
			}
		}
		if len(fileIDs) > 0 {
			return fileIDs, nil, nil
		}
		if len(sedimentIDs) > 0 {
			return nil, sedimentIDs, nil
		}
		// No results yet, wait before next poll
		wait := defaultImagePollInterval
		if remaining := time.Until(deadline); remaining < wait {
			wait = remaining
		}
		if wait <= 0 {
			break
		}
		select {
		case <-ctx.Done():
			return nil, nil, ctx.Err()
		case <-time.After(wait):
		}
	}
	return nil, nil, ErrImagePollTimeout
}

func (c *ChatGPTBackendAPI) pollOnce(ctx context.Context, conversationID string) ([]string, []string, error) {
	path := fmt.Sprintf("/backend-api/conversation/%s", conversationID)

	r := c.httpClient.R().SetContext(ctx)
	c.applyHeaders(r, c.apiHeaders(path, map[string]string{"Accept": "application/json"}))

	resp, err := r.Get(c.baseURL + path)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == 429 || resp.StatusCode >= 500 {
		return nil, nil, fmt.Errorf("poll conversation: HTTP %d", resp.StatusCode)
	}
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
		return nil, nil, fmt.Errorf("poll conversation: HTTP %d: %s", resp.StatusCode, truncateBytes(body, 200))
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, nil, err
	}
	return c.extractImageToolRecords(body)
}

// extractImageToolRecords extracts file IDs and sediment IDs from a conversation document.
func (c *ChatGPTBackendAPI) extractImageToolRecords(body []byte) ([]string, []string, error) {
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, nil, fmt.Errorf("conversation json: %w", err)
	}

	filePat := regexp.MustCompile(`file-service://([A-Za-z0-9_-]+)`)
	sedPat := regexp.MustCompile(`sediment://([A-Za-z0-9_-]+)`)

	var fileIDs, sedimentIDs []string
	seenFile := make(map[string]bool)
	seenSed := make(map[string]bool)

	mapping, _ := data["mapping"].(map[string]any)
	for _, node := range mapping {
		nodeMap, ok := node.(map[string]any)
		if !ok {
			continue
		}
		message, _ := nodeMap["message"].(map[string]any)
		if message == nil {
			continue
		}
		author, _ := message["author"].(map[string]any)
		metadata, _ := message["metadata"].(map[string]any)
		content, _ := message["content"].(map[string]any)

		if author == nil || fmt.Sprint(author["role"]) != "tool" {
			continue
		}
		if content == nil || fmt.Sprint(content["content_type"]) != "multimodal_text" {
			continue
		}

		isImageGen := metadata != nil && fmt.Sprint(metadata["async_task_type"]) == "image_gen"

		parts, _ := content["parts"].([]any)
		for _, part := range parts {
			var text string
			if partMap, ok := part.(map[string]any); ok {
				text = fmt.Sprint(partMap["asset_pointer"])
			} else if s, ok := part.(string); ok {
				text = s
			}
			for _, hit := range filePat.FindAllString(text, -1) {
				id := strings.TrimPrefix(hit, "file-service://")
				if id != "" && !seenFile[id] && id != "file_upload" {
					seenFile[id] = true
					fileIDs = append(fileIDs, id)
				}
			}
			for _, hit := range sedPat.FindAllString(text, -1) {
				id := strings.TrimPrefix(hit, "sediment://")
				if id != "" && !seenSed[id] {
					seenSed[id] = true
					sedimentIDs = append(sedimentIDs, id)
				}
			}
		}
		_ = isImageGen // used for filtering above, kept for clarity
	}

	return fileIDs, sedimentIDs, nil
}

// ---------------------------------------------------------------------------
// 7. GetFileDownloadURL — GET /backend-api/files/{fileID}/download
// ---------------------------------------------------------------------------

// GetFileDownloadURL returns the download URL for a given file ID.
func (c *ChatGPTBackendAPI) GetFileDownloadURL(ctx context.Context, fileID string) (string, error) {
	path := fmt.Sprintf("/backend-api/files/%s/download", fileID)

	r := c.httpClient.R().SetContext(ctx)
	c.applyHeaders(r, c.apiHeaders(path, map[string]string{"Accept": "application/json"}))

	resp, err := r.Get(c.baseURL + path)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode == 401 {
		return "", ErrInvalidAccessToken
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("file download URL failed: HTTP %d: %s", resp.StatusCode, truncateBytes(body, 200))
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	if u := mapString(result, "download_url"); u != "" {
		return u, nil
	}
	return mapString(result, "url"), nil
}

// ---------------------------------------------------------------------------
// 8. GetAttachmentDownloadURL — GET /backend-api/conversation/{id}/attachment/{id}/download
// ---------------------------------------------------------------------------

// GetAttachmentDownloadURL returns the download URL for a conversation attachment (sediment).
func (c *ChatGPTBackendAPI) GetAttachmentDownloadURL(ctx context.Context, conversationID string, attachmentID string) (string, error) {
	path := fmt.Sprintf("/backend-api/conversation/%s/attachment/%s/download", conversationID, attachmentID)

	r := c.httpClient.R().SetContext(ctx)
	c.applyHeaders(r, c.apiHeaders(path, map[string]string{"Accept": "application/json"}))

	resp, err := r.Get(c.baseURL + path)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode == 401 {
		return "", ErrInvalidAccessToken
	}
	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("attachment download URL failed: HTTP %d: %s", resp.StatusCode, truncateBytes(body, 200))
	}

	var result map[string]any
	if err := json.Unmarshal(body, &result); err != nil {
		return "", err
	}
	if u := mapString(result, "download_url"); u != "" {
		return u, nil
	}
	return mapString(result, "url"), nil
}

// ---------------------------------------------------------------------------
// 9. DownloadImage — GET download URL, return bytes
// ---------------------------------------------------------------------------

// DownloadImage downloads image bytes from a given URL.
func (c *ChatGPTBackendAPI) DownloadImage(ctx context.Context, imageURL string) ([]byte, error) {
	r := c.httpClient.R().SetContext(ctx)
	r.SetHeader("User-Agent", c.userAgent)

	resp, err := r.Get(imageURL)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		urlShort := imageURL
		if len(urlShort) > 200 {
			urlShort = urlShort[:200]
		}
		slog.Warn("chatgpt_image_download_failed", "status", resp.StatusCode, "url", urlShort)
		return nil, fmt.Errorf("image download failed: HTTP %d", resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 20<<20))
	if err != nil {
		return nil, fmt.Errorf("image download read: %w", err)
	}
	return data, nil
}

// ---------------------------------------------------------------------------
// 10. ResolveImageURLs — orchestrate file ID → download URL resolution
// ---------------------------------------------------------------------------

// ResolveImageURLs converts file IDs and sediment IDs to downloadable URLs.
// It first tries file IDs via GetFileDownloadURL, and falls back to sediment IDs
// via GetAttachmentDownloadURL.
func (c *ChatGPTBackendAPI) ResolveImageURLs(ctx context.Context, conversationID string, fileIDs []string, sedimentIDs []string) ([]string, error) {
	var urls []string

	for _, fileID := range fileIDs {
		if fileID == "file_upload" {
			continue
		}
		u, err := c.GetFileDownloadURL(ctx, fileID)
		if err != nil {
			slog.Warn("chatgpt_file_download_url_failed", "file_id", fileID, "error", err)
			continue // skip failed ones
		}
		if u != "" {
			slog.Info("chatgpt_file_download_url_resolved", "file_id", fileID, "url_prefix", func() string {
				if len(u) > 100 {
					return u[:100]
				}
				return u
			}())
			urls = append(urls, u)
		}
	}

	if len(urls) > 0 || conversationID == "" {
		return urls, nil
	}

	// Fallback: try sediment IDs
	for _, sedID := range sedimentIDs {
		u, err := c.GetAttachmentDownloadURL(ctx, conversationID, sedID)
		if err != nil {
			continue
		}
		if u != "" {
			urls = append(urls, u)
		}
	}
	return urls, nil
}

// ---------------------------------------------------------------------------
// Helper: image headers with sentinel/conduit tokens
// ---------------------------------------------------------------------------

func (c *ChatGPTBackendAPI) imageHeaders(path string, reqs *ChatRequirements, conduitToken string, accept string) map[string]string {
	extra := map[string]string{
		"Content-Type":          "application/json",
		"Accept":                accept,
		"Authorization":         "Bearer " + c.accessToken,
		"X-OpenAI-Target-Path":  path,
		"X-OpenAI-Target-Route": path,
	}
	if reqs != nil {
		if reqs.Token != "" {
			extra["OpenAI-Sentinel-Chat-Requirements-Token"] = reqs.Token
		}
		if reqs.ProofToken != "" {
			extra["OpenAI-Sentinel-Proof-Token"] = reqs.ProofToken
		}
		if reqs.TurnstileToken != "" {
			extra["OpenAI-Sentinel-Turnstile-Token"] = reqs.TurnstileToken
		}
		if reqs.SOToken != "" {
			extra["OpenAI-Sentinel-SO-Token"] = reqs.SOToken
		}
	}
	if conduitToken != "" {
		extra["X-Conduit-Token"] = conduitToken
	}
	if accept == "text/event-stream" {
		extra["X-Oai-Turn-Trace-Id"] = uuid.NewString()
	}
	return c.apiHeaders(path, extra)
}

// imageModelSlug maps public model names to internal ChatGPT model slugs.
func (c *ChatGPTBackendAPI) imageModelSlug(model string) string {
	model = strings.TrimSpace(model)
	if model == "" {
		return "auto"
	}
	if model == "gpt-image-2" {
		return "gpt-5-3"
	}
	return "auto"
}

// ---------------------------------------------------------------------------
// PoW helpers (ported from Python pow.py)
// ---------------------------------------------------------------------------

var (
	backendPowScriptSrcRE = regexp.MustCompile(`<script[^>]+src="([^"]+)"`)
	backendPowDataBuildRE = regexp.MustCompile(`<html[^>]*data-build="([^"]*)"`)
	backendPowCBuildRE    = regexp.MustCompile(`c/[^/]*/_`)
)

func parseChatGPTBackendPowResources(html string) ([]string, string) {
	matches := backendPowScriptSrcRE.FindAllStringSubmatch(html, -1)
	sources := make([]string, 0, len(matches))
	dataBuild := ""
	for _, m := range matches {
		if len(m) < 2 || strings.TrimSpace(m[1]) == "" {
			continue
		}
		src := m[1]
		sources = append(sources, src)
		if dataBuild == "" {
			dataBuild = backendPowCBuildRE.FindString(src)
		}
	}
	if dataBuild == "" {
		if m := backendPowDataBuildRE.FindStringSubmatch(html); len(m) > 1 {
			dataBuild = m[1]
		}
	}
	return sources, dataBuild
}

// ---------------------------------------------------------------------------
// Utility functions
// ---------------------------------------------------------------------------

func mapString(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	v, ok := m[key]
	if !ok || v == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(v))
}

func truncateBytes(b []byte, maxLen int) string {
	s := string(b)
	if len(s) > maxLen {
		return s[:maxLen] + "…[truncated]"
	}
	return s
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
