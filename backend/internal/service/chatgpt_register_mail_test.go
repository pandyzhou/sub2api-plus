package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/pagination"
)

func TestChatGPTRegisterLegacyMailConfigMigratesToProviders(t *testing.T) {
	raw, _ := json.Marshal(ChatGPTRegisterConfig{MailProvider: "custom", MailAPIBase: "https://mail.example", MailAPIKey: "k"})
	repo := &chatGPTRegisterSettingRepoStub{values: map[string]string{chatGPTRegisterSettingKey: string(raw)}}
	svc := NewChatGPTRegisterService(repo, nil)
	cfg := svc.currentConfig()
	if len(cfg.Mail.Providers) != 1 {
		t.Fatalf("providers len=%d, want 1", len(cfg.Mail.Providers))
	}
	p := cfg.Mail.Providers[0]
	if p.Type != "mailtm" || !p.Enable || p.APIBase != "https://mail.example" || p.APIKey != "k" {
		t.Fatalf("legacy provider mismatch: %#v", p)
	}
}

func TestChatGPTRegisterProviderAndDomainRotation(t *testing.T) {
	chatGPTRegisterProviderIx = 0
	chatGPTRegisterDomainIx = 0
	mail := ChatGPTRegisterMailConfig{Providers: []ChatGPTRegisterMailProviderConfig{{Type: "mailtm", Enable: true}, {Type: "inbucket", Enable: true}}}
	first, err := chatGPTRegisterNextMailEntry(mail)
	if err != nil {
		t.Fatal(err)
	}
	second, err := chatGPTRegisterNextMailEntry(mail)
	if err != nil {
		t.Fatal(err)
	}
	if first.Type != "mailtm" || second.Type != "inbucket" {
		t.Fatalf("rotation = %s,%s; want mailtm,inbucket", first.Type, second.Type)
	}
	d1, _ := nextRegisterDomain([]string{"a.test", "b.test"})
	d2, _ := nextRegisterDomain([]string{"a.test", "b.test"})
	if d1 != "a.test" || d2 != "b.test" {
		t.Fatalf("domain rotation = %s,%s", d1, d2)
	}
}

func TestChatGPTRegisterExtractCodeFromHTMLTextAndSkips177010(t *testing.T) {
	if got := chatGPTRegisterExtractOTP(`<p style="background-color:#F3F3F3"> 654321 </p>`); got != "654321" {
		t.Fatalf("html code = %q", got)
	}
	if got := chatGPTRegisterExtractOTP("177010 is not it. Verification code: 123456"); got != "123456" {
		t.Fatalf("text code = %q", got)
	}
}

func TestChatGPTRegisterTargetReachedModes(t *testing.T) {
	repo := &chatGPTRegisterAccountRepoStub{accounts: []Account{
		{Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Extra: map[string]any{"openai_backend_mode": "chatgpt_web", "quota": 3}},
		{Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusActive, Extra: map[string]any{"openai_backend_mode": "chatgpt_web", "quota": 4}},
		{Platform: PlatformOpenAI, Type: AccountTypeOAuth, Status: StatusDisabled, Extra: map[string]any{"openai_backend_mode": "chatgpt_web", "quota": 100}},
	}}
	svc := NewChatGPTRegisterService(nil, repo)
	if !svc.targetReached(ChatGPTRegisterConfig{Mode: "total", Total: 2}, 2) {
		t.Fatal("total mode should use submitted >= total")
	}
	if !svc.targetReached(ChatGPTRegisterConfig{Mode: "quota", TargetQuota: 7}, 0) {
		t.Fatal("quota mode should use active ChatGPT Web quota")
	}
	if !svc.targetReached(ChatGPTRegisterConfig{Mode: "available", TargetAvail: 2}, 0) {
		t.Fatal("available mode should use active ChatGPT Web count")
	}
	got := svc.currentConfig().Stats
	if got.CurrentQuota != 7 || got.CurrentAvail != 2 {
		t.Fatalf("stats = quota %d avail %d", got.CurrentQuota, got.CurrentAvail)
	}
}

