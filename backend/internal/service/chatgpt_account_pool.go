package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	fhttp "github.com/bogdanfinn/fhttp"
	tlsclient "github.com/bogdanfinn/tls-client"
	tlsclientprofiles "github.com/bogdanfinn/tls-client/profiles"
)

const (
	ChatGPTAccountPoolSettingKey = "chatgpt_account_pool_config"

	chatGPTAccountPoolOAuthTokenURL = "https://auth.openai.com/oauth/token"
	chatGPTAccountPoolOAuthClientID = "app_2SKx67EdpoN0G6j64rFvigXD"
	chatGPTAccountPoolRefreshSkew   = 24 * time.Hour
)

var ErrChatGPTAccountPoolInvalidAccessToken = errors.New("invalid access token")

// ChatGPTAccountPoolConfig stores account-pool-only settings.
type ChatGPTAccountPoolConfig struct {
	RefreshAccountIntervalMinute  int  `json:"refresh_account_interval_minute"`
	AutoRemoveInvalidAccounts     bool `json:"auto_remove_invalid_accounts"`
	AutoRemoveRateLimitedAccounts bool `json:"auto_remove_rate_limited_accounts"`
	ImageAccountConcurrency       int  `json:"image_account_concurrency"`
}

type ChatGPTAccountPoolTokenData struct {
	AccessToken  string
	RefreshToken string
	IDToken      string
}

type ChatGPTAccountPoolRemoteInfo struct {
	Email             string
	UserID            string
	AccountID         string
	PlanType          string
	Quota             int
	ImageQuotaUnknown bool
	LimitsProgress    any
	DefaultModelSlug  string
	RestoreAt         string
	Status            string
}

type ChatGPTAccountPoolRemoteClient interface {
	RefreshAccessToken(ctx context.Context, refreshToken string) (*ChatGPTAccountPoolTokenData, error)
	FetchRemoteInfo(ctx context.Context, accessToken string, account *Account) (*ChatGPTAccountPoolRemoteInfo, error)
}

type ChatGPTAccountPoolService struct {
	accounts AccountRepository
	settings SettingRepository
	remote   ChatGPTAccountPoolRemoteClient
}

func NewChatGPTAccountPoolService(accounts AccountRepository, settings SettingRepository) *ChatGPTAccountPoolService {
	return &ChatGPTAccountPoolService{accounts: accounts, settings: settings, remote: NewChatGPTAccountPoolHTTPClient(nil)}
}

func (s *ChatGPTAccountPoolService) SetRemoteClient(remote ChatGPTAccountPoolRemoteClient) {
	if remote != nil {
		s.remote = remote
	}
}

func DefaultChatGPTAccountPoolConfig() ChatGPTAccountPoolConfig {
	return ChatGPTAccountPoolConfig{
		RefreshAccountIntervalMinute:  5,
		AutoRemoveInvalidAccounts:     false,
		AutoRemoveRateLimitedAccounts: false,
		ImageAccountConcurrency:       3,
	}
}

func NormalizeChatGPTAccountPoolConfig(cfg ChatGPTAccountPoolConfig) ChatGPTAccountPoolConfig {
	if cfg.RefreshAccountIntervalMinute < 1 {
		cfg.RefreshAccountIntervalMinute = 5
	}
	if cfg.ImageAccountConcurrency < 1 {
		cfg.ImageAccountConcurrency = 3
	}
	return cfg
}

func (s *ChatGPTAccountPoolService) GetConfig(ctx context.Context) ChatGPTAccountPoolConfig {
	cfg := DefaultChatGPTAccountPoolConfig()
	if s == nil || s.settings == nil {
		return cfg
	}
	raw, err := s.settings.GetValue(ctx, ChatGPTAccountPoolSettingKey)
	if err != nil || strings.TrimSpace(raw) == "" {
		return cfg
	}
	var stored ChatGPTAccountPoolConfig
	if err := json.Unmarshal([]byte(raw), &stored); err != nil {
		return cfg
	}
	return NormalizeChatGPTAccountPoolConfig(stored)
}

func (s *ChatGPTAccountPoolService) UpdateConfig(ctx context.Context, updates map[string]any) (ChatGPTAccountPoolConfig, error) {
	cfg := s.GetConfig(ctx)
	if v, ok := intFromAny(updates["refresh_account_interval_minute"]); ok {
		cfg.RefreshAccountIntervalMinute = v
	}
	if v, ok := boolFromAny(updates["auto_remove_invalid_accounts"]); ok {
		cfg.AutoRemoveInvalidAccounts = v
	}
	if v, ok := boolFromAny(updates["auto_remove_rate_limited_accounts"]); ok {
		cfg.AutoRemoveRateLimitedAccounts = v
	}
	if v, ok := intFromAny(updates["image_account_concurrency"]); ok {
		cfg.ImageAccountConcurrency = v
	}
	cfg = NormalizeChatGPTAccountPoolConfig(cfg)
	if s != nil && s.settings != nil {
		data, err := json.Marshal(cfg)
		if err != nil {
			return cfg, err
		}
		if err := s.settings.Set(ctx, ChatGPTAccountPoolSettingKey, string(data)); err != nil {
			return cfg, err
		}
	}
	return cfg, nil
}

