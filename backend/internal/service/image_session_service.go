package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"
)

// ImageSession 图片会话
type ImageSession struct {
	ID        string        `json:"id"`
	UserID    int64         `json:"user_id"`
	Title     string        `json:"title"`
	Records   []ImageRecord `json:"records"`
	CreatedAt string        `json:"created_at"`
	UpdatedAt string        `json:"updated_at"`
}

// ImageRecord 图片生成记录
type ImageRecord struct {
	ID        string   `json:"id"`
	Prompt    string   `json:"prompt"`
	Model     string   `json:"model"`
	Images    []string `json:"images"` // base64 或 URL
	Params    any      `json:"params"`
	CreatedAt string   `json:"created_at"`
}

// imageSessionStore 存储在 setting 表中的数据结构
type imageSessionStore struct {
	Sessions []ImageSession `json:"sessions"`
}

// ImageSessionService 图片会话管理服务
type ImageSessionService struct {
	store SettingRepository
}

// NewImageSessionService 创建图片会话服务
func NewImageSessionService(store SettingRepository) *ImageSessionService {
	return &ImageSessionService{store: store}
}

// imageSessionKey 生成存储 key
func imageSessionKey(userID int64) string {
	return fmt.Sprintf("image_sessions_%d", userID)
}

// loadSessions 从存储中加载用户的所有会话
func (s *ImageSessionService) loadSessions(ctx context.Context, userID int64) ([]ImageSession, error) {
	if s.store == nil {
		return nil, fmt.Errorf("setting store not available")
	}
	raw, err := s.store.GetValue(ctx, imageSessionKey(userID))
	if err != nil {
		// 如果 key 不存在，返回空列表
		return nil, nil
	}
	if raw == "" {
		return nil, nil
	}
	var store imageSessionStore
	if err := json.Unmarshal([]byte(raw), &store); err != nil {
		return nil, fmt.Errorf("unmarshal image sessions: %w", err)
	}
	return store.Sessions, nil
}

// saveSessions 将用户的所有会话保存到存储
func (s *ImageSessionService) saveSessions(ctx context.Context, userID int64, sessions []ImageSession) error {
	if s.store == nil {
		return fmt.Errorf("setting store not available")
	}
	store := imageSessionStore{Sessions: sessions}
	data, err := json.Marshal(store)
	if err != nil {
		return fmt.Errorf("marshal image sessions: %w", err)
	}
	return s.store.Set(ctx, imageSessionKey(userID), string(data))
}

// ListSessions 列出用户的所有会话
func (s *ImageSessionService) ListSessions(ctx context.Context, userID int64) ([]ImageSession, error) {
	sessions, err := s.loadSessions(ctx, userID)
	if err != nil {
		return nil, err
	}
	if sessions == nil {
		return []ImageSession{}, nil
	}
	return sessions, nil
}

// GetSession 获取用户的单个会话
func (s *ImageSessionService) GetSession(ctx context.Context, userID int64, sessionID string) (*ImageSession, error) {
	sessions, err := s.loadSessions(ctx, userID)
	if err != nil {
		return nil, err
	}
	for i := range sessions {
		if sessions[i].ID == sessionID {
			return &sessions[i], nil
		}
	}
	return nil, fmt.Errorf("session not found")
}

// CreateSession 创建新的图片会话
func (s *ImageSessionService) CreateSession(ctx context.Context, userID int64, title string) (*ImageSession, error) {
	sessions, err := s.loadSessions(ctx, userID)
	if err != nil {
		return nil, err
	}
	if sessions == nil {
		sessions = []ImageSession{}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	session := ImageSession{
		ID:        fmt.Sprintf("img_sess_%d_%d", userID, time.Now().UnixNano()),
		UserID:    userID,
		Title:     title,
		Records:   []ImageRecord{},
		CreatedAt: now,
		UpdatedAt: now,
	}

	sessions = append(sessions, session)
	if err := s.saveSessions(ctx, userID, sessions); err != nil {
		return nil, err
	}
	return &session, nil
}

// AddRecord 向会话中添加图片生成记录
func (s *ImageSessionService) AddRecord(ctx context.Context, userID int64, sessionID string, record ImageRecord) error {
	sessions, err := s.loadSessions(ctx, userID)
	if err != nil {
		return err
	}
	for i := range sessions {
		if sessions[i].ID == sessionID {
			sessions[i].Records = append(sessions[i].Records, record)
			sessions[i].UpdatedAt = time.Now().UTC().Format(time.RFC3339)
			return s.saveSessions(ctx, userID, sessions)
		}
	}
	return fmt.Errorf("session not found")
}

// DeleteSession 删除用户的单个会话
func (s *ImageSessionService) DeleteSession(ctx context.Context, userID int64, sessionID string) error {
	sessions, err := s.loadSessions(ctx, userID)
	if err != nil {
		return err
	}
	for i := range sessions {
		if sessions[i].ID == sessionID {
			sessions = append(sessions[:i], sessions[i+1:]...)
			return s.saveSessions(ctx, userID, sessions)
		}
	}
	return fmt.Errorf("session not found")
}

// ClearSessions 清空用户的所有会话
func (s *ImageSessionService) ClearSessions(ctx context.Context, userID int64) error {
	return s.saveSessions(ctx, userID, []ImageSession{})
}
