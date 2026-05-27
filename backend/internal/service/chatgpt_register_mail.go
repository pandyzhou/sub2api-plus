package service

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/rand"
	"mime"
	"net/http"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

type chatGPTRegisterStringList []string

func (l *chatGPTRegisterStringList) UnmarshalJSON(data []byte) error {
	var arr []string
	if err := json.Unmarshal(data, &arr); err == nil {
		*l = normalizeRegisterStringList(arr)
		return nil
	}
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		if strings.TrimSpace(s) == "" {
			*l = nil
		} else {
			*l = []string{strings.TrimSpace(s)}
		}
		return nil
	}
	return nil
}

var (
	chatGPTRegisterProviderMu sync.Mutex
	chatGPTRegisterDomainMu   sync.Mutex
	chatGPTRegisterProviderIx int
	chatGPTRegisterDomainIx   int
	chatGPTRegisterCodeRE     = regexp.MustCompile(`(?is)background-color:\s*#F3F3F3[^>]*>[\s\S]*?(\d{6})[\s\S]*?</p>|(?:Verification code|code is|代码为|验证码)[:\s]*(\d{6})|>\s*(\d{6})\s*<|\b(\d{6})\b`)
)

func chatGPTRegisterNormalizeConfig(cfg ChatGPTRegisterConfig) ChatGPTRegisterConfig {
	if strings.TrimSpace(cfg.Mode) == "" {
		cfg.Mode = "total"
	}
	if cfg.Total < 1 {
		cfg.Total = 10
	}
	if cfg.Threads < 1 {
		cfg.Threads = 3
	}
	if cfg.TargetQuota < 1 {
		cfg.TargetQuota = 100
	}
	if cfg.TargetAvail < 1 {
		cfg.TargetAvail = 10
	}
	if cfg.CheckInterval < 1 {
		cfg.CheckInterval = 5
	}
	if cfg.Mail.RequestTimeout <= 0 {
		cfg.Mail.RequestTimeout = 30
	}
	if cfg.Mail.WaitTimeout <= 0 {
		cfg.Mail.WaitTimeout = 120
	}
	if cfg.Mail.WaitInterval <= 0 {
		cfg.Mail.WaitInterval = 3
	}
	legacyProvider := strings.TrimSpace(cfg.MailProvider)
	legacyBase := strings.TrimSpace(cfg.MailAPIBase)
	legacyKey := strings.TrimSpace(cfg.MailAPIKey)
	if len(cfg.Mail.Providers) == 0 {
		providerType := legacyProvider
		if providerType == "" || providerType == "custom" {
			providerType = "mailtm"
		}
		cfg.Mail.Providers = []ChatGPTRegisterMailProviderConfig{{Type: providerType, Enable: true, APIBase: legacyBase, APIKey: legacyKey}}
	}
	for i := range cfg.Mail.Providers {
		p := &cfg.Mail.Providers[i]
		p.Type = normalizeMailProviderType(p.Type)
		if p.Type == "" {
			p.Type = "mailtm"
		}
		p.APIBase = strings.TrimRight(strings.TrimSpace(p.APIBase), "/")
		p.APIKey = strings.TrimSpace(p.APIKey)
		p.AdminPassword = strings.TrimSpace(p.AdminPassword)
		p.AdminEmail = strings.TrimSpace(p.AdminEmail)
		p.DDGToken = strings.TrimSpace(p.DDGToken)
		p.CFInboxJWT = strings.TrimSpace(p.CFInboxJWT)
		p.CFAPIBase = strings.TrimRight(strings.TrimSpace(p.CFAPIBase), "/")
		p.CFAPIKey = strings.TrimSpace(p.CFAPIKey)
		p.CFAuthMode = strings.ToLower(strings.TrimSpace(p.CFAuthMode))
		p.CFCreatePath = normalizePathDefault(p.CFCreatePath, "/api/new_address")
		p.CFMessagesPath = normalizePathDefault(p.CFMessagesPath, "/api/mails")
		p.Domain = normalizeRegisterStringList(p.Domain)
		p.Subdomain = normalizeRegisterStringList(p.Subdomain)
		p.CFDomain = normalizeRegisterStringList(p.CFDomain)
	}
	if cfg.MailProvider == "" && len(cfg.Mail.Providers) > 0 {
		cfg.MailProvider = cfg.Mail.Providers[0].Type
	}
	if cfg.MailAPIBase == "" && len(cfg.Mail.Providers) > 0 {
		cfg.MailAPIBase = cfg.Mail.Providers[0].APIBase
	}
	if cfg.MailAPIKey == "" && len(cfg.Mail.Providers) > 0 {
		cfg.MailAPIKey = cfg.Mail.Providers[0].APIKey
	}
	return cfg
}

