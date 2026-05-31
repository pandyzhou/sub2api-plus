package handler

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/response"
	middleware2 "github.com/Wei-Shaw/sub2api/internal/server/middleware"
	"github.com/Wei-Shaw/sub2api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UserImageHandler 用户端图片生成代理
type UserImageHandler struct {
	gatewaySvc      *service.OpenAIGatewayService
	apiKeySvc       *service.APIKeyService
	imageSessionSvc *service.ImageSessionService
	chatgptImageSvc *service.ChatGPTImageService
}

// NewUserImageHandler 创建用户图片 handler
func NewUserImageHandler(
	gatewaySvc *service.OpenAIGatewayService,
	apiKeySvc *service.APIKeyService,
	imageSessionSvc *service.ImageSessionService,
	chatgptImageSvc *service.ChatGPTImageService,
) *UserImageHandler {
	return &UserImageHandler{
		gatewaySvc:      gatewaySvc,
		apiKeySvc:       apiKeySvc,
		imageSessionSvc: imageSessionSvc,
		chatgptImageSvc: chatgptImageSvc,
	}
}

// Generate 处理 POST /api/v1/user/image/generate
func (h *UserImageHandler) Generate(c *gin.Context) {
	// 1. 认证
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "用户未认证")
		return
	}

	// 2. 解析请求
	var req struct {
		Prompt    string  `json:"prompt"`
		Model     string  `json:"model"`
		N         int     `json:"n"`
		Size      string  `json:"size"`
		SessionID *string `json:"session_id"`
		Title     *string `json:"title"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}
	if req.Prompt == "" {
		response.BadRequest(c, "prompt 不能为空")
		return
	}

	// 3. 调用 ChatGPTImageService
	output, err := h.chatgptImageSvc.Generate(c.Request.Context(), service.ChatGPTImageGenerateInput{
		Prompt: req.Prompt,
		Model:  req.Model,
		N:      req.N,
		Size:   req.Size,
	})
	if err != nil {
		response.InternalError(c, "图片生成失败: "+err.Error())
		return
	}

	// 4. 记录会话
	if req.SessionID != nil && *req.SessionID != "" && h.imageSessionSvc != nil {
		go h.recordToSession(subject.UserID, *req.SessionID, &service.OpenAIImagesRequest{
			Prompt: req.Prompt,
			Model:  req.Model,
		})
	} else if req.Title != nil && *req.Title != "" && h.imageSessionSvc != nil {
		go h.autoCreateAndRecord(subject.UserID, *req.Title, &service.OpenAIImagesRequest{
			Prompt: req.Prompt,
			Model:  req.Model,
		})
	}

	// 5. 返回结果（OpenAI 兼容格式）
	c.JSON(http.StatusOK, output)
}

// Edit 处理 POST /api/v1/user/image/edit
func (h *UserImageHandler) Edit(c *gin.Context) {
	response.BadRequest(c, "图片编辑功能暂不支持，请使用文生图功能")
}

// ───────────────────── 会话管理 endpoints ─────────────────────

// recordToSession 记录内部图片生成到会话
func (h *UserImageHandler) recordToSession(userID int64, sessionID string, parsed *service.OpenAIImagesRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if h.imageSessionSvc == nil || parsed == nil {
		return
	}
	record := service.ImageRecord{
		ID:     fmt.Sprintf("img_rec_%d_%s", time.Now().UnixMilli(), uuid.NewString()[:8]),
		Prompt: parsed.Prompt,
		Model:  parsed.Model,
	}
	_ = h.imageSessionSvc.AddRecord(ctx, userID, sessionID, record)
}

// autoCreateAndRecord 内部图片生成自动创建并记录会话
func (h *UserImageHandler) autoCreateAndRecord(userID int64, title string, parsed *service.OpenAIImagesRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if h.imageSessionSvc == nil || parsed == nil {
		return
	}
	session, err := h.imageSessionSvc.CreateSession(ctx, userID, title)
	if err != nil {
		return
	}
	record := service.ImageRecord{
		ID:     fmt.Sprintf("img_rec_%d_%s", time.Now().UnixMilli(), uuid.NewString()[:8]),
		Prompt: parsed.Prompt,
		Model:  parsed.Model,
	}
	_ = h.imageSessionSvc.AddRecord(ctx, userID, session.ID, record)
}

// ListSessions GET /api/v1/user/image/sessions
func (h *UserImageHandler) ListSessions(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "用户未认证")
		return
	}
	if h.imageSessionSvc == nil {
		response.Success(c, []any{})
		return
	}
	sessions, err := h.imageSessionSvc.ListSessions(c.Request.Context(), subject.UserID)
	if err != nil {
		response.InternalError(c, "获取会话列表失败: "+err.Error())
		return
	}
	response.Success(c, sessions)
}

// GetSession GET /api/v1/user/image/sessions/:id
func (h *UserImageHandler) GetSession(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "用户未认证")
		return
	}
	sessionID := c.Param("id")
	if sessionID == "" {
		response.BadRequest(c, "会话 ID 不能为空")
		return
	}
	if h.imageSessionSvc == nil {
		response.NotFound(c, "会话服务未初始化")
		return
	}
	session, err := h.imageSessionSvc.GetSession(c.Request.Context(), subject.UserID, sessionID)
	if err != nil {
		response.NotFound(c, "会话不存在")
		return
	}
	response.Success(c, session)
}

// CreateSession POST /api/v1/user/image/sessions
func (h *UserImageHandler) CreateSession(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "用户未认证")
		return
	}
	if h.imageSessionSvc == nil {
		response.InternalError(c, "会话服务未初始化")
		return
	}

	var req struct {
		Title string `json:"title"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "请求参数错误")
		return
	}
	if strings.TrimSpace(req.Title) == "" {
		req.Title = "未命名会话"
	}

	session, err := h.imageSessionSvc.CreateSession(c.Request.Context(), subject.UserID, req.Title)
	if err != nil {
		response.InternalError(c, "创建会话失败: "+err.Error())
		return
	}
	response.Created(c, session)
}

// DeleteSession DELETE /api/v1/user/image/sessions/:id
func (h *UserImageHandler) DeleteSession(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "用户未认证")
		return
	}
	sessionID := c.Param("id")
	if sessionID == "" {
		response.BadRequest(c, "会话 ID 不能为空")
		return
	}
	if h.imageSessionSvc == nil {
		response.NotFound(c, "会话服务未初始化")
		return
	}
	if err := h.imageSessionSvc.DeleteSession(c.Request.Context(), subject.UserID, sessionID); err != nil {
		response.NotFound(c, "会话不存在")
		return
	}
	response.Success(c, gin.H{"message": "会话已删除"})
}

// ClearSessions DELETE /api/v1/user/image/sessions
func (h *UserImageHandler) ClearSessions(c *gin.Context) {
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "用户未认证")
		return
	}
	if h.imageSessionSvc == nil {
		response.Success(c, gin.H{"message": "会话服务未初始化"})
		return
	}
	if err := h.imageSessionSvc.ClearSessions(c.Request.Context(), subject.UserID); err != nil {
		response.InternalError(c, "清空会话失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"message": "所有会话已清空"})
}
