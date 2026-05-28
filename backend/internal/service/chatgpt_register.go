package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// ChatGPTRegisterService manages automated ChatGPT account registration.
type ChatGPTRegisterService struct {
	mu       sync.RWMutex
	store    SettingRepository
	accounts AccountRepository
	cfg      ChatGPTRegisterConfig
	logs     []ChatGPTRegisterLog
	running  atomic.Bool
	cancel   context.CancelFunc
}

type ChatGPTRegisterConfig struct {
	Enabled       bool                      `json:"enabled"`
	Mode          string                    `json:"mode"` // total | quota | available
	Total         int                       `json:"total"`
	Threads       int                       `json:"threads"`
	Proxy         string                    `json:"proxy"`
	TargetQuota   int                       `json:"target_quota"`
	TargetAvail   int                       `json:"target_available"`
	CheckInterval int                       `json:"check_interval"`
	Mail          ChatGPTRegisterMailConfig `json:"mail"`
	// Legacy flat fields kept for API/backward-compatible migration.
	MailProvider string               `json:"mail_provider,omitempty"`
	MailAPIBase  string               `json:"mail_api_base,omitempty"`
	MailAPIKey   string               `json:"mail_api_key,omitempty"`
	Stats        ChatGPTRegisterStats `json:"stats"`
}

type ChatGPTRegisterMailConfig struct {
	RequestTimeout float64                             `json:"request_timeout"`
	WaitTimeout    float64                             `json:"wait_timeout"`
	WaitInterval   float64                             `json:"wait_interval"`
	UserAgent      string                              `json:"user_agent,omitempty"`
	Providers      []ChatGPTRegisterMailProviderConfig `json:"providers"`
}

type ChatGPTRegisterMailProviderConfig struct {
	Type            string                    `json:"type"`
	Enable          bool                      `json:"enable"`
	ProviderRef     string                    `json:"provider_ref,omitempty"`
	Label           string                    `json:"label,omitempty"`
	APIBase         string                    `json:"api_base,omitempty"`
	APIKey          string                    `json:"api_key,omitempty"`
	AdminPassword   string                    `json:"admin_password,omitempty"`
	AdminEmail      string                    `json:"admin_email,omitempty"`
	DDGToken        string                    `json:"ddg_token,omitempty"`
	CFInboxJWT      string                    `json:"cf_inbox_jwt,omitempty"`
	CFAPIBase       string                    `json:"cf_api_base,omitempty"`
	CFAPIKey        string                    `json:"cf_api_key,omitempty"`
	CFAuthMode      string                    `json:"cf_auth_mode,omitempty"`
	CFDomain        chatGPTRegisterStringList `json:"cf_domain,omitempty"`
	CFCreatePath    string                    `json:"cf_create_path,omitempty"`
	CFMessagesPath  string                    `json:"cf_messages_path,omitempty"`
	Domain          chatGPTRegisterStringList `json:"domain,omitempty"`
	Subdomain       chatGPTRegisterStringList `json:"subdomain,omitempty"`
	DefaultDomain   string                    `json:"default_domain,omitempty"`
	ExpiryTime      int                       `json:"expiry_time,omitempty"`
	RandomSubdomain *bool                     `json:"random_subdomain,omitempty"`
	Wildcard        bool                      `json:"wildcard,omitempty"`
	EmailPrefix     string                    `json:"email_prefix,omitempty"`
	AuthMode        string                    `json:"auth_mode,omitempty"`
}

type ChatGPTRegisterStats struct {
	JobID          string  `json:"job_id,omitempty"`
	Success        int     `json:"success"`
	Fail           int     `json:"fail"`
	Done           int     `json:"done"`
	Running        int     `json:"running"`
	Threads        int     `json:"threads"`
	ElapsedSeconds float64 `json:"elapsed_seconds"`
	AvgSeconds     float64 `json:"avg_seconds"`
	SuccessRate    float64 `json:"success_rate"`
	CurrentQuota   int     `json:"current_quota"`
	CurrentAvail   int     `json:"current_available"`
	StartedAt      string  `json:"started_at,omitempty"`
	UpdatedAt      string  `json:"updated_at,omitempty"`
	FinishedAt     string  `json:"finished_at,omitempty"`
}

