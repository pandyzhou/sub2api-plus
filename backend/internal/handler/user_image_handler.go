package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	pkghttputil "github.com/Wei-Shaw/sub2api/internal/pkg/httputil"
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
}

// NewUserImageHandler 创建用户图片 handler
func NewUserImageHandler(
	gatewaySvc *service.OpenAIGatewayService,
	apiKeySvc *service.APIKeyService,
	imageSessionSvc *service.ImageSessionService,
) *UserImageHandler {
	return &UserImageHandler{
		gatewaySvc:      gatewaySvc,
		apiKeySvc:       apiKeySvc,
		imageSessionSvc: imageSessionSvc,
	}
}

// userImageGenerateRequest 图片生成请求（扩展字段，与 OpenAI 请求体合并解析）
type userImageGenerateRequest struct {
	GroupID   *int64  `json:"group_id"`
	SessionID *string `json:"session_id"`
	Title     *string `json:"title"`
}

// Generate 处理 POST /api/v1/user/image/generate
func (h *UserImageHandler) Generate(c *gin.Context) {
	// 1. 从 JWT 获取用户信息
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "用户未认证")
		return
	}

	// 2. 读取请求体
	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		response.BadRequest(c, "读取请求体失败")
		return
	}
	if len(body) == 0 {
		response.BadRequest(c, "请求体不能为空")
		return
	}

	// 3. 解析扩展字段（group_id, session_id 等，这些字段会被 ParseOpenAIImagesRequest 忽略）
	var extReq userImageGenerateRequest
	_ = json.Unmarshal(body, &extReq)

	// 4. 获取 groupID
	groupID, err := h.resolveGroupID(c, subject.UserID, extReq.GroupID)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// 5. 解析 OpenAI 图片请求
	// 将用户端路径重写为标准 OpenAI 路径，以便 ParseOpenAIImagesRequest 正确解析
	c.Request.URL.Path = "/v1/images/generations"
	parsed, err := h.gatewaySvc.ParseOpenAIImagesRequest(c, body)
	if err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	// 6. 选择可用账号（优先 ChatGPT Web 模式，无可用时回退到任意模式）
	sessionHash := uuid.NewString()
	selection, _, err := h.gatewaySvc.SelectAccountWithSchedulerForImagesBackendMode(
		c.Request.Context(),
		groupID,
		sessionHash,
		parsed.Model,
		nil,
		parsed.RequiredCapability,
		service.OpenAIBackendModeChatGPTWeb,
	)
	if err != nil {
		// ChatGPT Web 模式无可用账号，回退到任意模式
		selection, _, err = h.gatewaySvc.SelectAccountWithSchedulerForImagesBackendMode(
			c.Request.Context(),
			groupID,
			sessionHash,
			parsed.Model,
			nil,
			parsed.RequiredCapability,
			service.OpenAIBackendModeAny,
		)
	}
	if err != nil {
		response.InternalError(c, "没有可用的图片生成账号: "+err.Error())
		return
	}
	if selection == nil || selection.Account == nil {
		response.InternalError(c, "没有可用的图片生成账号")
		return
	}
	defer func() {
		if selection.ReleaseFunc != nil {
			selection.ReleaseFunc()
		}
	}()

	account := selection.Account

	// 7. 解析渠道模型映射
	channelMapping, _ := h.gatewaySvc.ResolveChannelMappingAndRestrict(c.Request.Context(), groupID, parsed.Model)

	// 8. 转发图片请求（ForwardImages 会直接将响应写入 c.Writer）
	result, err := h.gatewaySvc.ForwardImages(c.Request.Context(), c, account, body, parsed, channelMapping.MappedModel)
	if err != nil {
		_ = c.Error(err)
		return
	}

	// 9. 记录到会话（异步，不阻塞响应）
	if extReq.SessionID != nil && *extReq.SessionID != "" && h.imageSessionSvc != nil {
		go h.recordToSession(subject.UserID, *extReq.SessionID, parsed)
	} else if extReq.Title != nil && *extReq.Title != "" && h.imageSessionSvc != nil {
		go h.autoCreateAndRecord(subject.UserID, *extReq.Title, parsed)
	}

	_ = result
}