func chatGPTRegisterDecodeMailConfig(v any, fallback ChatGPTRegisterMailConfig) (ChatGPTRegisterMailConfig, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return fallback, err
	}
	var mail ChatGPTRegisterMailConfig
	if err := json.Unmarshal(data, &mail); err != nil {
		return fallback, err
	}
	cfg := chatGPTRegisterNormalizeConfig(ChatGPTRegisterConfig{Mail: mail})
	return cfg.Mail, nil
}

func normalizeMailProviderType(t string) string {
	switch strings.ToLower(strings.TrimSpace(t)) {
	case "mail.tm", "mail_tm", "mailtm", "custom", "":
		return "mailtm"
	default:
		return strings.ToLower(strings.TrimSpace(t))
	}
}

func normalizePathDefault(path, def string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return def
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	return path
}

func normalizeRegisterStringList[T ~string](in []T) chatGPTRegisterStringList {
	out := make([]string, 0, len(in))
	for _, item := range in {
		if s := strings.TrimSpace(string(item)); s != "" {
			out = append(out, s)
		}
	}
	return out
}

func chatGPTRegisterMailAPIBase(cfg ChatGPTRegisterConfig) string {
	cfg = chatGPTRegisterNormalizeConfig(cfg)
	base := strings.TrimSpace(cfg.MailAPIBase)
	if base == "" && len(cfg.Mail.Providers) > 0 {
		base = cfg.Mail.Providers[0].APIBase
	}
	if base == "" {
		return "https://api.mail.tm"
	}
	return strings.TrimRight(base, "/")
}

func chatGPTRegisterMailRequest(ctx context.Context, cfg ChatGPTRegisterConfig, method, path string, body *strings.Reader) (*http.Request, error) {
	if body == nil {
		body = strings.NewReader("")
	}
	req, err := http.NewRequestWithContext(ctx, method, chatGPTRegisterMailAPIBase(cfg)+path, body)
	if err != nil {
		return nil, err
	}
	key := strings.TrimSpace(cfg.MailAPIKey)
	if key == "" {
		cfg = chatGPTRegisterNormalizeConfig(cfg)
		if len(cfg.Mail.Providers) > 0 {
			key = strings.TrimSpace(cfg.Mail.Providers[0].APIKey)
		}
	}
	if key != "" {
		req.Header.Set("Authorization", "Bearer "+key)
	}
	return req, nil
}

type chatGPTRegisterMailProvider interface {
	Name() string
	ProviderRef() string
	CreateMailbox(ctx context.Context, username string) (*tempMailbox, error)
	FetchLatestMessage(ctx context.Context, mailbox *tempMailbox) (*chatGPTRegisterMailMessage, error)
}

type chatGPTRegisterMailMessage struct {
	Provider    string
	Mailbox     string
	MessageID   string
	Subject     string
	Sender      string
	TextContent string
	HTMLContent string
	ReceivedAt  time.Time
	Raw         any
}

type chatGPTRegisterBaseMailProvider struct {
	entry  ChatGPTRegisterMailProviderConfig
	cfg    ChatGPTRegisterMailConfig
	proxy  string
	client *http.Client
}