type ChatGPTRegisterLog struct {
	Time  string `json:"time"`
	Text  string `json:"text"`
	Level string `json:"level"`
}

const chatGPTRegisterSettingKey = "chatgpt_register_config"

func NewChatGPTRegisterService(store SettingRepository, accounts AccountRepository) *ChatGPTRegisterService {
	svc := &ChatGPTRegisterService{
		store:    store,
		accounts: accounts,
		cfg:      defaultRegisterConfig(),
	}
	svc.loadConfig()
	return svc
}

func defaultRegisterConfig() ChatGPTRegisterConfig {
	cfg := ChatGPTRegisterConfig{
		Enabled:       false,
		Mode:          "total",
		Total:         10,
		Threads:       3,
		TargetQuota:   100,
		TargetAvail:   10,
		CheckInterval: 5,
		Mail: ChatGPTRegisterMailConfig{
			RequestTimeout: 30,
			WaitTimeout:    120,
			WaitInterval:   3,
			Providers: []ChatGPTRegisterMailProviderConfig{{
				Type:   "mailtm",
				Enable: true,
			}},
		},
		Stats: ChatGPTRegisterStats{
			Threads: 3,
		},
	}
	return chatGPTRegisterNormalizeConfig(cfg)
}

func (s *ChatGPTRegisterService) loadConfig() {
	if s.store == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	raw, err := s.store.GetValue(ctx, chatGPTRegisterSettingKey)
	if err != nil || raw == "" {
		return
	}
	var cfg ChatGPTRegisterConfig
	if json.Unmarshal([]byte(raw), &cfg) == nil {
		s.cfg = chatGPTRegisterNormalizeConfig(cfg)
	}
}

func (s *ChatGPTRegisterService) saveConfig() {
	if s.store == nil {
		return
	}
	data, err := json.Marshal(s.cfg)
	if err != nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	_ = s.store.Set(ctx, chatGPTRegisterSettingKey, string(data))
}

// Get returns current config + logs.
func (s *ChatGPTRegisterService) Get() map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	cfgCopy := chatGPTRegisterNormalizeConfig(s.cfg)
	cfgCopy.Stats.Threads = cfgCopy.Threads
	logsCopy := make([]ChatGPTRegisterLog, len(s.logs))
	copy(logsCopy, s.logs)
	if len(logsCopy) > 300 {
		logsCopy = logsCopy[len(logsCopy)-300:]
	}
	return map[string]any{
		"register": map[string]any{
			"enabled":          cfgCopy.Enabled,
			"mode":             cfgCopy.Mode,
			"total":            cfgCopy.Total,
			"threads":          cfgCopy.Threads,
			"proxy":            cfgCopy.Proxy,
			"target_quota":     cfgCopy.TargetQuota,
			"target_available": cfgCopy.TargetAvail,
			"check_interval":   cfgCopy.CheckInterval,
			"mail":             cfgCopy.Mail,
			"mail_provider":    cfgCopy.MailProvider,
			"mail_api_base":    cfgCopy.MailAPIBase,
			"mail_api_key":     cfgCopy.MailAPIKey,
			"stats":            cfgCopy.Stats,
			"logs":             logsCopy,
		},
	}
}

