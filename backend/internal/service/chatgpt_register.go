package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
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
	Enabled       bool                 `json:"enabled"`
	Mode          string               `json:"mode"` // total | quota | available
	Total         int                  `json:"total"`
	Threads       int                  `json:"threads"`
	Proxy         string               `json:"proxy"`
	TargetQuota   int                  `json:"target_quota"`
	TargetAvail   int                  `json:"target_available"`
	CheckInterval int                  `json:"check_interval"`
	MailProvider  string               `json:"mail_provider"` // e.g. "mailtm", "custom"
	MailAPIBase   string               `json:"mail_api_base"` // custom mail provider base URL
	MailAPIKey    string               `json:"mail_api_key"`  // custom mail provider API key
	Stats         ChatGPTRegisterStats `json:"stats"`
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
	return ChatGPTRegisterConfig{
		Enabled:       false,
		Mode:          "total",
		Total:         10,
		Threads:       3,
		TargetQuota:   100,
		TargetAvail:   10,
		CheckInterval: 5,
		Stats: ChatGPTRegisterStats{
			Threads: 3,
		},
	}
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
		s.cfg = cfg
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
	cfgCopy := s.cfg
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
			"stats":            cfgCopy.Stats,
			"logs":             logsCopy,
		},
	}
}

// Update updates register configuration.
func (s *ChatGPTRegisterService) Update(updates map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
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
	if v, ok := updates["mail_provider"].(string); ok {
		s.cfg.MailProvider = v
	}
	if v, ok := updates["mail_api_base"].(string); ok {
		s.cfg.MailAPIBase = v
	}
	if v, ok := updates["mail_api_key"].(string); ok {
		s.cfg.MailAPIKey = v
	}
	s.saveConfig()
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
	return s.cfg
}