// Edit 处理 POST /api/v1/user/image/edit（multipart/form-data）
func (h *UserImageHandler) Edit(c *gin.Context) {
	// 1. 从 JWT 获取用户信息
	subject, ok := middleware2.GetAuthSubjectFromContext(c)
	if !ok {
		response.Unauthorized(c, "用户未认证")
		return
	}

	// 2. 读取 multipart 请求体
	body, err := pkghttputil.ReadRequestBodyWithPrealloc(c.Request)
	if err != nil {
		response.BadRequest(c, "读取请求体失败")
		return
	}
	if len(body) == 0 {
		response.BadRequest(c, "请求体不能为空")
		return
	}

	// 3. 从 form 获取 group_id 和 session_id
	var groupIDPtr *int64
	if gidStr := c.PostForm("group_id"); gidStr != "" {
		var gid int64
		if _, scanErr := fmt.Sscanf(gidStr, "%d", &gid); scanErr == nil && gid > 0 {
			groupIDPtr = &gid
		}
	}
	sessionID := c.PostForm("session_id")

	// 4. 获取 groupID
	groupID, err := h.resolveGroupID(c, subject.UserID, groupIDPtr)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	// 5. 解析 OpenAI 图片请求（multipart）
	// 将用户端路径重写为标准 OpenAI 路径
	c.Request.URL.Path = "/v1/images/edits"
	parsed, err := h.gatewaySvc.ParseOpenAIImagesRequest(c, body)
	if err != nil {
		response.BadRequest(c, "请求参数错误: "+err.Error())
		return
	}

	// 6. 选择可用账号（优先 ChatGPT Web 模式，无可用时回退到任意模式）
	sessionHash := uuid.NewString()
	selection, _, err := h.gatewaySvc.SelectAccountWithSchedulerForImagesBackendMode(
		c.Request.Context(),
		groupID,
		sessionHash,
		parsed.Model,
		nil,
		parsed.RequiredCapability,
		service.OpenAIBackendModeChatGPTWeb,
	)
	if err != nil {
		// ChatGPT Web 模式无可用账号，回退到任意模式
		selection, _, err = h.gatewaySvc.SelectAccountWithSchedulerForImagesBackendMode(
			c.Request.Context(),
			groupID,
			sessionHash,
			parsed.Model,
			nil,
			parsed.RequiredCapability,
			service.OpenAIBackendModeAny,
		)
	}
	if err != nil {
		response.InternalError(c, "没有可用的图片生成账号: "+err.Error())
		return
	}
	if selection == nil || selection.Account == nil {
		response.InternalError(c, "没有可用的图片生成账号")
		return
	}
	defer func() {
		if selection.ReleaseFunc != nil {
			selection.ReleaseFunc()
		}
	}()

	account := selection.Account

	// 7. 解析渠道模型映射
	channelMapping, _ := h.gatewaySvc.ResolveChannelMappingAndRestrict(c.Request.Context(), groupID, parsed.Model)

	// 8. 转发图片请求
	result, err := h.gatewaySvc.ForwardImages(c.Request.Context(), c, account, body, parsed, channelMapping.MappedModel)
	if err != nil {
		_ = c.Error(err)
		return
	}

	// 9. 记录到会话
	if sessionID != "" && h.imageSessionSvc != nil {
		go h.recordToSession(subject.UserID, sessionID, parsed)
	}

	_ = result
}

// ───────────────────── 会话管理 endpoints ─────────────────────

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
		response.Success(c, gin.H{"message": "所有会话已清空"})
		return
	}
	if err := h.imageSessionSvc.ClearSessions(c.Request.Context(), subject.UserID); err != nil {
		response.InternalError(c, "清空会话失败: "+err.Error())
		return
	}
	response.Success(c, gin.H{"message": "所有会话已清空"})
}

// ───────────────────── 内部辅助方法 ─────────────────────

// resolveGroupID 解析 groupID：优先使用请求中指定的，否则自动从用户可用分组中选取
func (h *UserImageHandler) resolveGroupID(c *gin.Context, userID int64, requested *int64) (*int64, error) {
	if requested != nil && *requested > 0 {
		return requested, nil
	}
	if h.apiKeySvc == nil {
		return nil, nil // 让调度器自行选择分组
	}

	// 先从用户订阅中找分组
	groups, err := h.apiKeySvc.GetAvailableGroups(c.Request.Context(), userID)
	if err == nil {
		for _, g := range groups {
			if g.ID > 0 {
				gid := g.ID
				return &gid, nil
			}
		}
	}

	// 用户没有订阅分组时（如 admin），不指定分组，让调度器自动选择
	return nil, nil
}

// recordToSession 将生成请求记录到已有会话
func (h *UserImageHandler) recordToSession(userID int64, sessionID string, parsed *service.OpenAIImagesRequest) {
	if h.imageSessionSvc == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	record := service.ImageRecord{
		ID:        fmt.Sprintf("img_rec_%d", time.Now().UnixNano()),
		Prompt:    parsed.Prompt,
		Model:     parsed.Model,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	_ = h.imageSessionSvc.AddRecord(ctx, userID, sessionID, record)
}

// autoCreateAndRecord 自动创建会话并记录
func (h *UserImageHandler) autoCreateAndRecord(userID int64, title string, parsed *service.OpenAIImagesRequest) {
	if h.imageSessionSvc == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	session, err := h.imageSessionSvc.CreateSession(ctx, userID, title)
	if err != nil {
		return
	}

	record := service.ImageRecord{
		ID:        fmt.Sprintf("img_rec_%d", time.Now().UnixNano()),
		Prompt:    parsed.Prompt,
		Model:     parsed.Model,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	_ = h.imageSessionSvc.AddRecord(ctx, userID, session.ID, record)
}