func TestChatGPTRegisterProxyHelperSupportsHTTPAndSocksErrorsUnsupported(t *testing.T) {
	if _, err := chatGPTRegisterHTTPClient("http://127.0.0.1:8080", time.Second); err != nil {
		t.Fatalf("http proxy should be accepted: %v", err)
	}
	if _, err := chatGPTRegisterHTTPClient("socks5://127.0.0.1:1080", time.Second); err != nil {
		t.Fatalf("socks5 proxy should be supported by x/net/proxy: %v", err)
	}
	if _, err := chatGPTRegisterHTTPClient("ftp://127.0.0.1:21", time.Second); err == nil || !strings.Contains(err.Error(), "unsupported proxy scheme") {
		t.Fatalf("unsupported proxy error = %v", err)
	}
}

func TestChatGPTRegisterTLSProxyURLFallsBackToEnvironment(t *testing.T) {
	t.Setenv("HTTP_PROXY", "")
	t.Setenv("HTTPS_PROXY", "http://127.0.0.1:7890")
	t.Setenv("NO_PROXY", "")

	got := chatGPTRegisterTLSProxyURL("")
	if got != "http://127.0.0.1:7890" {
		t.Fatalf("tls proxy fallback = %q, want env proxy", got)
	}
	if got := chatGPTRegisterTLSProxyURL("http://127.0.0.1:8080"); got != "http://127.0.0.1:8080" {
		t.Fatalf("explicit tls proxy = %q", got)
	}
}

func TestChatGPTRegisterRegisterUserUsesTLSClient(t *testing.T) {
	oldAuth, oldSentinel := chatGPTRegisterAuthBase, chatGPTRegisterSentinelBase
	defer func() {
		chatGPTRegisterAuthBase, chatGPTRegisterSentinelBase = oldAuth, oldSentinel
	}()
	var sawRegister bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/backend-api/sentinel/req":
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "sentinel-cookie"})
		case "/api/accounts/user/register":
			sawRegister = true
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	chatGPTRegisterAuthBase = server.URL
	chatGPTRegisterSentinelBase = server.URL
	client, err := newChatGPTRegisterOpenAIClient("", "device-id")
	if err != nil {
		t.Fatal(err)
	}
	client.http.Transport = failingRoundTripper{}
	if err := client.registerUser(context.Background(), "u@example.test", "Password1!"); err != nil {
		t.Fatal(err)
	}
	if !sawRegister {
		t.Fatal("register endpoint was not called")
	}
}