func (s *ChatGPTAccountPoolService) ListAccounts(ctx context.Context) ([]map[string]any, error) {
	accounts, err := s.listChatGPTAccounts(ctx)
	if err != nil {
		return nil, err
	}
	items := make([]map[string]any, 0, len(accounts))
	for i := range accounts {
		items = append(items, ChatGPTAccountToPoolItem(&accounts[i]))
	}
	return items, nil
}

func (s *ChatGPTAccountPoolService) CreateAccounts(ctx context.Context, tokens []string, payloads []map[string]any) (map[string]any, error) {
	if s == nil || s.accounts == nil {
		return nil, fmt.Errorf("account repository is not configured")
	}
	existing, err := s.chatgptAccountTokenSet(ctx)
	if err != nil {
		return nil, err
	}
	merged := make(map[string]map[string]any)
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		merged[token] = map[string]any{"access_token": token}
	}
	for _, payload := range payloads {
		prepared := prepareChatGPTAccountPayload(payload)
		if prepared == nil {
			continue
		}
		token := chatGPTPoolStringFromAny(prepared["access_token"])
		current := merged[token]
		if current == nil {
			current = map[string]any{}
		}
		for k, v := range prepared {
			current[k] = v
		}
		current["access_token"] = token
		merged[token] = current
	}
	keys := make([]string, 0, len(merged))
	for token := range merged {
		keys = append(keys, token)
	}
	sort.Strings(keys)
	added, skipped := 0, 0
	for _, token := range keys {
		if token == "" || existing[token] {
			skipped++
			continue
		}
		if err := s.createChatGPTAccount(ctx, merged[token]); err != nil {
			skipped++
			continue
		}
		existing[token] = true
		added++
	}
	items, _ := s.ListAccounts(ctx)
	return map[string]any{"added": added, "skipped": skipped, "items": items}, nil
}

func (s *ChatGPTAccountPoolService) DeleteAccounts(ctx context.Context, tokens []string) (map[string]any, error) {
	tokenIDs, err := s.chatgptAccountTokenIDs(ctx)
	if err != nil {
		return nil, err
	}
	removed := 0
	seen := map[string]struct{}{}
	for _, token := range tokens {
		token = strings.TrimSpace(token)
		if token == "" {
			continue
		}
		if _, ok := seen[token]; ok {
			continue
		}
		seen[token] = struct{}{}
		if id, ok := tokenIDs[token]; ok {
			if err := s.accounts.Delete(ctx, id); err == nil {
				removed++
			}
		}
	}
	items, _ := s.ListAccounts(ctx)
	return map[string]any{"removed": removed, "items": items}, nil
}

func (s *ChatGPTAccountPoolService) UpdateAccount(ctx context.Context, accessToken string, updates map[string]any) (*map[string]any, error) {
	acc, err := s.findByAccessToken(ctx, accessToken)
	if err != nil {
		return nil, err
	}
	if acc == nil {
		return nil, ErrAccountNotFound
	}
	if acc.Extra == nil {
		acc.Extra = map[string]any{}
	}
	if typ := strings.TrimSpace(chatGPTPoolStringFromAny(updates["type"])); typ != "" {
		if isNativeAccountType(typ) {
			acc.Type = typ
		} else {
			acc.Extra["plan_type"] = NormalizeChatGPTPlanType(typ)
		}
	}
	if status := strings.TrimSpace(chatGPTPoolStringFromAny(updates["status"])); status != "" {
		acc.Status = status
	}
	if quota, ok := intFromAny(updates["quota"]); ok {
		if quota < 0 {
			quota = 0
		}
		acc.Extra["quota"] = quota
	}
	if imageUnknown, ok := boolFromAny(updates["image_quota_unknown"]); ok {
		acc.Extra["image_quota_unknown"] = imageUnknown
	}
	if err := s.accounts.Update(ctx, acc); err != nil {
		return nil, err
	}
	item := ChatGPTAccountToPoolItem(acc)
	return &item, nil
}

func (s *ChatGPTAccountPoolService) RefreshAccounts(ctx context.Context, tokens []string) (map[string]any, error) {
	accounts, err := s.selectRefreshAccounts(ctx, tokens)
	if err != nil {
		return nil, err
	}
	if len(accounts) == 0 {
		items, _ := s.ListAccounts(ctx)
		return map[string]any{"refreshed": 0, "errors": []map[string]string{}, "items": items}, nil
	}
	cfg := s.GetConfig(ctx)
	refreshed := 0
	errorsOut := make([]map[string]string, 0)
	for i := range accounts {
		acc := accounts[i]
		beforeToken := acc.GetCredential("access_token")
		if beforeToken == "" {
			errorsOut = append(errorsOut, map[string]string{"token": "", "error": "missing access_token"})
			continue
		}
		updated, err := s.refreshOneAccount(ctx, &acc, cfg, false)
		if err != nil {
			errorsOut = append(errorsOut, map[string]string{"token": AnonymizeTokenForChatGPTPool(beforeToken), "error": err.Error()})
			continue
		}
		if updated {
			refreshed++
		}
	}
	items, _ := s.ListAccounts(ctx)
	return map[string]any{"refreshed": refreshed, "errors": errorsOut, "items": items}, nil
}