func (p *chatGPTRegisterBaseMailProvider) ProviderRef() string { return p.entry.ProviderRef }
func (p *chatGPTRegisterBaseMailProvider) timeout() time.Duration {
	return secondsDuration(p.cfg.RequestTimeout, 30)
}
func (p *chatGPTRegisterBaseMailProvider) apiBase(def string) string {
	base := strings.TrimRight(strings.TrimSpace(p.entry.APIBase), "/")
	if base == "" {
		base = def
	}
	return base
}
func (p *chatGPTRegisterBaseMailProvider) requestJSON(ctx context.Context, method, url string, headers map[string]string, params map[string]string, payload any, expected ...int) (any, []byte, int, error) {
	if p.client == nil {
		client, err := chatGPTRegisterHTTPClient(p.proxy, p.timeout())
		if err != nil {
			return nil, nil, 0, err
		}
		p.client = client
	}
	var body io.Reader
	if payload != nil {
		data, _ := json.Marshal(payload)
		body = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, nil, 0, err
	}
	q := req.URL.Query()
	for k, v := range params {
		q.Set(k, v)
	}
	req.URL.RawQuery = q.Encode()
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", registerFirstNonEmpty(p.cfg.UserAgent, chatGPTWebDefaultUserAgent))
	for k, v := range headers {
		if strings.TrimSpace(v) != "" {
			req.Header.Set(k, v)
		}
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if !statusIn(resp.StatusCode, expected...) {
		return nil, data, resp.StatusCode, fmt.Errorf("%s %s HTTP %d: %s", method, req.URL.Path, resp.StatusCode, truncateString(string(data), 300))
	}
	if resp.StatusCode == http.StatusNoContent || len(strings.TrimSpace(string(data))) == 0 {
		return map[string]any{}, data, resp.StatusCode, nil
	}
	var decoded any
	if strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), "json") || len(data) > 0 {
		if err := json.Unmarshal(data, &decoded); err != nil {
			return string(data), data, resp.StatusCode, nil
		}
	}
	return decoded, data, resp.StatusCode, nil
}

func statusIn(code int, expected ...int) bool {
	if len(expected) == 0 {
		expected = []int{200}
	}
	for _, v := range expected {
		if code == v {
			return true
		}
	}
	return false
}

func chatGPTRegisterCreateMailbox(ctx context.Context, cfg ChatGPTRegisterConfig, username string) (*tempMailbox, error) {
	cfg = chatGPTRegisterNormalizeConfig(cfg)
	enabled, err := chatGPTRegisterEnabledMailEntries(cfg.Mail)
	if err != nil {
		return nil, err
	}
	tried := map[string]bool{}
	lastErr := ""
	for range enabled {
		provider, err := chatGPTRegisterCreateMailProvider(cfg, "", "")
		if err != nil {
			return nil, err
		}
		key := provider.Name() + "#" + provider.ProviderRef()
		if tried[key] {
			continue
		}
		tried[key] = true
		mailbox, err := provider.CreateMailbox(ctx, username)
		if err == nil {
			return mailbox, nil
		}
		lastErr = err.Error()
		if !strings.Contains(lastErr, "DDG日上限已达") {
			return nil, err
		}
	}
	if lastErr == "" {
		lastErr = "所有启用的邮箱提供商均无法创建邮箱"
	}
	return nil, errors.New(lastErr)
}