func TestChatGPTRegisterLoginTokenExchangeUsesLoginPKCEVerifier(t *testing.T) {
	oldAuth, oldPlatform, oldSentinel := chatGPTRegisterAuthBase, chatGPTRegisterPlatformBase, chatGPTRegisterSentinelBase
	defer func() {
		chatGPTRegisterAuthBase, chatGPTRegisterPlatformBase, chatGPTRegisterSentinelBase = oldAuth, oldPlatform, oldSentinel
	}()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/accounts/authorize":
			w.WriteHeader(http.StatusOK)
		case "/backend-api/sentinel/req":
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "sentinel-cookie"})
		case "/api/accounts/authorize/continue":
			_ = json.NewEncoder(w).Encode(map[string]any{})
		case "/api/accounts/password/verify":
			_ = json.NewEncoder(w).Encode(map[string]any{
				"type": "token_exchange",
				"continue_url": "http://unused.local/auth/callback?code=oauth-code",
				"payload": map[string]any{"code": "oauth-code", "state": "state-1"},
			})
		case "/oauth/token":
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			if r.Form.Get("code_verifier") == "registration-verifier" {
				w.WriteHeader(http.StatusBadRequest)
				_, _ = w.Write([]byte(`{"error":"used registration verifier"}`))
				return
			}
			if r.Form.Get("code") != "oauth-code" {
				t.Fatalf("code = %q", r.Form.Get("code"))
			}
			_ = json.NewEncoder(w).Encode(map[string]any{"access_token": "access", "refresh_token": "refresh", "id_token": "id", "expires_in": 3600})
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	chatGPTRegisterAuthBase = server.URL
	chatGPTRegisterPlatformBase = server.URL
	chatGPTRegisterSentinelBase = server.URL
	client, err := newChatGPTRegisterOpenAIClient("", "device-id")
	if err != nil {
		t.Fatal(err)
	}
	tokens, err := client.loginAndExchangeTokens(context.Background(), "u@example.test", "Password1!", "registration-verifier", nil, ChatGPTRegisterConfig{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if tokens.AccessToken != "access" || tokens.RefreshToken != "refresh" || tokens.IDToken != "id" {
		t.Fatalf("tokens = %#v", tokens)
	}
}

func TestChatGPTRegisterOpenAIHeadersAndSentinel(t *testing.T) {
	oldAuth, oldPlatform, oldSentinel := chatGPTRegisterAuthBase, chatGPTRegisterPlatformBase, chatGPTRegisterSentinelBase
	defer func() {
		chatGPTRegisterAuthBase, chatGPTRegisterPlatformBase, chatGPTRegisterSentinelBase = oldAuth, oldPlatform, oldSentinel
	}()
	var sawAuth0, sawSentinel, sawTrace bool
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/accounts/authorize":
			if r.URL.Query().Get("auth0Client") == chatGPTRegisterPlatformAuth0Client {
				sawAuth0 = true
			}
			w.WriteHeader(http.StatusOK)
		case "/backend-api/sentinel/req":
			_ = json.NewEncoder(w).Encode(map[string]any{"token": "sentinel-cookie"})
		case "/api/accounts/user/register":
			if r.Header.Get("Openai-Sentinel-Token") != "" || r.Header.Get("OpenAI-Sentinel-Token") != "" {
				sawSentinel = true
			}
			if r.Header.Get("Traceparent") != "" || r.Header.Get("traceparent") != "" {
				sawTrace = true
			}
			w.WriteHeader(http.StatusOK)
		default:
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
	}))
	defer server.Close()
	chatGPTRegisterAuthBase = server.URL
	chatGPTRegisterPlatformBase = server.URL
	chatGPTRegisterSentinelBase = server.URL
	client, err := newChatGPTRegisterOpenAIClient("", "device-id")
	if err != nil {
		t.Fatal(err)
	}
	if err := client.platformAuthorize(context.Background(), "u@example.test", "challenge", "state", "nonce"); err != nil {
		t.Fatal(err)
	}
	if err := client.registerUser(context.Background(), "u@example.test", "Password1!"); err != nil {
		t.Fatal(err)
	}
	if !sawAuth0 || !sawSentinel || !sawTrace {
		t.Fatalf("headers seen auth0=%v sentinel=%v trace=%v", sawAuth0, sawSentinel, sawTrace)
	}
}

type failingRoundTripper struct{}

func (f failingRoundTripper) RoundTrip(*http.Request) (*http.Response, error) {
	return nil, errors.New("standard http client should not be used")
}

type chatGPTRegisterAccountRepoStub struct{ accounts []Account }