// Update updates register configuration.
func (s *ChatGPTRegisterService) Update(updates map[string]any) map[string]any {
	s.mu.Lock()
	if v, ok := updates["mode"].(string); ok && (v == "total" || v == "quota" || v == "available") {
		s.cfg.Mode = v
	}
	if v, ok := updates["total"].(float64); ok && v >= 1 {
		s.cfg.Total = int(v)
	}
	if v, ok := updates["threads"].(float64); ok && v >= 1 {
		s.cfg.Threads = int(v)
	}
	if v, ok := updates["proxy"].(string); ok {
		s.cfg.Proxy = v
	}
	if v, ok := updates["target_quota"].(float64); ok && v >= 1 {
		s.cfg.TargetQuota = int(v)
	}
	if v, ok := updates["target_available"].(float64); ok && v >= 1 {
		s.cfg.TargetAvail = int(v)
	}
	if v, ok := updates["check_interval"].(float64); ok && v >= 1 {
		s.cfg.CheckInterval = int(v)
	}
	legacyMailChanged := false
	if v, ok := updates["mail_provider"].(string); ok {
		s.cfg.MailProvider = strings.TrimSpace(v)
		legacyMailChanged = true
	}
	if v, ok := updates["mail_api_base"].(string); ok {
		s.cfg.MailAPIBase = strings.TrimSpace(v)
		legacyMailChanged = true
	}
	if v, ok := updates["mail_api_key"].(string); ok {
		s.cfg.MailAPIKey = strings.TrimSpace(v)
		legacyMailChanged = true
	}
	if legacyMailChanged {
		providerType := strings.TrimSpace(s.cfg.MailProvider)
		if providerType == "" || providerType == "custom" {
			providerType = "mailtm"
		}
		s.cfg.Mail.Providers = []ChatGPTRegisterMailProviderConfig{{Type: providerType, Enable: true, APIBase: strings.TrimSpace(s.cfg.MailAPIBase), APIKey: strings.TrimSpace(s.cfg.MailAPIKey)}}
	}
	if v, ok := updates["mail"]; ok {
		if mail, err := chatGPTRegisterDecodeMailConfig(v, s.cfg.Mail); err == nil {
			s.cfg.Mail = mail
		}
	}
	s.cfg = chatGPTRegisterNormalizeConfig(s.cfg)
	s.saveConfig()
	s.mu.Unlock()
	return s.Get()
}