func (s *ChatGPTAccountPoolService) BuildExportItems(ctx context.Context, tokens []string) ([]map[string]string, error) {
	accounts, err := s.listChatGPTAccounts(ctx)
	if err != nil {
		return nil, err
	}
	targets := map[string]struct{}{}
	for _, token := range tokens {
		if token = strings.TrimSpace(token); token != "" {
			targets[token] = struct{}{}
		}
	}
	items := make([]map[string]string, 0, len(accounts))
	for i := range accounts {
		acc := &accounts[i]
		accessToken := strings.TrimSpace(acc.GetCredential("access_token"))
		if len(targets) > 0 {
			if _, ok := targets[accessToken]; !ok {
				continue
			}
		}
		refreshToken := strings.TrimSpace(acc.GetCredential("refresh_token"))
		idToken := strings.TrimSpace(acc.GetCredential("id_token"))
		if accessToken == "" || refreshToken == "" || idToken == "" {
			continue
		}
		accessPayload := DecodeJWTPayload(accessToken)
		idPayload := DecodeJWTPayload(idToken)
		authClaim, _ := accessPayload["https://api.openai.com/auth"].(map[string]any)
		profileClaim, _ := accessPayload["https://api.openai.com/profile"].(map[string]any)
		email := chatGPTPoolFirstNonEmpty(acc.GetCredential("email"), chatGPTPoolStringFromAny(profileClaim["email"]), chatGPTPoolStringFromAny(idPayload["email"]))
		accountID := chatGPTPoolFirstNonEmpty(acc.GetCredential("account_id"), chatGPTPoolStringFromAny(authClaim["chatgpt_account_id"]), acc.GetCredential("user_id"))
		item := map[string]string{
			"type":          chatGPTPoolFirstNonEmpty(chatGPTPoolStringFromAny(acc.Extra["export_type"]), "codex"),
			"export_type":   chatGPTPoolFirstNonEmpty(chatGPTPoolStringFromAny(acc.Extra["export_type"]), "codex"),
			"email":         email,
			"account_id":    accountID,
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"id_token":      idToken,
			"expired":       chatGPTPoolFirstNonEmpty(acc.GetCredential("expired"), timestampToISO(accessPayload["exp"])),
			"last_refresh":  chatGPTPoolFirstNonEmpty(acc.GetCredential("last_refresh"), timestampToISO(accessPayload["iat"])),
		}
		if password := acc.GetCredential("password"); password != "" {
			item["password"] = password
		}
		items = append(items, item)
	}
	return items, nil
}

