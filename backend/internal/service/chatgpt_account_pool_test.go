package service

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

type chatGPTPoolAccountRepoStub struct {
	accounts []Account
	deleted  []int64
	nextID   int64
}

func (r *chatGPTPoolAccountRepoStub) Create(_ context.Context, account *Account) error {
	r.nextID++
	cp := *account
	cp.ID = r.nextID
	if cp.CreatedAt.IsZero() {
		cp.CreatedAt = time.Now()
	}
	cp.UpdatedAt = cp.CreatedAt
	r.accounts = append(r.accounts, cp)
	account.ID = cp.ID
	return nil
}
func (r *chatGPTPoolAccountRepoStub) GetByID(_ context.Context, id int64) (*Account, error) {
	for i := range r.accounts {
		if r.accounts[i].ID == id {
			cp := r.accounts[i]
			return &cp, nil
		}
	}
	return nil, ErrAccountNotFound
}
func (r *chatGPTPoolAccountRepoStub) GetByIDs(_ context.Context, ids []int64) ([]*Account, error) {
	return nil, nil
}
func (r *chatGPTPoolAccountRepoStub) ExistsByID(_ context.Context, id int64) (bool, error) {
	return true, nil
}
func (r *chatGPTPoolAccountRepoStub) GetByCRSAccountID(_ context.Context, crsAccountID string) (*Account, error) {
	return nil, nil
}
func (r *chatGPTPoolAccountRepoStub) FindByExtraField(_ context.Context, key string, value any) ([]Account, error) {
	return nil, nil
}
func (r *chatGPTPoolAccountRepoStub) ListCRSAccountIDs(_ context.Context) (map[string]int64, error) {
	return nil, nil
}
func (r *chatGPTPoolAccountRepoStub) Update(_ context.Context, account *Account) error {
	for i := range r.accounts {
		if r.accounts[i].ID == account.ID {
			cp := *account
			cp.UpdatedAt = time.Now()
			r.accounts[i] = cp
			return nil
		}
	}
	return ErrAccountNotFound
}
func (r *chatGPTPoolAccountRepoStub) Delete(_ context.Context, id int64) error {
	r.deleted = append(r.deleted, id)
	for i := range r.accounts {
		if r.accounts[i].ID == id {
			r.accounts = append(r.accounts[:i], r.accounts[i+1:]...)
			return nil
		}
	}
	return nil
}
func (r *chatGPTPoolAccountRepoStub) List(_ context.Context, params pagination.PaginationParams) ([]Account, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (r *chatGPTPoolAccountRepoStub) ListWithFilters(_ context.Context, params pagination.PaginationParams, platform, accountType, status, search string, groupID int64, privacyMode string) ([]Account, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (r *chatGPTPoolAccountRepoStub) ListByGroup(_ context.Context, groupID int64) ([]Account, error) {
	return nil, nil
}
func (r *chatGPTPoolAccountRepoStub) ListActive(_ context.Context) ([]Account, error) {
	return nil, nil
}
func (r *chatGPTPoolAccountRepoStub) ListByPlatform(_ context.Context, platform string) ([]Account, error) {
	out := []Account{}
	for _, acc := range r.accounts {
		if acc.Platform == platform {
			out = append(out, acc)
		}
	}
	return out, nil
}
func (r *chatGPTPoolAccountRepoStub) UpdateLastUsed(context.Context, int64) error { return nil }
func (r *chatGPTPoolAccountRepoStub) BatchUpdateLastUsed(context.Context, map[int64]time.Time) error {
	return nil
}
func (r *chatGPTPoolAccountRepoStub) SetError(context.Context, int64, string) error     { return nil }
func (r *chatGPTPoolAccountRepoStub) ClearError(context.Context, int64) error           { return nil }
func (r *chatGPTPoolAccountRepoStub) SetSchedulable(context.Context, int64, bool) error { return nil }
func (r *chatGPTPoolAccountRepoStub) AutoPauseExpiredAccounts(context.Context, time.Time) (int64, error) {
	return 0, nil
}
func (r *chatGPTPoolAccountRepoStub) BindGroups(context.Context, int64, []int64) error { return nil }
func (r *chatGPTPoolAccountRepoStub) ListSchedulable(context.Context) ([]Account, error) {
	return nil, nil
}
func (r *chatGPTPoolAccountRepoStub) ListSchedulableByGroupID(context.Context, int64) ([]Account, error) {
	return nil, nil
}
func (r *chatGPTPoolAccountRepoStub) ListSchedulableByPlatform(context.Context, string) ([]Account, error) {
	return nil, nil
}
func (r *chatGPTPoolAccountRepoStub) ListSchedulableByGroupIDAndPlatform(context.Context, int64, string) ([]Account, error) {
	return nil, nil
}
func (r *chatGPTPoolAccountRepoStub) ListSchedulableByPlatforms(context.Context, []string) ([]Account, error) {
	return nil, nil
}
func (r *chatGPTPoolAccountRepoStub) ListSchedulableByGroupIDAndPlatforms(context.Context, int64, []string) ([]Account, error) {
	return nil, nil
}
func (r *chatGPTPoolAccountRepoStub) ListSchedulableUngroupedByPlatform(context.Context, string) ([]Account, error) {
	return nil, nil
}
func (r *chatGPTPoolAccountRepoStub) ListSchedulableUngroupedByPlatforms(context.Context, []string) ([]Account, error) {
	return nil, nil
}
func (r *chatGPTPoolAccountRepoStub) SetRateLimited(context.Context, int64, time.Time) error {
	return nil
}
func (r *chatGPTPoolAccountRepoStub) SetModelRateLimit(context.Context, int64, string, time.Time, ...string) error {
	return nil
}
func (r *chatGPTPoolAccountRepoStub) SetOverloaded(context.Context, int64, time.Time) error {
	return nil
}
func (r *chatGPTPoolAccountRepoStub) SetTempUnschedulable(context.Context, int64, time.Time, string) error {
	return nil
}
func (r *chatGPTPoolAccountRepoStub) ClearTempUnschedulable(context.Context, int64) error { return nil }
func (r *chatGPTPoolAccountRepoStub) ClearRateLimit(context.Context, int64) error         { return nil }
func (r *chatGPTPoolAccountRepoStub) ClearAntigravityQuotaScopes(context.Context, int64) error {
	return nil
}
func (r *chatGPTPoolAccountRepoStub) ClearModelRateLimits(context.Context, int64) error { return nil }
func (r *chatGPTPoolAccountRepoStub) UpdateSessionWindow(context.Context, int64, *time.Time, *time.Time, string) error {
	return nil
}
func (r *chatGPTPoolAccountRepoStub) UpdateExtra(_ context.Context, id int64, updates map[string]any) error {
	for i := range r.accounts {
		if r.accounts[i].ID == id {
			if r.accounts[i].Extra == nil {
				r.accounts[i].Extra = map[string]any{}
			}
			for k, v := range updates {
				r.accounts[i].Extra[k] = v
			}
		}
	}
	return nil
}
func (r *chatGPTPoolAccountRepoStub) BulkUpdate(context.Context, []int64, AccountBulkUpdate) (int64, error) {
	return 0, nil
}
func (r *chatGPTPoolAccountRepoStub) IncrementQuotaUsed(context.Context, int64, float64) error {
	return nil
}
func (r *chatGPTPoolAccountRepoStub) ResetQuotaUsed(context.Context, int64) error { return nil }

type chatGPTPoolRemoteStub struct {
	refreshed bool
	info      *ChatGPTAccountPoolRemoteInfo
}

func (r *chatGPTPoolRemoteStub) RefreshAccessToken(_ context.Context, refreshToken string) (*ChatGPTAccountPoolTokenData, error) {
	r.refreshed = true
	return &ChatGPTAccountPoolTokenData{AccessToken: makeChatGPTPoolJWT(time.Now().Add(48*time.Hour), time.Now()), RefreshToken: refreshToken, IDToken: makeChatGPTPoolJWT(time.Now().Add(48*time.Hour), time.Now())}, nil
}
func (r *chatGPTPoolRemoteStub) FetchRemoteInfo(_ context.Context, accessToken string, account *Account) (*ChatGPTAccountPoolRemoteInfo, error) {
	return r.info, nil
}

func TestChatGPTAccountPoolHTTPClientDoJSONUsesTLSClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/backend-api/me" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Header.Get("Authorization") != "Bearer access-token" {
			t.Fatalf("authorization header = %q", r.Header.Get("Authorization"))
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"email": "u@example.test"})
	}))
	defer server.Close()

	client := NewChatGPTAccountPoolHTTPClient(&http.Client{Transport: failingRoundTripper{}})
	client.BaseURL = server.URL
	out, err := client.doJSON(context.Background(), http.MethodGet, "/backend-api/me", "access-token", nil)
	if err != nil {
		t.Fatal(err)
	}
	if out["email"] != "u@example.test" {
		t.Fatalf("email = %v", out["email"])
	}
}