// Start starts the registration goroutine.
func (s *ChatGPTRegisterService) Start() map[string]any {
	s.mu.Lock()
	if s.running.Load() {
		s.cfg.Enabled = true
		s.saveConfig()
		s.mu.Unlock()
		return s.Get()
	}
	s.cfg.Enabled = true
	s.cfg.Stats = ChatGPTRegisterStats{
		JobID:     fmt.Sprintf("%x", time.Now().UnixNano()),
		Threads:   s.cfg.Threads,
		StartedAt: time.Now().UTC().Format(time.RFC3339),
		UpdatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	s.logs = nil
	s.saveConfig()
	ctx, cancel := context.WithCancel(context.Background())
	s.cancel = cancel
	s.running.Store(true)
	s.mu.Unlock()

	go s.run(ctx)
	s.appendLog("注册任务启动", "info")
	return s.Get()
}

// Stop stops the registration goroutine.
func (s *ChatGPTRegisterService) Stop() map[string]any {
	s.mu.Lock()
	s.cfg.Enabled = false
	s.cfg.Stats.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	s.saveConfig()
	if s.cancel != nil {
		s.cancel()
	}
	s.mu.Unlock()
	s.appendLog("已请求停止注册任务", "info")
	return s.Get()
}

// Reset resets stats.
func (s *ChatGPTRegisterService) Reset() map[string]any {
	s.mu.Lock()
	s.cfg.Stats = ChatGPTRegisterStats{Threads: s.cfg.Threads, UpdatedAt: time.Now().UTC().Format(time.RFC3339)}
	s.logs = nil
	s.saveConfig()
	s.mu.Unlock()
	return s.Get()
}

func (s *ChatGPTRegisterService) appendLog(text, level string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.logs = append(s.logs, ChatGPTRegisterLog{
		Time:  time.Now().UTC().Format(time.RFC3339),
		Text:  text,
		Level: level,
	})
	if len(s.logs) > 300 {
		s.logs = s.logs[len(s.logs)-300:]
	}
}

func (s *ChatGPTRegisterService) bumpStats(success, fail, done, running int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.cfg.Stats.Success += success
	s.cfg.Stats.Fail += fail
	s.cfg.Stats.Done += done
	s.cfg.Stats.Running = running
	s.cfg.Stats.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	if s.cfg.Stats.StartedAt != "" {
		started, err := time.Parse(time.RFC3339, s.cfg.Stats.StartedAt)
		if err == nil {
			elapsed := time.Since(started).Seconds()
			s.cfg.Stats.ElapsedSeconds = elapsed
			if s.cfg.Stats.Success > 0 {
				s.cfg.Stats.AvgSeconds = elapsed / float64(s.cfg.Stats.Success)
			}
			total := s.cfg.Stats.Success + s.cfg.Stats.Fail
			if total > 0 {
				s.cfg.Stats.SuccessRate = float64(s.cfg.Stats.Success) * 100 / float64(total)
			}
		}
	}
	s.saveConfig()
}

func (s *ChatGPTRegisterService) run(ctx context.Context) {
	defer func() {
		s.running.Store(false)
		s.mu.Lock()
		s.cfg.Enabled = false
		s.cfg.Stats.FinishedAt = time.Now().UTC().Format(time.RFC3339)
		s.cfg.Stats.Running = 0
		s.saveConfig()
		s.mu.Unlock()
		s.appendLog("注册任务结束", "info")
	}()

	s.appendLog(fmt.Sprintf("注册任务运行中，模式=%s，线程=%d", s.cfg.Mode, s.cfg.Threads), "info")

	submitted := 0
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		cfg := s.currentConfig()
		if !cfg.Enabled {
			return
		}
		if s.targetReached(cfg, submitted) {
			s.appendLog("已达到目标，停止注册", "info")
			return
		}

		// Execute one registration
		s.bumpStats(0, 0, 0, 1)
		s.appendLog(fmt.Sprintf("开始注册第 %d 个账号", submitted+1), "info")
		result := s.registerOne(ctx, cfg)
		if result {
			s.bumpStats(1, 0, 1, 0)
			s.appendLog(fmt.Sprintf("第 %d 个注册成功", submitted+1), "info")
		} else {
			s.bumpStats(0, 1, 1, 0)
			s.appendLog(fmt.Sprintf("第 %d 个注册失败", submitted+1), "error")
		}
		submitted++

		// Wait between registrations
		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}

func (s *ChatGPTRegisterService) currentConfig() ChatGPTRegisterConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return chatGPTRegisterNormalizeConfig(s.cfg)
}

func (s *ChatGPTRegisterService) targetReached(cfg ChatGPTRegisterConfig, submitted int) bool {
	quota, available := s.chatGPTWebAccountStats(context.Background())
	s.mu.Lock()
	s.cfg.Stats.CurrentQuota = quota
	s.cfg.Stats.CurrentAvail = available
	s.cfg.Stats.UpdatedAt = time.Now().UTC().Format(time.RFC3339)
	s.mu.Unlock()

	switch cfg.Mode {
	case "quota":
		return quota >= cfg.TargetQuota
	case "available":
		return available >= cfg.TargetAvail
	default:
		return submitted >= cfg.Total
	}
}

