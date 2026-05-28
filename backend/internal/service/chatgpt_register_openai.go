package service

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	mathrand "math/rand"
	"net/http"
	"net/http/cookiejar"
	"net/url"
	"strings"

	fhttp "github.com/bogdanfinn/fhttp"
	tlsclient "github.com/bogdanfinn/tls-client"
	tlsclientprofiles "github.com/bogdanfinn/tls-client/profiles"
	"time"

	openaioauth "github.com/Wei-Shaw/sub2api/internal/pkg/openai"
)

var (
	chatGPTRegisterAuthBase     = "https://auth.openai.com"
	chatGPTRegisterPlatformBase = "https://platform.openai.com"
	chatGPTRegisterSentinelBase = "https://sentinel.openai.com"
)

const (
	chatGPTRegisterPlatformOAuthClientID    = "app_2SKx67EdpoN0G6j64rFvigXD"
	chatGPTRegisterPlatformOAuthAudience    = "https://api.openai.com/v1"
	chatGPTRegisterPlatformAuth0Client      = "eyJuYW1lIjoiYXV0aDAtc3BhLWpzIiwidmVyc2lvbiI6IjEuMjEuMCJ9"
	chatGPTRegisterSecCHUA                  = `"Google Chrome";v="145", "Not?A_Brand";v="8", "Chromium";v="145"`
	chatGPTRegisterSecCHUAFullVersionList   = `"Chromium";v="145.0.0.0", "Not:A-Brand";v="99.0.0.0", "Google Chrome";v="145.0.0.0"`
	chatGPTRegisterSentinelErrorTokenPrefix = "wQ8Lk5FbGpA2NcR9dShT6gYjU7VxZ4D"
	chatGPTRegisterSentinelMaxAttempt       = 500000
)

type chatGPTRegisterOpenAIClient struct {
	http       *http.Client
	tlsClient  tlsclient.HttpClient
	deviceID   string
	proxyURL   string
}

func newChatGPTRegisterOpenAIClient(proxyURL, deviceID string) (*chatGPTRegisterOpenAIClient, error) {
	client, err := chatGPTRegisterHTTPClient(proxyURL, 60*time.Second)
	if err != nil {
		return nil, err
	}
	jar, _ := cookiejar.New(nil)
	client.Jar = jar
	if strings.TrimSpace(deviceID) == "" {
		deviceID = chatGPTRegisterRandomUUID()
	}

	tlsProxyURL := chatGPTRegisterTLSProxyURL(proxyURL)
	tlsJar := tlsclient.NewCookieJar()
	tlsOpts := []tlsclient.HttpClientOption{
		tlsclient.WithTimeoutSeconds(60),
		tlsclient.WithClientProfile(tlsclientprofiles.Chrome_144),
		tlsclient.WithNotFollowRedirects(),
		tlsclient.WithCookieJar(tlsJar),
	}
	if tlsProxyURL != "" {
		tlsOpts = append(tlsOpts, tlsclient.WithProxyUrl(tlsProxyURL))
	}
	tlsCli, err := tlsclient.NewHttpClient(tlsclient.NewNoopLogger(), tlsOpts...)
	if err != nil {
		return nil, fmt.Errorf("init tls-client: %w", err)
	}

	return &chatGPTRegisterOpenAIClient{http: client, tlsClient: tlsCli, deviceID: deviceID, proxyURL: tlsProxyURL}, nil
}

func chatGPTRegisterTLSProxyURL(proxyURL string) string {
	if proxyURL = strings.TrimSpace(proxyURL); proxyURL != "" {
		return proxyURL
	}
	req, err := http.NewRequest(http.MethodGet, strings.TrimRight(chatGPTRegisterAuthBase, "/"), nil)
	if err != nil {
		return ""
	}
	u, err := http.ProxyFromEnvironment(req)
	if err != nil || u == nil {
		return ""
	}
	return u.String()
}

func (c *chatGPTRegisterOpenAIClient) close() {
	if c.tlsClient != nil {
		c.tlsClient.CloseIdleConnections()
	}
}

func chatGPTRegisterPlatformRedirectURI() string {
	return strings.TrimRight(chatGPTRegisterPlatformBase, "/") + "/auth/callback"
}