func (r *chatGPTRegisterAccountRepoStub) Create(ctx context.Context, account *Account) error {
	r.accounts = append(r.accounts, *account)
	return nil
}
func (r *chatGPTRegisterAccountRepoStub) GetByID(context.Context, int64) (*Account, error) {
	return nil, nil
}
func (r *chatGPTRegisterAccountRepoStub) GetByIDs(context.Context, []int64) ([]*Account, error) {
	return nil, nil
}
func (r *chatGPTRegisterAccountRepoStub) ExistsByID(context.Context, int64) (bool, error) {
	return false, nil
}
func (r *chatGPTRegisterAccountRepoStub) GetByCRSAccountID(context.Context, string) (*Account, error) {
	return nil, nil
}
func (r *chatGPTRegisterAccountRepoStub) FindByExtraField(context.Context, string, any) ([]Account, error) {
	return nil, nil
}
func (r *chatGPTRegisterAccountRepoStub) ListCRSAccountIDs(context.Context) (map[string]int64, error) {
	return nil, nil
}
func (r *chatGPTRegisterAccountRepoStub) Update(context.Context, *Account) error { return nil }
func (r *chatGPTRegisterAccountRepoStub) Delete(context.Context, int64) error    { return nil }
func (r *chatGPTRegisterAccountRepoStub) List(context.Context, pagination.PaginationParams) ([]Account, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (r *chatGPTRegisterAccountRepoStub) ListWithFilters(context.Context, pagination.PaginationParams, string, string, string, string, int64, string) ([]Account, *pagination.PaginationResult, error) {
	return nil, nil, nil
}
func (r *chatGPTRegisterAccountRepoStub) ListByGroup(context.Context, int64) ([]Account, error) {
	return nil, nil
}
func (r *chatGPTRegisterAccountRepoStub) ListActive(context.Context) ([]Account, error) {
	return r.accounts, nil
}
func (r *chatGPTRegisterAccountRepoStub) ListByPlatform(context.Context, string) ([]Account, error) {
	return r.accounts, nil
}
func (r *chatGPTRegisterAccountRepoStub) UpdateLastUsed(context.Context, int64) error { return nil }
func (r *chatGPTRegisterAccountRepoStub) BatchUpdateLastUsed(context.Context, map[int64]time.Time) error {
	return nil
}
func (r *chatGPTRegisterAccountRepoStub) SetError(context.Context, int64, string) error { return nil }
func (r *chatGPTRegisterAccountRepoStub) ClearError(context.Context, int64) error       { return nil }
func (r *chatGPTRegisterAccountRepoStub) SetSchedulable(context.Context, int64, bool) error {
	return nil
}
func (r *chatGPTRegisterAccountRepoStub) AutoPauseExpiredAccounts(context.Context, time.Time) (int64, error) {
	return 0, nil
}
func (r *chatGPTRegisterAccountRepoStub) BindGroups(context.Context, int64, []int64) error {
	return nil
}
func (r *chatGPTRegisterAccountRepoStub) ListSchedulable(context.Context) ([]Account, error) {
	return nil, nil
}
func (r *chatGPTRegisterAccountRepoStub) ListSchedulableByGroupID(context.Context, int64) ([]Account, error) {
	return nil, nil
}
func (r *chatGPTRegisterAccountRepoStub) ListSchedulableByPlatform(context.Context, string) ([]Account, error) {
	return nil, nil
}
func (r *chatGPTRegisterAccountRepoStub) ListSchedulableByGroupIDAndPlatform(context.Context, int64, string) ([]Account, error) {
	return nil, nil
}
func (r *chatGPTRegisterAccountRepoStub) ListSchedulableByPlatforms(context.Context, []string) ([]Account, error) {
	return nil, nil
}
func (r *chatGPTRegisterAccountRepoStub) ListSchedulableByGroupIDAndPlatforms(context.Context, int64, []string) ([]Account, error) {
	return nil, nil
}
func (r *chatGPTRegisterAccountRepoStub) ListSchedulableUngroupedByPlatform(context.Context, string) ([]Account, error) {
	return nil, nil
}
func (r *chatGPTRegisterAccountRepoStub) ListSchedulableUngroupedByPlatforms(context.Context, []string) ([]Account, error) {
	return nil, nil
}
func (r *chatGPTRegisterAccountRepoStub) SetRateLimited(context.Context, int64, time.Time) error {
	return nil
}
func (r *chatGPTRegisterAccountRepoStub) SetModelRateLimit(context.Context, int64, string, time.Time, ...string) error {
	return nil
}
func (r *chatGPTRegisterAccountRepoStub) SetOverloaded(context.Context, int64, time.Time) error {
	return nil
}
func (r *chatGPTRegisterAccountRepoStub) SetTempUnschedulable(context.Context, int64, time.Time, string) error {
	return nil
}
func (r *chatGPTRegisterAccountRepoStub) ClearTempUnschedulable(context.Context, int64) error {
	return nil
}
func (r *chatGPTRegisterAccountRepoStub) ClearRateLimit(context.Context, int64) error { return nil }
func (r *chatGPTRegisterAccountRepoStub) ClearAntigravityQuotaScopes(context.Context, int64) error {
	return nil
}
func (r *chatGPTRegisterAccountRepoStub) ClearModelRateLimits(context.Context, int64) error {
	return nil
}
func (r *chatGPTRegisterAccountRepoStub) UpdateSessionWindow(context.Context, int64, *time.Time, *time.Time, string) error {
	return nil
}
func (r *chatGPTRegisterAccountRepoStub) UpdateExtra(context.Context, int64, map[string]any) error {
	return nil
}
func (r *chatGPTRegisterAccountRepoStub) BulkUpdate(context.Context, []int64, AccountBulkUpdate) (int64, error) {
	return 0, nil
}
func (r *chatGPTRegisterAccountRepoStub) IncrementQuotaUsed(context.Context, int64, float64) error {
	return nil
}
func (r *chatGPTRegisterAccountRepoStub) ResetQuotaUsed(context.Context, int64) error { return nil }