// registerOne performs a single ChatGPT account registration.
// Returns true on success.
func (s *ChatGPTRegisterService) registerOne(ctx context.Context, cfg ChatGPTRegisterConfig) bool {
	// Step 1: Create temp email
	mailbox, err := s.createTempEmail(cfg)
	if err != nil {
		s.appendLog(fmt.Sprintf("创建临时邮箱失败: %v", err), "error")
		return false
	}
	email := mailbox.Email
	s.appendLog(fmt.Sprintf("创建临时邮箱: %s", email), "info")

	codeVerifier, codeChallenge, state, nonce := chatGPTRegisterGeneratePKCE()
	deviceID := chatGPTRegisterRandomUUID()
	registrar, err := newChatGPTRegisterOpenAIClient(cfg.Proxy, deviceID)
	if err != nil {
		s.appendLog(fmt.Sprintf("初始化 OpenAI 注册客户端失败: %v", err), "error")
		return false
	}
	defer registrar.close()

	password := chatGPTRegisterRandomPassword(16)
	firstName, lastName := chatGPTRegisterRandomName()
	birthdate := chatGPTRegisterRandomBirthdate()

	if err = registrar.platformAuthorize(ctx, email, codeChallenge, state, nonce); err != nil {
		s.appendLog(fmt.Sprintf("platform authorize 失败: %v", err), "error")
		return false
	}
	s.appendLog("platform authorize 成功", "info")
	if err = registrar.registerUser(ctx, email, password); err != nil {
		s.appendLog(fmt.Sprintf("注册用户失败: %v", err), "error")
		return false
	}
	s.appendLog(fmt.Sprintf("注册用户成功: %s", email), "info")
	if err = registrar.sendOTP(ctx); err != nil {
		s.appendLog(fmt.Sprintf("发送验证码失败: %v", err), "error")
		return false
	}
	s.appendLog("发送验证码成功，等待邮件验证码...", "info")
	code, err := s.waitForOTPCode(ctx, mailbox, cfg)
	if err != nil {
		s.appendLog(fmt.Sprintf("获取验证码失败: %v", err), "error")
		return false
	}
	s.appendLog(fmt.Sprintf("获取验证码成功: %s", code), "info")
	if err = registrar.validateOTP(ctx, code); err != nil {
		s.appendLog(fmt.Sprintf("验证码校验失败: %v", err), "error")
		return false
	}
	s.appendLog("验证码校验成功", "info")
	if err = registrar.createAccountProfile(ctx, firstName+" "+lastName, birthdate); err != nil {
		s.appendLog(fmt.Sprintf("创建账号资料失败: %v", err), "error")
		return false
	}
	s.appendLog(fmt.Sprintf("创建账号资料成功: %s", firstName+" "+lastName), "info")
	tokens, err := registrar.loginAndExchangeTokens(ctx, email, password, codeVerifier, mailbox, cfg, s)
	if err != nil {
		s.appendLog(fmt.Sprintf("换 token 失败: %v", err), "error")
		return false
	}
	s.appendLog("换 token 成功", "info")

	// Step 9: Save account
	err = s.accounts.Create(ctx, &Account{
		Name:     email,
		Platform: PlatformOpenAI,
		Type:     AccountTypeOAuth,
		Credentials: map[string]any{
			"access_token":  tokens.AccessToken,
			"refresh_token": tokens.RefreshToken,
			"id_token":      tokens.IDToken,
			"email":         email,
			"password":      password,
		},
		Extra:       map[string]any{"openai_backend_mode": "chatgpt_web"},
		Status:      StatusActive,
		Concurrency: 3,
		Priority:    50,
	})
	if err != nil {
		s.appendLog(fmt.Sprintf("保存账号失败: %v", err), "error")
		return false
	}

	s.appendLog(fmt.Sprintf("注册成功: %s", email), "info")
	return true
}

type tempMailbox struct {
	Email       string
	Password    string
	ID          string
	Provider    string
	ProviderRef string
	Token       string
	EmailID     string
	AccountID   string
	MailboxName string
	Label       string
	Extra       map[string]any
	seenRefs    map[string]bool
}

type registerTokens struct {
	AccessToken  string
	RefreshToken string
	IDToken      string
}

func (s *ChatGPTRegisterService) createTempEmail(cfg ChatGPTRegisterConfig) (*tempMailbox, error) {
	return chatGPTRegisterCreateMailbox(context.Background(), cfg, "")
}

func (s *ChatGPTRegisterService) platformAuthorize(ctx context.Context, email, deviceID, codeChallenge, state, nonce, proxyURL string) error {
	client, err := newChatGPTRegisterOpenAIClient(proxyURL, deviceID)
	if err != nil {
		return err
	}
	defer client.close()
	return client.platformAuthorize(ctx, email, codeChallenge, state, nonce)
}