func TestChatGPTAccountPool_CreateListUpdateExportAndRefresh(t *testing.T) {
	repo := &chatGPTPoolAccountRepoStub{}
	settings := &chatGPTRegisterSettingRepoStub{values: map[string]string{}}
	svc := NewChatGPTAccountPoolService(repo, settings)
	expiring := makeChatGPTPoolJWT(time.Now().Add(time.Hour), time.Now().Add(-time.Hour))
	idToken := makeChatGPTPoolJWT(time.Now().Add(2*time.Hour), time.Now())

	created, err := svc.CreateAccounts(context.Background(), []string{"token-only"}, []map[string]any{{
		"accessToken":         expiring,
		"refresh_token":       "refresh-1",
		"id_token":            idToken,
		"password":            "pw",
		"email":               "a@example.com",
		"account_id":          "acc-import",
		"user_id":             "user-import",
		"expired":             "old-expired",
		"last_refresh":        "old-refresh",
		"type":                "codex",
		"plan_type":           "plus",
		"quota":               7,
		"image_quota_unknown": true,
		"limits_progress":     []any{map[string]any{"feature_name": "image_gen", "remaining": float64(7)}},
		"default_model_slug":  "gpt-5",
		"restore_at":          "2026-05-28T00:00:00Z",
		"success":             2,
		"fail":                1,
	}})
	if err != nil {
		t.Fatalf("CreateAccounts error: %v", err)
	}
	if created["added"] != 2 {
		t.Fatalf("added = %v, want 2", created["added"])
	}

	items, err := svc.ListAccounts(context.Background())
	if err != nil {
		t.Fatalf("ListAccounts error: %v", err)
	}
	var imported map[string]any
	for _, item := range items {
		if item["access_token"] == expiring {
			imported = item
		}
	}
	if imported == nil {
		t.Fatalf("imported account not listed: %#v", items)
	}
	if imported["refresh_token"] != "refresh-1" || imported["email"] != "a@example.com" || imported["account_id"] != "acc-import" || imported["expired"] != "old-expired" || imported["last_refresh"] != "old-refresh" {
		t.Fatalf("credentials not preserved: %#v", imported)
	}
	if imported["export_type"] != "codex" {
		t.Fatalf("export_type = %v, want codex", imported["export_type"])
	}
	if imported["type"] != "Plus" {
		t.Fatalf("type = %v, want Plus", imported["type"])
	}
	if imported["quota"] != 7 || imported["image_quota_unknown"] != true || imported["success"] != 2 || imported["fail"] != 1 {
		t.Fatalf("extra fields missing: %#v", imported)
	}

	updated, err := svc.UpdateAccount(context.Background(), expiring, map[string]any{"type": "Team", "status": "限流", "quota": 3, "image_quota_unknown": false})
	if err != nil {
		t.Fatalf("UpdateAccount error: %v", err)
	}
	if (*updated)["type"] != "Team" || (*updated)["status"] != "限流" || (*updated)["quota"] != 3 || (*updated)["image_quota_unknown"] != false {
		t.Fatalf("updated item mismatch: type=%v status=%v quota=%v unknown=%v", (*updated)["type"], (*updated)["status"], (*updated)["quota"], (*updated)["image_quota_unknown"])
	}

	exportItems, err := svc.BuildExportItems(context.Background(), []string{expiring})
	if err != nil {
		t.Fatalf("BuildExportItems error: %v", err)
	}
	if len(exportItems) != 1 {
		t.Fatalf("export items len = %d", len(exportItems))
	}
	if exportItems[0]["type"] != "codex" || exportItems[0]["password"] != "pw" || exportItems[0]["refresh_token"] != "refresh-1" {
		t.Fatalf("export item mismatch: %#v", exportItems[0])
	}
	jsonData, err := BuildChatGPTAccountExportJSON(exportItems)
	if err != nil {
		t.Fatalf("json export error: %v", err)
	}
	var jsonObj map[string]string
	if err := json.Unmarshal(jsonData, &jsonObj); err != nil {
		t.Fatalf("single json should be object: %v", err)
	}
	zipData, err := BuildChatGPTAccountExportZIP(exportItems)
	if err != nil {
		t.Fatalf("zip export error: %v", err)
	}
	zr, err := zip.NewReader(bytes.NewReader(zipData), int64(len(zipData)))
	if err != nil {
		t.Fatalf("zip open error: %v", err)
	}
	if len(zr.File) != 1 || !strings.Contains(zr.File[0].Name, "a-example.com") {
		t.Fatalf("zip names = %s", zr.File[0].Name)
	}
	f, err := zr.File[0].Open()
	if err != nil {
		t.Fatalf("open zip entry: %v", err)
	}
	defer func() { _ = f.Close() }()
	body, _ := io.ReadAll(f)
	if !bytes.Contains(body, []byte("refresh-1")) {
		t.Fatalf("zip json missing token: %s", string(body))
	}

	remote := &chatGPTPoolRemoteStub{info: &ChatGPTAccountPoolRemoteInfo{Email: "remote@example.com", UserID: "remote-user", AccountID: "remote-account", PlanType: "Pro", Quota: 9, ImageQuotaUnknown: false, LimitsProgress: []any{}, DefaultModelSlug: "gpt-5.1", RestoreAt: "2026-05-29T00:00:00Z", Status: "正常"}}
	svc.SetRemoteClient(remote)
	result, err := svc.RefreshAccounts(context.Background(), []string{expiring})
	if err != nil {
		t.Fatalf("RefreshAccounts error: %v", err)
	}
	if result["refreshed"] != 1 {
		t.Fatalf("refreshed = %v", result["refreshed"])
	}
	if !remote.refreshed {
		t.Fatalf("expected expiring token to refresh via refresh_token")
	}
	items, _ = svc.ListAccounts(context.Background())
	var refreshed map[string]any
	for _, item := range items {
		if item["email"] == "remote@example.com" {
			refreshed = item
		}
	}
	if refreshed == nil || refreshed["quota"] != 9 || refreshed["type"] != "Pro" || refreshed["account_id"] != "remote-account" {
		t.Fatalf("refresh fields mismatch: %#v", refreshed)
	}
}