func chatGPTRegisterCommonHeaders(deviceID, referer string) map[string]string {
	h := map[string]string{
		"Accept":                      "application/json",
		"Accept-Language":             "en-US,en;q=0.9",
		"Content-Type":                "application/json",
		"Origin":                      strings.TrimRight(chatGPTRegisterAuthBase, "/"),
		"Priority":                    "u=1, i",
		"User-Agent":                  chatGPTWebDefaultUserAgent,
		"Sec-Ch-Ua":                   chatGPTRegisterSecCHUA,
		"Sec-Ch-Ua-Arch":              `"x86_64"`,
		"Sec-Ch-Ua-Bitness":           `"64"`,
		"Sec-Ch-Ua-Full-Version-List": chatGPTRegisterSecCHUAFullVersionList,
		"Sec-Ch-Ua-Mobile":            "?0",
		"Sec-Ch-Ua-Model":             `""`,
		"Sec-Ch-Ua-Platform":          `"Windows"`,
		"Sec-Ch-Ua-Platform-Version":  `"10.0.0"`,
		"Sec-Fetch-Dest":              "empty",
		"Sec-Fetch-Mode":              "cors",
		"Sec-Fetch-Site":              "same-origin",
		"OAI-Device-Id":               deviceID,
	}
	if referer != "" {
		h["Referer"] = referer
	}
	for k, v := range chatGPTRegisterTraceHeaders() {
		h[k] = v
	}
	return h
}

func chatGPTRegisterNavigateHeaders(referer string) map[string]string {
	h := map[string]string{
		"Accept":                      "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
		"Accept-Language":             "en-US,en;q=0.9",
		"User-Agent":                  chatGPTWebDefaultUserAgent,
		"Sec-Ch-Ua":                   chatGPTRegisterSecCHUA,
		"Sec-Ch-Ua-Arch":              `"x86_64"`,
		"Sec-Ch-Ua-Bitness":           `"64"`,
		"Sec-Ch-Ua-Full-Version-List": chatGPTRegisterSecCHUAFullVersionList,
		"Sec-Ch-Ua-Mobile":            "?0",
		"Sec-Ch-Ua-Model":             `""`,
		"Sec-Ch-Ua-Platform":          `"Windows"`,
		"Sec-Ch-Ua-Platform-Version":  `"10.0.0"`,
		"Sec-Fetch-Dest":              "document",
		"Sec-Fetch-Mode":              "navigate",
		"Sec-Fetch-Site":              "same-origin",
		"Sec-Fetch-User":              "?1",
		"Upgrade-Insecure-Requests":   "1",
	}
	if referer != "" {
		h["Referer"] = referer
	}
	return h
}

func chatGPTRegisterTraceHeaders() map[string]string {
	traceID := fmt.Sprintf("%d", mathrand.Uint64())
	parentID := fmt.Sprintf("%d", mathrand.Uint64())
	return map[string]string{
		"Traceparent":                 "00-" + chatGPTRegisterRandomHex(16) + "-" + fmt.Sprintf("%016x", mathrand.Uint64()) + "-01",
		"Tracestate":                  "dd=s:1;o:rum",
		"X-Datadog-Origin":            "rum",
		"X-Datadog-Parent-Id":         parentID,
		"X-Datadog-Sampling-Priority": "1",
		"X-Datadog-Trace-Id":          traceID,
	}
}

func (c *chatGPTRegisterOpenAIClient) setDeviceCookies() {
	u, _ := url.Parse(strings.TrimRight(chatGPTRegisterAuthBase, "/"))
	cookies := []*fhttp.Cookie{{Name: "oai-did", Value: c.deviceID, Path: "/"}}
	if c.tlsClient != nil {
		c.tlsClient.SetCookies(u, cookies)
	}
}

func (c *chatGPTRegisterOpenAIClient) do(req *http.Request) (*http.Response, []byte, error) {
	resp, err := c.http.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return resp, nil, err
	}
	return resp, body, nil
}

