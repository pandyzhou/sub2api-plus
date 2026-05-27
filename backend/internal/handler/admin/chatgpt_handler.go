package admin

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	"github.com/Wei-Shaw/sub2api/internal/service"
	"github.com/gin-gonic/gin"
)

// ChatGPTAccountHandler manages ChatGPT Web OAuth accounts as a pooled resource.
type ChatGPTAccountHandler struct {
	poolSvc     *service.ChatGPTAccountPoolService
	registerSvc *service.ChatGPTRegisterService
}

// NewChatGPTAccountHandler creates a new ChatGPT account handler.
func NewChatGPTAccountHandler(accountRepo service.AccountRepository, settingRepo service.SettingRepository, registerSvc *service.ChatGPTRegisterService) *ChatGPTAccountHandler {
	return &ChatGPTAccountHandler{poolSvc: service.NewChatGPTAccountPoolService(accountRepo, settingRepo), registerSvc: registerSvc}
}

// ListAccounts returns all ChatGPT Web accounts.
// GET /api/v1/admin/chatgpt/accounts
func (h *ChatGPTAccountHandler) ListAccounts(c *gin.Context) {
	items, err := h.poolSvc.ListAccounts(c.Request.Context())
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, gin.H{"items": items})
}

// CreateAccounts adds one or more ChatGPT Web accounts.
// POST /api/v1/admin/chatgpt/accounts
func (h *ChatGPTAccountHandler) CreateAccounts(c *gin.Context) {
	var body map[string]any
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	tokens := stringListFromAny(body["tokens"])
	accounts := accountPayloadsFromAny(body["accounts"])
	if len(tokens) == 0 && len(accounts) == 0 {
		response.BadRequest(c, "tokens is required")
		return
	}
	result, err := h.poolSvc.CreateAccounts(c.Request.Context(), tokens, accounts)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

// DeleteAccounts removes accounts by token list.
// DELETE /api/v1/admin/chatgpt/accounts
func (h *ChatGPTAccountHandler) DeleteAccounts(c *gin.Context) {
	var body struct {
		Tokens []string `json:"tokens"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	if len(body.Tokens) == 0 {
		response.BadRequest(c, "tokens is required")
		return
	}
	result, err := h.poolSvc.DeleteAccounts(c.Request.Context(), body.Tokens)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

// RefreshAccounts refreshes ChatGPT Web user info for the given tokens or all accounts.
// POST /api/v1/admin/chatgpt/accounts/refresh
func (h *ChatGPTAccountHandler) RefreshAccounts(c *gin.Context) {
	var body struct {
		Tokens       []string `json:"access_tokens"`
		LegacyTokens []string `json:"tokens"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	tokens := body.Tokens
	if len(tokens) == 0 {
		tokens = body.LegacyTokens
	}
	result, err := h.poolSvc.RefreshAccounts(c.Request.Context(), tokens)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, result)
}

// UpdateAccount updates status/type/quota/image_quota_unknown.
// POST /api/v1/admin/chatgpt/accounts/update
func (h *ChatGPTAccountHandler) UpdateAccount(c *gin.Context) {
	var body map[string]any
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	accessToken := strings.TrimSpace(stringFromJSONAny(body["access_token"]))
	if accessToken == "" {
		response.BadRequest(c, "access_token is required")
		return
	}
	updates := map[string]any{}
	for _, key := range []string{"type", "status", "quota", "image_quota_unknown"} {
		if v, ok := body[key]; ok {
			updates[key] = v
		}
	}
	if len(updates) == 0 {
		response.BadRequest(c, "还没有检测到改动，请修改后再保存")
		return
	}
	item, err := h.poolSvc.UpdateAccount(c.Request.Context(), accessToken, updates)
	if err != nil {
		if err == service.ErrAccountNotFound {
			response.NotFound(c, "account not found")
			return
		}
		response.InternalError(c, err.Error())
		return
	}
	items, _ := h.poolSvc.ListAccounts(c.Request.Context())
	response.Success(c, gin.H{"item": item, "items": items})
}

// ExportAccounts exports accounts as JSON or ZIP.
// POST /api/v1/admin/chatgpt/accounts/export
func (h *ChatGPTAccountHandler) ExportAccounts(c *gin.Context) {
	var body struct {
		Tokens []string `json:"access_tokens"`
		Format string   `json:"format"`
	}
	_ = c.ShouldBindJSON(&body)
	format := strings.ToLower(strings.TrimSpace(body.Format))
	if format == "" {
		format = "json"
	}
	if format != "json" && format != "zip" {
		response.BadRequest(c, "format must be json or zip")
		return
	}
	items, err := h.poolSvc.BuildExportItems(c.Request.Context(), body.Tokens)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	if len(items) == 0 {
		response.BadRequest(c, "没有可导出的完整账号，需要同时有 access_token、refresh_token 和 id_token")
		return
	}
	var data []byte
	mediaType := "application/json"
	if format == "zip" {
		data, err = service.BuildChatGPTAccountExportZIP(items)
		mediaType = "application/zip"
	} else {
		data, err = service.BuildChatGPTAccountExportJSON(items)
	}
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	c.Header("Content-Disposition", service.ChatGPTAccountExportContentDisposition(format, time.Now()))
	c.Data(http.StatusOK, mediaType, data)
}

// AccountPoolConfig returns ChatGPT account pool configuration.
// GET /api/v1/admin/chatgpt/account-pool/config
func (h *ChatGPTAccountHandler) AccountPoolConfig(c *gin.Context) {
	response.Success(c, h.poolSvc.GetConfig(c.Request.Context()))
}

// UpdateAccountPoolConfig updates ChatGPT account pool configuration.
// POST /api/v1/admin/chatgpt/account-pool/config
func (h *ChatGPTAccountHandler) UpdateAccountPoolConfig(c *gin.Context) {
	var body map[string]any
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	cfg, err := h.poolSvc.UpdateConfig(c.Request.Context(), body)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}
	response.Success(c, cfg)
}