func chatGPTRegisterWaitForCode(ctx context.Context, cfg ChatGPTRegisterConfig, mailbox *tempMailbox) (string, error) {
	cfg = chatGPTRegisterNormalizeConfig(cfg)
	provider, err := chatGPTRegisterCreateMailProvider(cfg, mailbox.Provider, mailbox.ProviderRef)
	if err != nil {
		return "", err
	}
	if mailbox.seenRefs == nil {
		mailbox.seenRefs = map[string]bool{}
	}
	deadline := time.NewTimer(secondsDuration(cfg.Mail.WaitTimeout, 120))
	defer deadline.Stop()
	interval := time.NewTicker(secondsDuration(cfg.Mail.WaitInterval, 3))
	defer interval.Stop()
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-deadline.C:
			return "", fmt.Errorf("OTP wait timeout")
		case <-interval.C:
			msg, err := provider.FetchLatestMessage(ctx, mailbox)
			if err != nil || msg == nil {
				continue
			}
			ref := msg.trackingRef(provider.Name(), mailbox.Email)
			if mailbox.seenRefs[ref] {
				continue
			}
			code := chatGPTRegisterExtractCode(msg.Subject + "\n" + msg.TextContent + "\n" + msg.HTMLContent)
			if code != "" {
				mailbox.seenRefs[ref] = true
				return code, nil
			}
		}
	}
}

func (m *chatGPTRegisterMailMessage) trackingRef(provider, mailbox string) string {
	if m == nil {
		return ""
	}
	if m.MessageID != "" {
		return "id:" + provider + ":" + mailbox + ":" + m.MessageID
	}
	h := sha256.Sum256([]byte(m.Subject + "\n" + m.Sender + "\n" + m.TextContent + "\n" + m.HTMLContent))
	return fmt.Sprintf("content:%s:%s:%d:%x", provider, mailbox, m.ReceivedAt.UnixNano(), h[:])
}

func chatGPTRegisterExtractCode(content string) string {
	content = strings.TrimSpace(content)
	if content == "" {
		return ""
	}
	matches := chatGPTRegisterCodeRE.FindAllStringSubmatch(content, -1)
	for _, m := range matches {
		for i := 1; i < len(m); i++ {
			if m[i] != "" && m[i] != "177010" {
				return m[i]
			}
		}
	}
	return ""
}