func (c *chatGPTRegisterOpenAIClient) tlsDo(req *http.Request) (*http.Response, []byte, error) {
	freq := &fhttp.Request{
		Method: req.Method,
		URL:    req.URL,
		Header: fhttp.Header(req.Header),
		Body:   req.Body,
	}
	resp, err := c.tlsClient.Do(freq)
	if err != nil {
		return nil, nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 10<<20))
	if err != nil {
		return nil, body, err
	}
	return &http.Response{StatusCode: resp.StatusCode, Header: http.Header(resp.Header), Status: resp.Status, Request: req}, body, nil
}

func (c *chatGPTRegisterOpenAIClient) requestJSON(ctx context.Context, method, urlValue string, payload any, headers map[string]string, expected ...int) (map[string]any, *http.Response, []byte, error) {
	var body io.Reader
	if payload != nil {
		data, _ := json.Marshal(payload)
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, urlValue, body)
	if err != nil {
		return nil, nil, nil, err
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, respBody, err := c.tlsDo(req)
	if err != nil {
		return nil, nil, nil, err
	}
	if !statusIn(resp.StatusCode, expected...) {
		return nil, resp, respBody, fmt.Errorf("%s %s HTTP %d: %s", method, req.URL.Path, resp.StatusCode, truncateString(string(respBody), 500))
	}
	var data map[string]any
	_ = json.Unmarshal(respBody, &data)
	return data, resp, respBody, nil
}

func (c *chatGPTRegisterOpenAIClient) platformAuthorize(ctx context.Context, email, codeChallenge, state, nonce string) error {
	c.setDeviceCookies()
	params := url.Values{
		"issuer": {chatGPTRegisterAuthBase}, "client_id": {chatGPTRegisterPlatformOAuthClientID}, "audience": {chatGPTRegisterPlatformOAuthAudience}, "redirect_uri": {chatGPTRegisterPlatformRedirectURI()}, "device_id": {c.deviceID}, "screen_hint": {"login_or_signup"}, "max_age": {"0"}, "login_hint": {email}, "scope": {"openid profile email offline_access"}, "response_type": {"code"}, "response_mode": {"query"}, "state": {state}, "nonce": {nonce}, "code_challenge": {codeChallenge}, "code_challenge_method": {"S256"}, "auth0Client": {chatGPTRegisterPlatformAuth0Client},
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(chatGPTRegisterAuthBase, "/")+"/api/accounts/authorize?"+params.Encode(), nil)
	if err != nil {
		return err
	}
	for k, v := range chatGPTRegisterNavigateHeaders(strings.TrimRight(chatGPTRegisterPlatformBase, "/") + "/") {
		req.Header.Set(k, v)
	}
	resp, body, err := c.tlsDo(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 && resp.StatusCode != 302 && resp.StatusCode != 307 {
		return fmt.Errorf("platform_authorize_http_%d: %s", resp.StatusCode, truncateString(string(body), 300))
	}
	return nil
}

func (c *chatGPTRegisterOpenAIClient) registerUser(ctx context.Context, email, password string) error {
	h := chatGPTRegisterCommonHeaders(c.deviceID, strings.TrimRight(chatGPTRegisterAuthBase, "/")+"/create-account/password")
	token, err := c.buildSentinelToken(ctx, "username_password_create")
	if err != nil {
		return err
	}
	h["OpenAI-Sentinel-Token"] = token
	_, _, _, err = c.requestJSON(ctx, http.MethodPost, strings.TrimRight(chatGPTRegisterAuthBase, "/")+"/api/accounts/user/register", map[string]string{"username": email, "password": password}, h, 200)
	return err
}

func (c *chatGPTRegisterOpenAIClient) sendOTP(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(chatGPTRegisterAuthBase, "/")+"/api/accounts/email-otp/send", nil)
	if err != nil {
		return err
	}
	for k, v := range chatGPTRegisterNavigateHeaders(strings.TrimRight(chatGPTRegisterAuthBase, "/") + "/create-account/password") {
		req.Header.Set(k, v)
	}
	resp, body, err := c.tlsDo(req)
	if err != nil {
		return err
	}
	if resp.StatusCode != 200 && resp.StatusCode != 302 {
		return fmt.Errorf("send_otp_http_%d: %s", resp.StatusCode, truncateString(string(body), 300))
	}
	return nil
}

func (c *chatGPTRegisterOpenAIClient) validateOTP(ctx context.Context, code string) error {
	h := chatGPTRegisterCommonHeaders(c.deviceID, strings.TrimRight(chatGPTRegisterAuthBase, "/")+"/email-verification")
	_, resp, body, err := c.requestJSON(ctx, http.MethodPost, strings.TrimRight(chatGPTRegisterAuthBase, "/")+"/api/accounts/email-otp/validate", map[string]string{"code": code}, h, 200)
	if err == nil && resp != nil && resp.StatusCode == 200 {
		return nil
	}
	token, tokErr := c.buildSentinelToken(ctx, "authorize_continue")
	if tokErr != nil {
		return fmt.Errorf("validate_otp_http_%d: %s; sentinel retry failed: %w", statusCode(resp), truncateString(string(body), 300), tokErr)
	}
	h["OpenAI-Sentinel-Token"] = token
	_, _, _, err = c.requestJSON(ctx, http.MethodPost, strings.TrimRight(chatGPTRegisterAuthBase, "/")+"/api/accounts/email-otp/validate", map[string]string{"code": code}, h, 200)
	return err
}

func (c *chatGPTRegisterOpenAIClient) createAccountProfile(ctx context.Context, name, birthdate string) error {
	h := chatGPTRegisterCommonHeaders(c.deviceID, strings.TrimRight(chatGPTRegisterAuthBase, "/")+"/about-you")
	token, err := c.buildSentinelToken(ctx, "oauth_create_account")
	if err != nil {
		return err
	}
	h["OpenAI-Sentinel-Token"] = token
	_, _, _, err = c.requestJSON(ctx, http.MethodPost, strings.TrimRight(chatGPTRegisterAuthBase, "/")+"/api/accounts/create_account", map[string]string{"name": name, "birthdate": birthdate}, h, 200, 302)
	return err
}

func (c *chatGPTRegisterOpenAIClient) loginAndExchangeTokens(ctx context.Context, email, password, _ string, mailbox *tempMailbox, cfg ChatGPTRegisterConfig, svc *ChatGPTRegisterService) (*registerTokens, error) {
	login, err := newChatGPTRegisterOpenAIClient(cfg.Proxy, "")
	if err != nil {
		return nil, err
	}
	login.setDeviceCookies()

	loginVerifier, challenge, state, nonce := chatGPTRegisterGeneratePKCE()
	params := url.Values{"issuer": {chatGPTRegisterAuthBase}, "client_id": {chatGPTRegisterPlatformOAuthClientID}, "audience": {chatGPTRegisterPlatformOAuthAudience}, "redirect_uri": {chatGPTRegisterPlatformRedirectURI()}, "device_id": {login.deviceID}, "screen_hint": {"login_or_signup"}, "max_age": {"0"}, "login_hint": {email}, "scope": {"openid profile email offline_access"}, "response_type": {"code"}, "response_mode": {"query"}, "state": {state}, "nonce": {nonce}, "code_challenge": {challenge}, "code_challenge_method": {"S256"}, "auth0Client": {chatGPTRegisterPlatformAuth0Client}}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, strings.TrimRight(chatGPTRegisterAuthBase, "/")+"/api/accounts/authorize?"+params.Encode(), nil)
	for k, v := range chatGPTRegisterNavigateHeaders(strings.TrimRight(chatGPTRegisterPlatformBase, "/")+"/") {
		req.Header.Set(k, v)
	}
	resp, body, err := login.tlsDo(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != 200 && resp.StatusCode != 302 && resp.StatusCode != 307 {
		return nil, fmt.Errorf("platform_login_authorize_http_%d: %s", resp.StatusCode, truncateString(string(body), 300))
	}

	h := chatGPTRegisterCommonHeaders(login.deviceID, strings.TrimRight(chatGPTRegisterAuthBase, "/")+"/log-in?usernameKind=email")
	token, err := login.buildSentinelToken(ctx, "authorize_continue")
	if err != nil {
		return nil, err
	}
	h["OpenAI-Sentinel-Token"] = token
	_, _, _, err = login.requestJSON(ctx, http.MethodPost, strings.TrimRight(chatGPTRegisterAuthBase, "/")+"/api/accounts/authorize/continue", map[string]any{"username": map[string]string{"kind": "email", "value": email}}, h, 200)
	if err != nil {
		return nil, err
	}

	h = chatGPTRegisterCommonHeaders(login.deviceID, strings.TrimRight(chatGPTRegisterAuthBase, "/")+"/log-in/password")
	token, err = login.buildSentinelToken(ctx, "password_verify")
	if err != nil {
		return nil, err
	}
	h["OpenAI-Sentinel-Token"] = token
	payload, _, _, err := login.requestJSON(ctx, http.MethodPost, strings.TrimRight(chatGPTRegisterAuthBase, "/")+"/api/accounts/password/verify", map[string]string{"password": password}, h, 200)
	if err != nil {
		return nil, err
	}

	continueURL := strings.TrimSpace(fmt.Sprint(payload["continue_url"]))
	pageType := strings.TrimSpace(fmt.Sprint(payload["type"]))
	if page := mapAny(payload["page"]); page != nil {
		pageType = stringAny(page, "type")
	}
	if pageType == "email_otp_verification" || strings.Contains(continueURL, "email-verification") || strings.Contains(continueURL, "email-otp") {
		if svc == nil {
			return nil, fmt.Errorf("独立登录需要邮箱验证码但服务不可用")
		}
		code, err := svc.waitForOTPCode(ctx, mailbox, cfg)
		if err != nil {
			return nil, err
		}
		if err := login.validateOTP(ctx, code); err != nil {
			return nil, err
		}
	}

	code := ""
	if tokenPayload := mapAny(payload["payload"]); tokenPayload != nil {
		code = strings.TrimSpace(fmt.Sprint(tokenPayload["code"]))
	}
	if code == "" {
		code = oauthCodeFromURL(continueURL)
	}
	if code == "" {
		return nil, fmt.Errorf("login_exchange: token_exchange code missing, type=%q continue_url=%q payload_keys=%v", pageType, continueURL, mapKeys(payload))
	}

	form := url.Values{"grant_type": {"authorization_code"}, "code": {code}, "redirect_uri": {chatGPTRegisterPlatformRedirectURI()}, "client_id": {chatGPTRegisterPlatformOAuthClientID}, "code_verifier": {loginVerifier}}
	tokenReq, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(chatGPTRegisterAuthBase, "/")+"/oauth/token", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	tokenReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	tokenResp, tokenBody, err := login.tlsDo(tokenReq)
	if err != nil {
		return nil, err
	}
	if tokenResp.StatusCode != 200 {
		return nil, fmt.Errorf("login_exchange_token_http_%d: %s", tokenResp.StatusCode, truncateString(string(tokenBody), 300))
	}
	var data registerTokens
	var raw map[string]any
	if err := json.Unmarshal(tokenBody, &raw); err != nil {
		return nil, err
	}
	data.AccessToken = strings.TrimSpace(fmt.Sprint(raw["access_token"]))
	data.RefreshToken = strings.TrimSpace(fmt.Sprint(raw["refresh_token"]))
	data.IDToken = strings.TrimSpace(fmt.Sprint(raw["id_token"]))
	if data.AccessToken == "" || data.RefreshToken == "" || data.IDToken == "" {
		return nil, fmt.Errorf("login_exchange: token exchange failed, empty tokens")
	}
	return &data, nil
}

func mapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func oauthCodeFromURL(raw string) string {
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(u.Query().Get("code"))
}

func chatGPTRegisterCodexAuthorizationURL(state, codeChallenge string) string {
	params := url.Values{}
	params.Set("response_type", "code")
	params.Set("client_id", openaioauth.ClientID)
	params.Set("redirect_uri", openaioauth.DefaultRedirectURI)
	params.Set("scope", openaioauth.DefaultScopes)
	params.Set("state", state)
	params.Set("code_challenge", codeChallenge)
	params.Set("code_challenge_method", "S256")
	params.Set("id_token_add_organizations", "true")
	params.Set("codex_cli_simplified_flow", "true")
	return strings.TrimRight(chatGPTRegisterAuthBase, "/") + "/oauth/authorize?" + params.Encode()
}

func (c *chatGPTRegisterOpenAIClient) buildSentinelToken(ctx context.Context, flow string) (string, error) {
	generator := newSentinelTokenGenerator(c.deviceID, chatGPTWebDefaultUserAgent)
	payload := map[string]any{"p": generator.requirementsToken(), "id": c.deviceID, "flow": flow}
	dataBytes, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(chatGPTRegisterSentinelBase, "/")+"/backend-api/sentinel/req", bytes.NewReader(dataBytes))
	if err != nil {
		return "", err
	}
	for k, v := range map[string]string{"Content-Type": "text/plain;charset=UTF-8", "Referer": strings.TrimRight(chatGPTRegisterSentinelBase, "/") + "/backend-api/sentinel/frame.html", "Origin": strings.TrimRight(chatGPTRegisterSentinelBase, "/"), "User-Agent": chatGPTWebDefaultUserAgent, "Sec-Ch-Ua": chatGPTRegisterSecCHUA, "Sec-Ch-Ua-Mobile": "?0", "Sec-Ch-Ua-Platform": "\"Windows\""} {
		req.Header.Set(k, v)
	}
	resp, body, err := c.tlsDo(req)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != 200 {
		return "", fmt.Errorf("sentinel_req_failed_%d", resp.StatusCode)
	}
	var data map[string]any
	_ = json.Unmarshal(body, &data)
	token := strings.TrimSpace(fmt.Sprint(data["token"]))
	if token == "" {
		return "", fmt.Errorf("sentinel_req_failed_empty_token")
	}
	pValue := generator.requirementsToken()
	if pow := mapAny(data["proofofwork"]); pow != nil {
		if reqd, ok := pow["required"].(bool); ok && reqd {
			pValue = generator.powToken(stringAny(pow, "seed"), stringAny(pow, "difficulty"))
		}
	}
	out, _ := json.Marshal(map[string]any{"p": pValue, "t": "", "c": token, "id": c.deviceID, "flow": flow})
	return string(out), nil
}