func (s *ChatGPTRegisterService) targetReached(cfg ChatGPTRegisterConfig, submitted int) bool {
	switch cfg.Mode {
	case "quota":
		// Would need to check actual quota from accounts
		return submitted >= cfg.Total
	case "available":
		return submitted >= cfg.Total
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

	// Step 2: PKCE + authorize
	codeVerifier, codeChallenge, state, nonce := chatGPTRegisterGeneratePKCE()
	deviceID := chatGPTRegisterRandomUUID()
	proxyURL := cfg.Proxy

	err = s.platformAuthorize(ctx, email, deviceID, codeChallenge, state, nonce, proxyURL)
	if err != nil {
		s.appendLog(fmt.Sprintf("platform authorize 失败: %v", err), "error")
		return false
	}

	// Step 3: Register user with password
	password := chatGPTRegisterRandomPassword(16)
	err = s.registerUser(ctx, email, password, deviceID, proxyURL)
	if err != nil {
		s.appendLog(fmt.Sprintf("注册用户失败: %v", err), "error")
		return false
	}

	// Step 4: Send OTP
	err = s.sendOTP(ctx, deviceID, proxyURL)
	if err != nil {
		s.appendLog(fmt.Sprintf("发送验证码失败: %v", err), "error")
		return false
	}

	// Step 5: Wait for OTP code
	code, err := s.waitForOTPCode(ctx, mailbox, cfg)
	if err != nil {
		s.appendLog(fmt.Sprintf("获取验证码失败: %v", err), "error")
		return false
	}

	// Step 6: Validate OTP
	err = s.validateOTP(ctx, code, deviceID, proxyURL)
	if err != nil {
		s.appendLog(fmt.Sprintf("验证码校验失败: %v", err), "error")
		return false
	}

	// Step 7: Create account profile
	firstName, lastName := chatGPTRegisterRandomName()
	birthdate := chatGPTRegisterRandomBirthdate()
	err = s.createAccountProfile(ctx, firstName, lastName, birthdate, deviceID, proxyURL)
	if err != nil {
		s.appendLog(fmt.Sprintf("创建账号资料失败: %v", err), "error")
		return false
	}

	// Step 8: Exchange tokens
	tokens, err := s.exchangeTokens(ctx, email, password, codeVerifier, deviceID, proxyURL)
	if err != nil {
		s.appendLog(fmt.Sprintf("换 token 失败: %v", err), "error")
		return false
	}

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
	Email    string
	Password string
	ID       string
}

type registerTokens struct {
	AccessToken  string
	RefreshToken string
	IDToken      string
}

func (s *ChatGPTRegisterService) createTempEmail(cfg ChatGPTRegisterConfig) (*tempMailbox, error) {
	// Use mail.tm API to create a temp email
	client := &http.Client{Timeout: 30 * time.Second}
	domainsResp, err := client.Get("https://api.mail.tm/domains")
	if err != nil {
		return nil, fmt.Errorf("fetch domains: %w", err)
	}
	defer func() { _ = domainsResp.Body.Close() }()
	var domainsData struct {
		HydraMember []struct {
			Domain string `json:"domain"`
		} `json:"hydra:member"`
	}
	if err := json.NewDecoder(domainsResp.Body).Decode(&domainsData); err != nil {
		return nil, fmt.Errorf("parse domains: %w", err)
	}
	if len(domainsData.HydraMember) == 0 {
		return nil, fmt.Errorf("no domains available")
	}
	domain := domainsData.HydraMember[0].Domain
	localPart := fmt.Sprintf("r%x", time.Now().UnixNano()%0xFFFFFF)
	email := localPart + "@" + domain
	password := chatGPTRegisterRandomPassword(12)

	createBody, _ := json.Marshal(map[string]string{"address": email, "password": password})
	createResp, err := client.Post("https://api.mail.tm/accounts", "application/json", strings.NewReader(string(createBody)))
	if err != nil {
		return nil, fmt.Errorf("create mailbox: %w", err)
	}
	defer func() { _ = createResp.Body.Close() }()
	if createResp.StatusCode >= 400 {
		return nil, fmt.Errorf("create mailbox HTTP %d", createResp.StatusCode)
	}
	return &tempMailbox{Email: email, Password: password}, nil
}

func (s *ChatGPTRegisterService) platformAuthorize(ctx context.Context, email, deviceID, codeChallenge, state, nonce, proxyURL string) error {
	params := url.Values{
		"issuer":                {"https://auth.openai.com"},
		"client_id":             {"app_2SKx67EdpoN0G6j64rFvigXD"},
		"audience":              {"https://api.openai.com/v1"},
		"redirect_uri":          {"https://platform.openai.com/auth/callback"},
		"device_id":             {deviceID},
		"screen_hint":           {"login_or_signup"},
		"max_age":               {"0"},
		"login_hint":            {email},
		"scope":                 {"openid profile email offline_access"},
		"response_type":         {"code"},
		"response_mode":         {"query"},
		"state":                 {state},
		"nonce":                 {nonce},
		"code_challenge":        {codeChallenge},
		"code_challenge_method": {"S256"},
	}
	reqURL := "https://auth.openai.com/api/accounts/authorize?" + params.Encode()
	req, _ := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	req.Header.Set("User-Agent", chatGPTWebDefaultUserAgent)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("authorize HTTP %d", resp.StatusCode)
	}
	return nil
}

func (s *ChatGPTRegisterService) registerUser(ctx context.Context, email, password, deviceID, proxyURL string) error {
	body, _ := json.Marshal(map[string]string{"username": email, "password": password})
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://auth.openai.com/api/accounts/user/register", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", chatGPTWebDefaultUserAgent)
	req.Header.Set("Oai-Device-Id", deviceID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("register HTTP %d", resp.StatusCode)
	}
	return nil
}

func (s *ChatGPTRegisterService) sendOTP(ctx context.Context, deviceID, proxyURL string) error {
	req, _ := http.NewRequestWithContext(ctx, "GET", "https://auth.openai.com/api/accounts/email-otp/send", nil)
	req.Header.Set("User-Agent", chatGPTWebDefaultUserAgent)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("send OTP HTTP %d", resp.StatusCode)
	}
	return nil
}