func TestChatGPTAccountPoolConfig_NormalizesAndPersists(t *testing.T) {
	repo := &chatGPTRegisterSettingRepoStub{values: map[string]string{}}
	svc := NewChatGPTAccountPoolService(&chatGPTPoolAccountRepoStub{}, repo)
	cfg, err := svc.UpdateConfig(context.Background(), map[string]any{"refresh_account_interval_minute": float64(0), "image_account_concurrency": "0", "auto_remove_invalid_accounts": "true", "auto_remove_rate_limited_accounts": true})
	if err != nil {
		t.Fatalf("UpdateConfig error: %v", err)
	}
	if cfg.RefreshAccountIntervalMinute != 5 || cfg.ImageAccountConcurrency != 3 || !cfg.AutoRemoveInvalidAccounts || !cfg.AutoRemoveRateLimitedAccounts {
		t.Fatalf("normalized config mismatch: %#v", cfg)
	}
	if repo.values[ChatGPTAccountPoolSettingKey] == "" {
		t.Fatalf("config not persisted")
	}
}

func TestChatGPTRegisterComputeAccountStats_UsesPoolFields(t *testing.T) {
	quota, available := chatGPTRegisterComputeAccountStats([]Account{{Platform: PlatformOpenAI, Type: "free", Status: "正常", Extra: map[string]any{"openai_backend_mode": "chatgpt_web", "quota": float64(4)}}})
	if quota != 4 || available != 1 {
		t.Fatalf("stats = quota %d available %d", quota, available)
	}
}

func TestChatGPTRegisterComputeAccountStats_DefaultsFreeAccountQuota(t *testing.T) {
	quota, available := chatGPTRegisterComputeAccountStats([]Account{{Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Extra: map[string]any{"openai_backend_mode": "chatgpt_web", "plan_type": "free"}}})
	if quota != 5 || available != 1 {
		t.Fatalf("stats = quota %d available %d, want quota 5 available 1", quota, available)
	}
}

func TestChatGPTAccountToPoolItem_DefaultsFreeAccountQuota(t *testing.T) {
	item := ChatGPTAccountToPoolItem(&Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Extra: map[string]any{"openai_backend_mode": "chatgpt_web", "plan_type": "free"}})
	if item["quota"] != 5 {
		t.Fatalf("quota = %v, want 5", item["quota"])
	}
}

func makeChatGPTPoolJWT(exp, iat time.Time) string {
	header := base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"none"}`))
	payload, _ := json.Marshal(map[string]any{"exp": exp.Unix(), "iat": iat.Unix(), "email": "jwt@example.com", "https://api.openai.com/auth": map[string]any{"chatgpt_account_id": "jwt-account"}})
	return header + "." + base64.RawURLEncoding.EncodeToString(payload) + ".sig"
}