type sentinelTokenGenerator struct{ deviceID, userAgent, sid string }

func newSentinelTokenGenerator(deviceID, ua string) *sentinelTokenGenerator {
	return &sentinelTokenGenerator{deviceID: deviceID, userAgent: ua, sid: chatGPTRegisterRandomUUID()}
}
func (g *sentinelTokenGenerator) config() []any {
	perf := mathrand.Float64()*49000 + 1000
	return []any{"1920x1080", time.Now().UTC().Format("Mon Jan 02 2006 15:04:05 GMT+0000 (Coordinated Universal Time)"), 4294705152, mathrand.Float64(), g.userAgent, "https://sentinel.openai.com/sentinel/20260124ceb8/sdk.js", nil, nil, "en-US", mathrand.Float64(), "webdriver-false", "location", "window", perf, g.sid, "", 8, float64(time.Now().UnixMilli()) - perf}
}
func (g *sentinelTokenGenerator) b64(data any) string {
	b, _ := json.Marshal(data)
	return base64.StdEncoding.EncodeToString(b)
}
func (g *sentinelTokenGenerator) requirementsToken() string {
	data := g.config()
	data[3] = 1
	data[9] = mathrand.Intn(46) + 5
	return "gAAAAAC" + g.b64(data)
}
func (g *sentinelTokenGenerator) powToken(seed, difficulty string) string {
	if difficulty == "" {
		difficulty = "0"
	}
	start := time.Now()
	data := g.config()
	for i := 0; i < chatGPTRegisterSentinelMaxAttempt; i++ {
		data[3] = i
		data[9] = time.Since(start).Milliseconds()
		payload := g.b64(data)
		if fnv1a32(seed + payload)[:len(difficulty)] <= difficulty {
			return "gAAAAAB" + payload + "~S"
		}
	}
	return "gAAAAAB" + chatGPTRegisterSentinelErrorTokenPrefix + g.b64("null")
}

func fnv1a32(text string) string {
	var h uint32 = 2166136261
	for _, ch := range text {
		h ^= uint32(ch)
		h *= 16777619
	}
	h ^= h >> 16
	h *= 2246822507
	h ^= h >> 13
	h *= 3266489909
	h ^= h >> 16
	return fmt.Sprintf("%08x", h)
}
func statusCode(resp *http.Response) int {
	if resp == nil {
		return 0
	}
	return resp.StatusCode
}

func randomBase64URL(n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return strings.TrimRight(base64.URLEncoding.EncodeToString(b), "=")
}

var _ = randomBase64URL