func (s *ChatGPTRegisterService) waitForOTPCode(ctx context.Context, mailbox *tempMailbox, cfg ChatGPTRegisterConfig) (string, error) {
	client := &http.Client{Timeout: 10 * time.Second}
	tokenResp, err := client.Post("https://api.mail.tm/token", "application/json",
		strings.NewReader(fmt.Sprintf(`{"address":"%s","password":"%s"}`, mailbox.Email, mailbox.Password)))
	if err != nil {
		return "", fmt.Errorf("get mail token: %w", err)
	}
	defer func() { _ = tokenResp.Body.Close() }()
	var tokenData struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(tokenResp.Body).Decode(&tokenData); err != nil {
		return "", fmt.Errorf("parse mail token: %w", err)
	}
	if tokenData.Token == "" {
		return "", fmt.Errorf("empty mail token")
	}

	deadline := time.After(120 * time.Second)
	ticker := time.NewTicker(3 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-deadline:
			return "", fmt.Errorf("OTP wait timeout")
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			msgReq, _ := http.NewRequest("GET", "https://api.mail.tm/messages", nil)
			msgReq.Header.Set("Authorization", "Bearer "+tokenData.Token)
			msgResp, err := client.Do(msgReq)
			if err != nil {
				continue
			}
			var msgData struct {
				HydraMember []struct {
					Subject string `json:"subject"`
					Text    string `json:"text"`
				} `json:"hydra:member"`
			}
			_ = json.NewDecoder(msgResp.Body).Decode(&msgData)
			_ = msgResp.Body.Close()
			for _, msg := range msgData.HydraMember {
				if strings.Contains(msg.Subject, "OpenAI") || strings.Contains(msg.Text, "verification code") {
					code := chatGPTRegisterExtractOTP(msg.Text)
					if code != "" {
						return code, nil
					}
				}
			}
		}
	}
}

func (s *ChatGPTRegisterService) validateOTP(ctx context.Context, code, deviceID, proxyURL string) error {
	body, _ := json.Marshal(map[string]string{"code": code})
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://auth.openai.com/api/accounts/email-otp/validate", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", chatGPTWebDefaultUserAgent)
	req.Header.Set("Oai-Device-Id", deviceID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("validate OTP HTTP %d", resp.StatusCode)
	}
	return nil
}

func (s *ChatGPTRegisterService) createAccountProfile(ctx context.Context, firstName, lastName, birthdate, deviceID, proxyURL string) error {
	body, _ := json.Marshal(map[string]string{
		"name":      firstName + " " + lastName,
		"birthdate": birthdate,
	})
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://auth.openai.com/api/accounts/create_account", strings.NewReader(string(body)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", chatGPTWebDefaultUserAgent)
	req.Header.Set("Oai-Device-Id", deviceID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		return fmt.Errorf("create account HTTP %d", resp.StatusCode)
	}
	return nil
}

func (s *ChatGPTRegisterService) exchangeTokens(ctx context.Context, email, password, codeVerifier, deviceID, proxyURL string) (*registerTokens, error) {
	// Exchange authorization code for tokens
	body := url.Values{
		"grant_type":    {"authorization_code"},
		"redirect_uri":  {"https://platform.openai.com/auth/callback"},
		"client_id":     {"app_2SKx67EdpoN0G6j64rFvigXD"},
		"code_verifier": {codeVerifier},
	}
	req, _ := http.NewRequestWithContext(ctx, "POST", "https://auth.openai.com/oauth/token", strings.NewReader(body.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	var data struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		IDToken      string `json:"id_token"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, err
	}
	if data.AccessToken == "" || data.RefreshToken == "" {
		return nil, fmt.Errorf("token exchange failed: empty tokens")
	}
	return &registerTokens{
		AccessToken:  data.AccessToken,
		RefreshToken: data.RefreshToken,
		IDToken:      data.IDToken,
	}, nil
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
	// Extract 6-digit code from email text
	for i := 0; i <= len(text)-6; i++ {
		if text[i] >= '0' && text[i] <= '9' {
			end := i + 1
			for end < len(text) && text[end] >= '0' && text[end] <= '9' {
				end++
			}
			if end-i == 6 {
				return text[i:end]
			}
			i = end - 1
		}
	}
	return ""
}

// Ensure unused imports don't cause compile error
var _ = context.Background