// RegisterConfig returns register machine configuration.
// GET /api/v1/admin/chatgpt/register
func (h *ChatGPTAccountHandler) RegisterConfig(c *gin.Context) {
	if h.registerSvc == nil {
		response.Error(c, 503, "register service not available")
		return
	}
	response.Success(c, h.registerSvc.Get())
}

// UpdateRegisterConfig updates register machine configuration.
// POST /api/v1/admin/chatgpt/register
func (h *ChatGPTAccountHandler) UpdateRegisterConfig(c *gin.Context) {
	if h.registerSvc == nil {
		response.Error(c, 503, "register service not available")
		return
	}
	var body map[string]any
	if err := c.ShouldBindJSON(&body); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}
	response.Success(c, h.registerSvc.Update(body))
}

// StartRegister starts the registration process.
// POST /api/v1/admin/chatgpt/register/start
func (h *ChatGPTAccountHandler) StartRegister(c *gin.Context) {
	if h.registerSvc == nil {
		response.Error(c, 503, "register service not available")
		return
	}
	response.Success(c, h.registerSvc.Start())
}

// StopRegister stops the registration process.
// POST /api/v1/admin/chatgpt/register/stop
func (h *ChatGPTAccountHandler) StopRegister(c *gin.Context) {
	if h.registerSvc == nil {
		response.Error(c, 503, "register service not available")
		return
	}
	response.Success(c, h.registerSvc.Stop())
}

// ResetRegister resets the registration stats.
// POST /api/v1/admin/chatgpt/register/reset
func (h *ChatGPTAccountHandler) ResetRegister(c *gin.Context) {
	if h.registerSvc == nil {
		response.Error(c, 503, "register service not available")
		return
	}
	response.Success(c, h.registerSvc.Reset())
}

// CreateRegisterEventsToken creates a short-lived token for EventSource clients.
// POST /api/v1/admin/chatgpt/register/events-token
func (h *ChatGPTAccountHandler) CreateRegisterEventsToken(c *gin.Context) {
	token, expiresAt := newChatGPTRegisterEventToken(5 * time.Minute)
	response.Success(c, gin.H{"token": token, "expires_at": expiresAt.UTC().Format(time.RFC3339), "ttl_seconds": 300})
}

// RegisterEvents streams register status as SSE using a short-lived query token.
// GET /api/v1/admin/chatgpt/register/events?token=xxx
func (h *ChatGPTAccountHandler) RegisterEvents(c *gin.Context) {
	if h.registerSvc == nil {
		response.Error(c, 503, "register service not available")
		return
	}
	if !validateChatGPTRegisterEventToken(c.Query("token"), time.Now()) {
		response.Unauthorized(c, "invalid or expired event token")
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	flusher, _ := c.Writer.(http.Flusher)
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-ticker.C:
			payload, _ := json.Marshal(h.registerSvc.Get())
			_, _ = c.Writer.Write([]byte("data: "))
			_, _ = c.Writer.Write(payload)
			_, _ = c.Writer.Write([]byte("\n\n"))
			if flusher != nil {
				flusher.Flush()
			}
		}
	}
}

func stringListFromAny(value any) []string {
	arr, ok := value.([]any)
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	seen := map[string]struct{}{}
	for _, item := range arr {
		s := strings.TrimSpace(stringFromJSONAny(item))
		if s == "" {
			continue
		}
		if _, ok := seen[s]; ok {
			continue
		}
		seen[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func accountPayloadsFromAny(value any) []map[string]any {
	switch v := value.(type) {
	case []any:
		out := make([]map[string]any, 0, len(v))
		for _, item := range v {
			if m, ok := item.(map[string]any); ok {
				out = append(out, m)
			}
		}
		return out
	case map[string]any:
		return []map[string]any{v}
	default:
		return nil
	}
}

func stringFromJSONAny(value any) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return strings.TrimSpace(jsonNumberSprint(value))
}

func jsonNumberSprint(value any) string {
	return strings.TrimSpace(strings.Trim(strings.ReplaceAll(strings.ReplaceAll(strings.TrimSpace(toJSONString(value)), "\n", ""), "\t", ""), "\""))
}

func toJSONString(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		return ""
	}
	return string(data)
}

type chatGPTRegisterEventTokenRecord struct{ expiresAt time.Time }

var chatGPTRegisterEventTokens sync.Map

func newChatGPTRegisterEventToken(ttl time.Duration) (string, time.Time) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		buf = []byte(time.Now().Format(time.RFC3339Nano))
	}
	token := base64.RawURLEncoding.EncodeToString(buf)
	expiresAt := time.Now().Add(ttl)
	chatGPTRegisterEventTokens.Store(token, chatGPTRegisterEventTokenRecord{expiresAt: expiresAt})
	return token, expiresAt
}

func validateChatGPTRegisterEventToken(token string, now time.Time) bool {
	token = strings.TrimSpace(token)
	if token == "" {
		return false
	}
	raw, ok := chatGPTRegisterEventTokens.Load(token)
	if !ok {
		return false
	}
	rec, ok := raw.(chatGPTRegisterEventTokenRecord)
	if !ok || now.After(rec.expiresAt) {
		chatGPTRegisterEventTokens.Delete(token)
		return false
	}
	return true
}