func BuildChatGPTAccountExportJSON(items []map[string]string) ([]byte, error) {
	var payload any = items
	if len(items) == 1 {
		payload = items[0]
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

func BuildChatGPTAccountExportZIP(items []map[string]string) ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	zw := zip.NewWriter(buf)
	used := map[string]int{}
	for i, item := range items {
		fallback := fmt.Sprintf("account-%03d", i+1)
		raw := chatGPTPoolFirstNonEmpty(item["email"], item["account_id"], fallback)
		base := safeChatGPTExportName(raw, fallback)
		name := base
		if n := used[base]; n > 0 {
			name = fmt.Sprintf("%s-%d", base, n+1)
		}
		used[base]++
		w, err := zw.Create(name + ".json")
		if err != nil {
			_ = zw.Close()
			return nil, err
		}
		data, err := json.MarshalIndent(item, "", "  ")
		if err != nil {
			_ = zw.Close()
			return nil, err
		}
		if _, err := w.Write(append(data, '\n')); err != nil {
			_ = zw.Close()
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (s *ChatGPTAccountPoolService) refreshOneAccount(ctx context.Context, acc *Account, cfg ChatGPTAccountPoolConfig, forceTokenRefresh bool) (bool, error) {
	if acc == nil {
		return false, ErrAccountNotFound
	}
	if acc.Credentials == nil {
		acc.Credentials = map[string]any{}
	}
	if acc.Extra == nil {
		acc.Extra = map[string]any{}
	}
	accessToken := acc.GetCredential("access_token")
	if accessToken == "" {
		return false, fmt.Errorf("missing access_token")
	}
	if (forceTokenRefresh || ChatGPTAccessTokenNeedsRefresh(accessToken, time.Now())) && strings.TrimSpace(acc.GetCredential("refresh_token")) != "" {
		if err := s.refreshAccessToken(ctx, acc); err != nil {
			recordChatGPTTokenRefreshError(acc, err)
			_ = s.accounts.Update(ctx, acc)
		} else {
			accessToken = acc.GetCredential("access_token")
		}
	}
	info, err := s.remote.FetchRemoteInfo(ctx, accessToken, acc)
	if err != nil && errors.Is(err, ErrChatGPTAccountPoolInvalidAccessToken) && strings.TrimSpace(acc.GetCredential("refresh_token")) != "" {
		if refreshErr := s.refreshAccessToken(ctx, acc); refreshErr == nil {
			accessToken = acc.GetCredential("access_token")
			info, err = s.remote.FetchRemoteInfo(ctx, accessToken, acc)
		} else {
			recordChatGPTTokenRefreshError(acc, refreshErr)
		}
	}
	if err != nil {
		return false, s.handleRefreshError(ctx, acc, cfg, err)
	}
	applyChatGPTRemoteInfo(acc, info)
	applyChatGPTTokenTimestampsWithOverwrite(acc, true)
	if acc.Status == "限流" && cfg.AutoRemoveRateLimitedAccounts {
		return false, s.accounts.Delete(ctx, acc.ID)
	}
	if err := s.accounts.Update(ctx, acc); err != nil {
		return false, err
	}
	return true, nil
}

func (s *ChatGPTAccountPoolService) refreshAccessToken(ctx context.Context, acc *Account) error {
	if s.remote == nil {
		return fmt.Errorf("chatgpt account pool remote client is not configured")
	}
	refreshToken := strings.TrimSpace(acc.GetCredential("refresh_token"))
	if refreshToken == "" {
		return fmt.Errorf("missing refresh_token")
	}
	data, err := s.remote.RefreshAccessToken(ctx, refreshToken)
	if err != nil {
		return err
	}
	if data == nil || strings.TrimSpace(data.AccessToken) == "" {
		return fmt.Errorf("refresh response missing access_token")
	}
	acc.Credentials["access_token"] = strings.TrimSpace(data.AccessToken)
	if strings.TrimSpace(data.RefreshToken) != "" {
		acc.Credentials["refresh_token"] = strings.TrimSpace(data.RefreshToken)
	}
	if strings.TrimSpace(data.IDToken) != "" {
		acc.Credentials["id_token"] = strings.TrimSpace(data.IDToken)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	acc.Extra["last_token_refresh_at"] = now
	acc.Extra["last_token_refresh_error"] = nil
	acc.Extra["last_token_refresh_error_at"] = nil
	acc.Extra["invalid_count"] = 0
	acc.Extra["last_invalid_at"] = nil
	applyChatGPTTokenTimestampsWithOverwrite(acc, true)
	return nil
}

func (s *ChatGPTAccountPoolService) handleRefreshError(ctx context.Context, acc *Account, cfg ChatGPTAccountPoolConfig, err error) error {
	now := time.Now().UTC().Format(time.RFC3339)
	msg := truncateChatGPTPoolString(err.Error(), 500)
	acc.Extra["last_refresh_error"] = msg
	acc.Extra["last_refresh_error_at"] = now
	if errors.Is(err, ErrChatGPTAccountPoolInvalidAccessToken) {
		acc.Extra["invalid_count"] = intValue(acc.Extra["invalid_count"]) + 1
		acc.Extra["last_invalid_at"] = now
		if cfg.AutoRemoveInvalidAccounts {
			return s.accounts.Delete(ctx, acc.ID)
		}
		acc.Status = "异常"
		acc.Extra["quota"] = 0
	}
	_ = s.accounts.Update(ctx, acc)
	return err
}

func applyChatGPTRemoteInfo(acc *Account, info *ChatGPTAccountPoolRemoteInfo) {
	if acc == nil || info == nil {
		return
	}
	if acc.Credentials == nil {
		acc.Credentials = map[string]any{}
	}
	if acc.Extra == nil {
		acc.Extra = map[string]any{}
	}
	setCredentialIfNotEmpty(acc, "email", info.Email)
	setCredentialIfNotEmpty(acc, "user_id", info.UserID)
	setCredentialIfNotEmpty(acc, "account_id", info.AccountID)
	setCredentialIfNotEmpty(acc, "chatgpt_account_id", info.AccountID)
	if info.Email != "" {
		acc.Name = info.Email
	}
	plan := NormalizeChatGPTPlanType(info.PlanType)
	if plan == "" {
		plan = "free"
	}
	acc.Extra["plan_type"] = plan
	acc.Extra["quota"] = maxInt(0, info.Quota)
	acc.Extra["image_quota_unknown"] = info.ImageQuotaUnknown
	if info.LimitsProgress != nil {
		acc.Extra["limits_progress"] = info.LimitsProgress
	}
	if info.DefaultModelSlug != "" {
		acc.Extra["default_model_slug"] = info.DefaultModelSlug
	}
	if info.RestoreAt != "" {
		acc.Extra["restore_at"] = info.RestoreAt
	}
	status := strings.TrimSpace(info.Status)
	if status == "" {
		if info.ImageQuotaUnknown && strings.ToLower(plan) != "free" {
			status = "正常"
		} else if info.Quota <= 0 {
			status = "限流"
		} else {
			status = "正常"
		}
	}
	acc.Status = status
	acc.Extra["last_refresh_error"] = nil
	acc.Extra["last_refresh_error_at"] = nil
	acc.Extra["invalid_count"] = 0
	acc.Extra["last_invalid_at"] = nil
}

func applyChatGPTTokenTimestamps(acc *Account) {
	applyChatGPTTokenTimestampsWithOverwrite(acc, false)
}

func applyChatGPTTokenTimestampsWithOverwrite(acc *Account, overwrite bool) {
	if acc == nil || acc.Credentials == nil {
		return
	}
	payload := DecodeJWTPayload(acc.GetCredential("access_token"))
	if exp := timestampToISO(payload["exp"]); exp != "" && (overwrite || acc.GetCredential("expired") == "") {
		acc.Credentials["expired"] = exp
	}
	if iat := timestampToISO(payload["iat"]); iat != "" && (overwrite || acc.GetCredential("last_refresh") == "") {
		acc.Credentials["last_refresh"] = iat
	}
}

func recordChatGPTTokenRefreshError(acc *Account, err error) {
	if acc == nil {
		return
	}
	if acc.Extra == nil {
		acc.Extra = map[string]any{}
	}
	now := time.Now().UTC().Format(time.RFC3339)
	acc.Extra["last_token_refresh_error"] = truncateChatGPTPoolString(err.Error(), 500)
	acc.Extra["last_token_refresh_error_at"] = now
}

func ChatGPTAccountToPoolItem(acc *Account) map[string]any {
	item := map[string]any{}
	if acc == nil {
		return item
	}
	credKeys := []string{"access_token", "refresh_token", "id_token", "password", "email", "user_id", "account_id", "expired", "last_refresh"}
	for _, key := range credKeys {
		item[key] = acc.GetCredential(key)
	}
	if item["account_id"] == "" {
		item["account_id"] = acc.GetCredential("chatgpt_account_id")
	}
	extraKeys := []string{"openai_backend_mode", "export_type", "plan_type", "quota", "image_quota_unknown", "limits_progress", "default_model_slug", "restore_at", "success", "fail", "invalid_count", "last_invalid_at", "last_refresh_error", "last_refresh_error_at", "last_token_refresh_at", "last_token_refresh_error", "last_token_refresh_error_at"}
	for _, key := range extraKeys {
		if acc.Extra != nil {
			if v, ok := acc.Extra[key]; ok {
				item[key] = v
				continue
			}
		}
		item[key] = defaultChatGPTPoolExtraValue(key)
	}
	planType := strings.TrimSpace(chatGPTPoolStringFromAny(item["plan_type"]))
	item["type"] = chatGPTPoolFirstNonEmpty(planType, acc.Type)
	item["account_type"] = acc.Type
	item["status"] = acc.Status
	item["name"] = acc.Name
	item["created_at"] = acc.CreatedAt.UTC().Format(time.RFC3339)
	item["updated_at"] = acc.UpdatedAt.UTC().Format(time.RFC3339)
	return item
}

func defaultChatGPTPoolExtraValue(key string) any {
	switch key {
	case "openai_backend_mode":
		return "chatgpt_web"
	case "quota", "success", "fail", "invalid_count":
		return 0
	case "image_quota_unknown":
		return false
	case "limits_progress":
		return []any{}
	default:
		return nil
	}
}

func prepareChatGPTAccountPayload(item map[string]any) map[string]any {
	if item == nil {
		return nil
	}
	accessToken := chatGPTPoolFirstNonEmpty(chatGPTPoolStringFromAny(item["access_token"]), chatGPTPoolStringFromAny(item["accessToken"]))
	if accessToken == "" {
		return nil
	}
	payload := chatGPTPoolCopyMap(item)
	delete(payload, "accessToken")
	payload["access_token"] = accessToken
	if strings.EqualFold(strings.TrimSpace(chatGPTPoolStringFromAny(payload["type"])), "codex") {
		payload["export_type"] = "codex"
		delete(payload, "type")
	}
	if chatGPTPoolStringFromAny(payload["plan_type"]) != "" && chatGPTPoolStringFromAny(payload["type"]) == "" {
		payload["type"] = chatGPTPoolStringFromAny(payload["plan_type"])
	}
	return payload
}

func (s *ChatGPTAccountPoolService) createChatGPTAccount(ctx context.Context, payload map[string]any) error {
	accessToken := chatGPTPoolStringFromAny(payload["access_token"])
	if accessToken == "" {
		return fmt.Errorf("access_token is required")
	}
	creds := map[string]any{"access_token": accessToken}
	for _, key := range []string{"refresh_token", "id_token", "password", "email", "user_id", "account_id", "expired", "last_refresh"} {
		if v, ok := payload[key]; ok && !isEmptyAny(v) {
			creds[key] = v
		}
	}
	if v, ok := creds["account_id"]; ok && !isEmptyAny(v) {
		creds["chatgpt_account_id"] = v
	}
	accType := AccountTypeOAuth
	planType := NormalizeChatGPTPlanType(chatGPTPoolFirstNonEmpty(chatGPTPoolStringFromAny(payload["plan_type"]), chatGPTPoolStringFromAny(payload["type"])))
	if isNativeAccountType(planType) {
		accType = planType
		planType = ""
	}
	extra := map[string]any{"openai_backend_mode": "chatgpt_web"}
	if planType != "" {
		extra["plan_type"] = planType
	}
	for _, key := range []string{"export_type", "quota", "image_quota_unknown", "limits_progress", "default_model_slug", "restore_at", "success", "fail", "invalid_count", "last_invalid_at", "last_refresh_error", "last_refresh_error_at", "last_token_refresh_at", "last_token_refresh_error", "last_token_refresh_error_at"} {
		if v, ok := payload[key]; ok {
			extra[key] = v
		}
	}
	if _, ok := extra["quota"]; !ok {
		extra["quota"] = 0
	}
	if _, ok := extra["image_quota_unknown"]; !ok {
		extra["image_quota_unknown"] = false
	}
	if _, ok := extra["success"]; !ok {
		extra["success"] = 0
	}
	if _, ok := extra["fail"]; !ok {
		extra["fail"] = 0
	}
	name := chatGPTPoolFirstNonEmpty(chatGPTPoolStringFromAny(creds["email"]), "ChatGPT-"+prefixString(accessToken, 8))
	acc := &Account{Name: name, Platform: PlatformOpenAI, Type: accType, Credentials: creds, Extra: extra, Status: StatusActive, Schedulable: true, Concurrency: 3, Priority: 50}
	applyChatGPTTokenTimestamps(acc)
	return s.accounts.Create(ctx, acc)
}

func (s *ChatGPTAccountPoolService) listChatGPTAccounts(ctx context.Context) ([]Account, error) {
	if s == nil || s.accounts == nil {
		return nil, fmt.Errorf("account repository is not configured")
	}
	accounts, err := s.accounts.ListByPlatform(ctx, PlatformOpenAI)
	if err != nil {
		return nil, err
	}
	out := make([]Account, 0, len(accounts))
	for i := range accounts {
		if IsChatGPTWebPoolAccount(&accounts[i]) {
			out = append(out, accounts[i])
		}
	}
	return out, nil
}

func IsChatGPTWebPoolAccount(acc *Account) bool {
	if acc == nil || acc.Platform != PlatformOpenAI || acc.Extra == nil {
		return false
	}
	return strings.TrimSpace(fmt.Sprint(acc.Extra["openai_backend_mode"])) == "chatgpt_web"
}

func (s *ChatGPTAccountPoolService) chatgptAccountTokenSet(ctx context.Context) (map[string]bool, error) {
	accounts, err := s.listChatGPTAccounts(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[string]bool, len(accounts))
	for i := range accounts {
		if token := accounts[i].GetCredential("access_token"); token != "" {
			result[token] = true
		}
	}
	return result, nil
}

func (s *ChatGPTAccountPoolService) chatgptAccountTokenIDs(ctx context.Context) (map[string]int64, error) {
	accounts, err := s.listChatGPTAccounts(ctx)
	if err != nil {
		return nil, err
	}
	result := make(map[string]int64, len(accounts))
	for i := range accounts {
		if token := accounts[i].GetCredential("access_token"); token != "" {
			result[token] = accounts[i].ID
		}
	}
	return result, nil
}

func (s *ChatGPTAccountPoolService) findByAccessToken(ctx context.Context, token string) (*Account, error) {
	token = strings.TrimSpace(token)
	if token == "" {
		return nil, nil
	}
	ids, err := s.chatgptAccountTokenIDs(ctx)
	if err != nil {
		return nil, err
	}
	id, ok := ids[token]
	if !ok {
		return nil, nil
	}
	return s.accounts.GetByID(ctx, id)
}

func (s *ChatGPTAccountPoolService) selectRefreshAccounts(ctx context.Context, tokens []string) ([]Account, error) {
	all, err := s.listChatGPTAccounts(ctx)
	if err != nil {
		return nil, err
	}
	if len(tokens) == 0 {
		return all, nil
	}
	targets := map[string]struct{}{}
	for _, token := range tokens {
		if token = strings.TrimSpace(token); token != "" {
			targets[token] = struct{}{}
		}
	}
	out := make([]Account, 0, len(targets))
	for i := range all {
		if _, ok := targets[all[i].GetCredential("access_token")]; ok {
			out = append(out, all[i])
		}
	}
	return out, nil
}

func NormalizeChatGPTPlanType(value string) string {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return ""
	}
	key := strings.ToLower(strings.NewReplacer("-", "_", " ", "_").Replace(raw))
	compact := strings.ReplaceAll(key, "_", "")
	aliases := map[string]string{"free": "free", "plus": "Plus", "pro": "Pro", "prolite": "ProLite", "team": "Team", "business": "Team", "enterprise": "Enterprise"}
	if v := aliases[compact]; v != "" {
		return v
	}
	if v := aliases[key]; v != "" {
		return v
	}
	return raw
}

func isNativeAccountType(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case AccountTypeOAuth, AccountTypeAPIKey, AccountTypeSetupToken:
		return true
	default:
		return false
	}
}

func DecodeJWTPayload(token string) map[string]any {
	parts := strings.Split(token, ".")
	if len(parts) < 2 {
		return map[string]any{}
	}
	payload := parts[1]
	payload += strings.Repeat("=", (4-len(payload)%4)%4)
	data, err := base64.URLEncoding.DecodeString(payload)
	if err != nil {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func ChatGPTAccessTokenNeedsRefresh(token string, now time.Time) bool {
	payload := DecodeJWTPayload(token)
	exp := int64FromAny(payload["exp"])
	if exp <= 0 {
		return false
	}
	if now.IsZero() {
		now = time.Now()
	}
	return time.Unix(exp, 0).Sub(now) <= chatGPTAccountPoolRefreshSkew
}

func timestampToISO(value any) string {
	ts := int64FromAny(value)
	if ts <= 0 {
		return ""
	}
	return time.Unix(ts, 0).UTC().Format(time.RFC3339)
}

func setCredentialIfNotEmpty(acc *Account, key, value string) {
	if strings.TrimSpace(value) != "" {
		acc.Credentials[key] = strings.TrimSpace(value)
	}
}

func chatGPTPoolFirstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func chatGPTPoolCopyMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func chatGPTPoolStringFromAny(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case json.Number:
		return v.String()
	default:
		return strings.TrimSpace(fmt.Sprint(v))
	}
}

func intFromAny(value any) (int, bool) {
	switch v := value.(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		return int(v), true
	case float32:
		return int(v), true
	case json.Number:
		i, err := v.Int64()
		return int(i), err == nil
	case string:
		i, err := strconv.Atoi(strings.TrimSpace(v))
		return i, err == nil
	default:
		return 0, false
	}
}

func intValue(value any) int {
	v, _ := intFromAny(value)
	return v
}

func int64FromAny(value any) int64 {
	switch v := value.(type) {
	case int:
		return int64(v)
	case int64:
		return v
	case float64:
		return int64(v)
	case float32:
		return int64(v)
	case json.Number:
		i, _ := v.Int64()
		return i
	case string:
		i, _ := strconv.ParseInt(strings.TrimSpace(v), 10, 64)
		return i
	default:
		return 0
	}
}

func boolFromAny(value any) (bool, bool) {
	switch v := value.(type) {
	case bool:
		return v, true
	case string:
		switch strings.ToLower(strings.TrimSpace(v)) {
		case "1", "true", "yes", "on":
			return true, true
		case "0", "false", "no", "off":
			return false, true
		}
	}
	return false, false
}

func isEmptyAny(value any) bool {
	if value == nil {
		return true
	}
	if s, ok := value.(string); ok {
		return strings.TrimSpace(s) == ""
	}
	return false
}

func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func prefixString(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n]
}

func truncateChatGPTPoolString(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max]
}

func AnonymizeTokenForChatGPTPool(token string) string {
	token = strings.TrimSpace(token)
	if len(token) <= 12 {
		return token
	}
	return token[:6] + "..." + token[len(token)-4:]
}

var chatGPTExportSafeNameRE = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func safeChatGPTExportName(value, fallback string) string {
	clean := chatGPTExportSafeNameRE.ReplaceAllString(strings.TrimSpace(value), "-")
	clean = strings.Trim(clean, "-._")
	if clean == "" {
		clean = fallback
	}
	if len(clean) > 80 {
		clean = clean[:80]
	}
	return clean
}

// ChatGPTAccountPoolHTTPClient is a small, injectable implementation of the ChatGPT Web info and OAuth refresh calls.
type ChatGPTAccountPoolHTTPClient struct {
	HTTPClient *http.Client
	BaseURL    string
	OAuthURL   string
	UserAgent  string
}

func NewChatGPTAccountPoolHTTPClient(client *http.Client) *ChatGPTAccountPoolHTTPClient {
	if client == nil {
		client = http.DefaultClient
	}
	return &ChatGPTAccountPoolHTTPClient{HTTPClient: client, BaseURL: chatGPTWebBaseURL, OAuthURL: chatGPTAccountPoolOAuthTokenURL, UserAgent: chatGPTWebDefaultUserAgent}
}

func (c *ChatGPTAccountPoolHTTPClient) tlsDo(req *http.Request) (*http.Response, []byte, error) {
	proxyURL := chatGPTRegisterTLSProxyURL("")
	opts := []tlsclient.HttpClientOption{
		tlsclient.WithTimeoutSeconds(60),
		tlsclient.WithClientProfile(tlsclientprofiles.Chrome_144),
	}
	if proxyURL != "" {
		opts = append(opts, tlsclient.WithProxyUrl(proxyURL))
	}
	cli, err := tlsclient.NewHttpClient(tlsclient.NewNoopLogger(), opts...)
	if err != nil {
		return nil, nil, err
	}

	fReq, err := fhttp.NewRequestWithContext(req.Context(), req.Method, req.URL.String(), req.Body)
	if err != nil {
		return nil, nil, err
	}
	for k, vals := range req.Header {
		for _, v := range vals {
			fReq.Header.Add(k, v)
		}
	}
	fResp, err := cli.Do(fReq)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = fResp.Body.Close() }()
	data, _ := io.ReadAll(io.LimitReader(fResp.Body, 4<<20))
	return &http.Response{StatusCode: fResp.StatusCode, Header: http.Header(fResp.Header), Body: io.NopCloser(bytes.NewReader(data))}, data, nil
}

func (c *ChatGPTAccountPoolHTTPClient) RefreshAccessToken(ctx context.Context, refreshToken string) (*ChatGPTAccountPoolTokenData, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("refresh_token", refreshToken)
	form.Set("client_id", chatGPTAccountPoolOAuthClientID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.OAuthURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", chatGPTPoolFirstNonEmpty(c.UserAgent, chatGPTWebDefaultUserAgent))
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	var data map[string]any
	_ = json.Unmarshal(body, &data)
	accessToken := chatGPTPoolStringFromAny(data["access_token"])
	if resp.StatusCode != http.StatusOK || accessToken == "" {
		detail := chatGPTPoolFirstNonEmpty(chatGPTPoolStringFromAny(data["error_description"]), chatGPTPoolStringFromAny(data["error"]), chatGPTPoolStringFromAny(data["message"]), string(body))
		return nil, fmt.Errorf("oauth_refresh_http_%d: %s", resp.StatusCode, truncateChatGPTPoolString(detail, 300))
	}
	return &ChatGPTAccountPoolTokenData{AccessToken: accessToken, RefreshToken: chatGPTPoolFirstNonEmpty(chatGPTPoolStringFromAny(data["refresh_token"]), refreshToken), IDToken: chatGPTPoolStringFromAny(data["id_token"])}, nil
}

func (c *ChatGPTAccountPoolHTTPClient) FetchRemoteInfo(ctx context.Context, accessToken string, account *Account) (*ChatGPTAccountPoolRemoteInfo, error) {
	me, err := c.doJSON(ctx, http.MethodGet, "/backend-api/me", accessToken, nil)
	if err != nil {
		return nil, err
	}
	initPayload, err := c.doJSON(ctx, http.MethodPost, "/backend-api/conversation/init", accessToken, map[string]any{"gizmo_id": nil, "requested_default_model": nil, "conversation_id": nil, "timezone_offset_min": -480})
	if err != nil {
		return nil, err
	}
	accountPayload, err := c.doJSON(ctx, http.MethodGet, "/backend-api/accounts/check/v4-2023-04-27?timezone_offset_min=-480", accessToken, nil)
	if err != nil {
		return nil, err
	}
	defaultAccount := map[string]any{}
	if accounts, _ := accountPayload["accounts"].(map[string]any); accounts != nil {
		if def, _ := accounts["default"].(map[string]any); def != nil {
			defaultAccount, _ = def["account"].(map[string]any)
		}
	}
	limits := initPayload["limits_progress"]
	limitsList, _ := limits.([]any)
	quota, restoreAt, unknown := extractChatGPTImageQuota(limitsList)
	plan := chatGPTPoolFirstNonEmpty(chatGPTPoolStringFromAny(defaultAccount["plan_type"]), "free")
	status := "正常"
	if !unknown && quota == 0 {
		status = "限流"
	}
	return &ChatGPTAccountPoolRemoteInfo{Email: chatGPTPoolStringFromAny(me["email"]), UserID: chatGPTPoolStringFromAny(me["id"]), AccountID: chatGPTPoolStringFromAny(defaultAccount["id"]), PlanType: plan, Quota: quota, ImageQuotaUnknown: unknown, LimitsProgress: limitsList, DefaultModelSlug: chatGPTPoolStringFromAny(initPayload["default_model_slug"]), RestoreAt: restoreAt, Status: status}, nil
}

func (c *ChatGPTAccountPoolHTTPClient) doJSON(ctx context.Context, method, path, accessToken string, payload map[string]any) (map[string]any, error) {
	var body io.Reader
	if payload != nil {
		data, _ := json.Marshal(payload)
		body = bytes.NewReader(data)
	}
	base := strings.TrimRight(chatGPTPoolFirstNonEmpty(c.BaseURL, chatGPTWebBaseURL), "/")
	urlStr := base + path
	if strings.HasPrefix(path, "http://") || strings.HasPrefix(path, "https://") {
		urlStr = path
	}
	req, err := http.NewRequestWithContext(ctx, method, urlStr, body)
	if err != nil {
		return nil, err
	}
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", chatGPTPoolFirstNonEmpty(c.UserAgent, chatGPTWebDefaultUserAgent))
	req.Header.Set("Authorization", "Bearer "+accessToken)
	targetPath := path
	if u, err := url.Parse(path); err == nil && u.Path != "" {
		targetPath = u.Path
	}
	req.Header.Set("X-OpenAI-Target-Path", targetPath)
	req.Header.Set("X-OpenAI-Target-Route", targetPath)
	resp, data, err := c.tlsDo(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode == http.StatusUnauthorized {
		return nil, ErrChatGPTAccountPoolInvalidAccessToken
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("%s failed: HTTP %d: %s", targetPath, resp.StatusCode, truncateChatGPTPoolString(string(data), 300))
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func extractChatGPTImageQuota(limits []any) (int, string, bool) {
	for _, item := range limits {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if chatGPTPoolStringFromAny(m["feature_name"]) == "image_gen" {
			return maxInt(0, intValue(m["remaining"])), chatGPTPoolStringFromAny(m["reset_after"]), false
		}
	}
	return 0, "", true
}

func ChatGPTAccountExportContentDisposition(format string, now time.Time) string {
	ext := "json"
	if strings.EqualFold(format, "zip") {
		ext = "zip"
	}
	return mime.FormatMediaType("attachment", map[string]string{"filename": fmt.Sprintf("codex-accounts-%s.%s", now.Format("20060102-150405"), ext)})
}