func chatGPTRegisterEnabledMailEntries(mail ChatGPTRegisterMailConfig) ([]ChatGPTRegisterMailProviderConfig, error) {
	entries := chatGPTRegisterMailEntries(mail)
	out := make([]ChatGPTRegisterMailProviderConfig, 0, len(entries))
	for _, item := range entries {
		if item.Enable {
			out = append(out, item)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("mail.providers 没有启用的 provider")
	}
	return out, nil
}

func chatGPTRegisterMailEntries(mail ChatGPTRegisterMailConfig) []ChatGPTRegisterMailProviderConfig {
	out := make([]ChatGPTRegisterMailProviderConfig, 0, len(mail.Providers))
	counters := map[string]int{}
	for i, item := range mail.Providers {
		item.Type = normalizeMailProviderType(item.Type)
		counters[item.Type]++
		idx := i + 1
		item.ProviderRef = fmt.Sprintf("%s#%d", item.Type, idx)
		if item.Type == "ddg_mail" {
			item.Label = fmt.Sprintf("DDG-%d", counters[item.Type])
		} else if item.Label == "" {
			item.Label = item.ProviderRef
		}
		out = append(out, item)
	}
	return out
}

func chatGPTRegisterNextMailEntry(mail ChatGPTRegisterMailConfig) (ChatGPTRegisterMailProviderConfig, error) {
	items, err := chatGPTRegisterEnabledMailEntries(mail)
	if err != nil {
		return ChatGPTRegisterMailProviderConfig{}, err
	}
	if len(items) == 1 {
		return items[0], nil
	}
	chatGPTRegisterProviderMu.Lock()
	defer chatGPTRegisterProviderMu.Unlock()
	value := items[chatGPTRegisterProviderIx%len(items)]
	chatGPTRegisterProviderIx = (chatGPTRegisterProviderIx + 1) % len(items)
	return value, nil
}

func chatGPTRegisterCreateMailProvider(cfg ChatGPTRegisterConfig, providerType, providerRef string) (chatGPTRegisterMailProvider, error) {
	cfg = chatGPTRegisterNormalizeConfig(cfg)
	var entry ChatGPTRegisterMailProviderConfig
	entries := chatGPTRegisterMailEntries(cfg.Mail)
	for _, item := range entries {
		if providerRef != "" && item.ProviderRef == providerRef {
			entry = item
			break
		}
	}
	if entry.Type == "" && providerType != "" {
		for _, item := range entries {
			if item.Enable && item.Type == normalizeMailProviderType(providerType) {
				entry = item
				break
			}
		}
	}
	if entry.Type == "" {
		var err error
		entry, err = chatGPTRegisterNextMailEntry(cfg.Mail)
		if err != nil {
			return nil, err
		}
	}
	base := chatGPTRegisterBaseMailProvider{entry: entry, cfg: cfg.Mail, proxy: cfg.Proxy}
	switch entry.Type {
	case "mailtm":
		return &mailTMProvider{chatGPTRegisterBaseMailProvider: base}, nil
	case "cloudflare_temp_email":
		return &cloudflareTempEmailProvider{chatGPTRegisterBaseMailProvider: base}, nil
	case "tempmail_lol":
		return &tempMailLolProvider{chatGPTRegisterBaseMailProvider: base}, nil
	case "inbucket":
		return &inbucketProvider{chatGPTRegisterBaseMailProvider: base}, nil
	case "moemail":
		return &moEmailProvider{chatGPTRegisterBaseMailProvider: base}, nil
	case "cloudmail_gen":
		return &cloudMailGenProvider{chatGPTRegisterBaseMailProvider: base}, nil
	case "ddg_mail":
		return &ddgMailProvider{chatGPTRegisterBaseMailProvider: base}, nil
	case "duckmail":
		return &duckMailProvider{chatGPTRegisterBaseMailProvider: base}, nil
	case "gptmail":
		return &gptMailProvider{chatGPTRegisterBaseMailProvider: base}, nil
	case "yyds_mail":
		return &yydsMailProvider{chatGPTRegisterBaseMailProvider: base}, nil
	default:
		return nil, fmt.Errorf("不支持的 mail.provider: %s", entry.Type)
	}
}

func nextRegisterDomain(domains []string) (string, error) {
	items := normalizeRegisterStringList(domains)
	if len(items) == 0 {
		return "", fmt.Errorf("mail.domain 不能为空")
	}
	if len(items) == 1 {
		return items[0], nil
	}
	chatGPTRegisterDomainMu.Lock()
	defer chatGPTRegisterDomainMu.Unlock()
	value := items[chatGPTRegisterDomainIx%len(items)]
	chatGPTRegisterDomainIx = (chatGPTRegisterDomainIx + 1) % len(items)
	return value, nil
}

func randomMailboxName() string {
	return fmt.Sprintf("%s%d%s", randomLower(5), rand.Intn(999)+1, randomLower(rand.Intn(3)+1))
}
func randomSubdomainLabel() string { return randomLower(rand.Intn(7) + 4) }
func randomLower(n int) string {
	const chars = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}
func secondsDuration(v float64, def float64) time.Duration {
	if v <= 0 {
		v = def
	}
	return time.Duration(v * float64(time.Second))
}
func registerFirstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func parseMessageTime(v any) time.Time {
	switch x := v.(type) {
	case float64:
		return time.Unix(int64(x), 0).UTC()
	case int64:
		return time.Unix(x, 0).UTC()
	case string:
		s := strings.TrimSpace(x)
		if s == "" {
			return time.Time{}
		}
		if t, err := time.Parse(time.RFC3339, s); err == nil {
			return t
		}
		if t, err := time.Parse(time.RFC1123Z, s); err == nil {
			return t
		}
		if t, err := http.ParseTime(s); err == nil {
			return t
		}
	}
	return time.Time{}
}

func extractContent(item map[string]any) (string, string) {
	text := stringAny(item, "text_content", "text", "body", "content")
	html := stringAny(item, "html_content", "html", "html_body", "body_html")
	if arr, ok := item["html"].([]any); ok {
		var b strings.Builder
		for _, v := range arr {
			b.WriteString(fmt.Sprint(v))
		}
		html = b.String()
	}
	if text == "" && html == "" {
		if raw := stringAny(item, "raw"); raw != "" {
			text, html = extractRawEmailContent(raw)
		}
	}
	return text, html
}

func extractRawEmailContent(raw string) (string, string) {
	parts := strings.SplitN(raw, "\r\n\r\n", 2)
	if len(parts) < 2 {
		parts = strings.SplitN(raw, "\n\n", 2)
	}
	if len(parts) < 2 {
		return raw, ""
	}
	headers := parts[0]
	body := parts[1]
	ct := ""
	for _, line := range strings.Split(headers, "\n") {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(line)), "content-type:") {
			ct = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
			break
		}
	}
	mt, _, _ := mime.ParseMediaType(ct)
	if mt == "text/html" {
		return "", body
	}
	return body, ""
}