func (s *ChatGPTRegisterService) registerUser(ctx context.Context, email, password, deviceID, proxyURL string) error {
	client, err := newChatGPTRegisterOpenAIClient(proxyURL, deviceID)
	if err != nil {
		return err
	}
	defer client.close()
	return client.registerUser(ctx, email, password)
}

func (s *ChatGPTRegisterService) sendOTP(ctx context.Context, deviceID, proxyURL string) error {
	client, err := newChatGPTRegisterOpenAIClient(proxyURL, deviceID)
	if err != nil {
		return err
	}
	defer client.close()
	return client.sendOTP(ctx)
}

func (s *ChatGPTRegisterService) waitForOTPCode(ctx context.Context, mailbox *tempMailbox, cfg ChatGPTRegisterConfig) (string, error) {
	return chatGPTRegisterWaitForCode(ctx, cfg, mailbox)
}

func (s *ChatGPTRegisterService) validateOTP(ctx context.Context, code, deviceID, proxyURL string) error {
	client, err := newChatGPTRegisterOpenAIClient(proxyURL, deviceID)
	if err != nil {
		return err
	}
	defer client.close()
	return client.validateOTP(ctx, code)
}

func (s *ChatGPTRegisterService) createAccountProfile(ctx context.Context, firstName, lastName, birthdate, deviceID, proxyURL string) error {
	client, err := newChatGPTRegisterOpenAIClient(proxyURL, deviceID)
	if err != nil {
		return err
	}
	defer client.close()
	name := strings.TrimSpace(firstName + " " + lastName)
	return client.createAccountProfile(ctx, name, birthdate)
}

func (s *ChatGPTRegisterService) exchangeTokens(ctx context.Context, email, password, codeVerifier, deviceID, proxyURL string) (*registerTokens, error) {
	client, err := newChatGPTRegisterOpenAIClient(proxyURL, deviceID)
	if err != nil {
		return nil, err
	}
	defer client.close()
	return client.loginAndExchangeTokens(ctx, email, password, codeVerifier, &tempMailbox{Email: email}, ChatGPTRegisterConfig{Proxy: proxyURL, Mail: defaultRegisterConfig().Mail}, s)
}

// Helpers

func chatGPTRegisterGeneratePKCE() (verifier, challenge, state, nonce string) {
	b := make([]byte, 64)
	_, _ = rand.Read(b)
	verifier = strings.TrimRight(base64.URLEncoding.EncodeToString(b), "=")
	h := sha256.Sum256([]byte(verifier))
	challenge = strings.TrimRight(base64.URLEncoding.EncodeToString(h[:]), "=")
	state = chatGPTRegisterRandomHex(32)
	nonce = chatGPTRegisterRandomHex(32)
	return
}

func chatGPTRegisterRandomPassword(length int) string {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%"
	b := make([]byte, length)
	_, _ = rand.Read(b)
	for i, v := range b {
		b[i] = chars[int(v)%len(chars)]
	}
	return string(b)
}

func chatGPTRegisterRandomName() (string, string) {
	first := []string{"James", "Robert", "John", "Michael", "David", "Emma", "Olivia", "Sophia"}
	last := []string{"Smith", "Johnson", "Williams", "Brown", "Jones", "Garcia", "Miller"}
	var a, b [1]byte
	_, _ = rand.Read(a[:])
	_, _ = rand.Read(b[:])
	return first[int(a[0])%len(first)], last[int(b[0])%len(last)]
}

func chatGPTRegisterRandomBirthdate() string {
	var b [3]byte
	_, _ = rand.Read(b[:])
	year := 1996 + int(b[0])%11
	month := 1 + int(b[1])%12
	day := 1 + int(b[2])%28
	return fmt.Sprintf("%04d-%02d-%02d", year, month, day)
}

func chatGPTRegisterRandomHex(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return fmt.Sprintf("%x", b)
}

func chatGPTRegisterRandomUUID() string {
	b := make([]byte, 16)
	_, _ = rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
}

func chatGPTRegisterExtractOTP(text string) string {
	return chatGPTRegisterExtractCode(text)
}

// Ensure unused imports don't cause compile error
var _ = context.Background