func messageMatchesEmail(item map[string]any, email string) bool {
	target := strings.ToLower(strings.TrimSpace(email))
	if target == "" {
		return true
	}
	var candidates []string
	for _, k := range []string{"to", "mailTo", "receiver", "receivers", "address", "email", "envelope_to"} {
		candidates = append(candidates, extractTextCandidates(item[k])...)
	}
	if len(candidates) == 0 {
		return true
	}
	for _, c := range candidates {
		if strings.Contains(strings.ToLower(strings.TrimSpace(c)), target) {
			return true
		}
	}
	return false
}

func extractTextCandidates(v any) []string {
	switch x := v.(type) {
	case string:
		return []string{x}
	case []any:
		var out []string
		for _, item := range x {
			out = append(out, extractTextCandidates(item)...)
		}
		return out
	case map[string]any:
		var out []string
		for _, k := range []string{"address", "email", "name", "value"} {
			out = append(out, extractTextCandidates(x[k])...)
		}
		return out
	default:
		return nil
	}
}

func stringAny(m map[string]any, keys ...string) string {
	for _, k := range keys {
		if v, ok := m[k]; ok && v != nil {
			s := strings.TrimSpace(fmt.Sprint(v))
			if s != "" && s != "<nil>" {
				return s
			}
		}
	}
	return ""
}
func mapAny(v any) map[string]any {
	if m, ok := v.(map[string]any); ok {
		return m
	}
	return nil
}
func listAny(v any) []any {
	if a, ok := v.([]any); ok {
		return a
	}
	return nil
}
func itemsFromAny(data any, keys ...string) []map[string]any {
	var raw []any
	if a, ok := data.([]any); ok {
		raw = a
	}
	if m, ok := data.(map[string]any); ok {
		for _, k := range keys {
			if a, ok := m[k].([]any); ok {
				raw = a
				break
			}
			if mm, ok := m[k].(map[string]any); ok {
				if a, ok := mm["messages"].([]any); ok {
					raw = a
					break
				}
			}
		}
	}
	out := make([]map[string]any, 0, len(raw))
	for _, item := range raw {
		if m, ok := item.(map[string]any); ok {
			out = append(out, m)
		}
	}
	return out
}
func latestItem(items []map[string]any) map[string]any {
	if len(items) == 0 {
		return nil
	}
	sort.SliceStable(items, func(i, j int) bool {
		ti := parseMessageTime(firstMessageTimeValue(items[i])).UnixNano()
		tj := parseMessageTime(firstMessageTimeValue(items[j])).UnixNano()
		if ti == tj {
			return stringAny(items[i], "id", "token", "message_id") > stringAny(items[j], "id", "token", "message_id")
		}
		return ti > tj
	})
	return items[0]
}
func firstMessageTimeValue(m map[string]any) any {
	for _, k := range []string{"createdAt", "created_at", "receivedAt", "received_at", "date", "timestamp"} {
		if v, ok := m[k]; ok {
			return v
		}
	}
	return nil
}
