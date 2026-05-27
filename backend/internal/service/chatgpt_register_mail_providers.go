package service

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	xnetproxy "golang.org/x/net/proxy"
)

func chatGPTRegisterHTTPClient(proxyURL string, timeout time.Duration) (*http.Client, error) {
	transport, err := chatGPTRegisterHTTPTransport(proxyURL)
	if err != nil {
		return nil, err
	}
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	return &http.Client{Timeout: timeout, Transport: transport}, nil
}

func chatGPTRegisterHTTPTransport(proxyURL string) (*http.Transport, error) {
	transport := &http.Transport{Proxy: http.ProxyFromEnvironment}
	proxyURL = strings.TrimSpace(proxyURL)
	if proxyURL == "" {
		return transport, nil
	}
	u, err := url.Parse(proxyURL)
	if err != nil {
		return nil, fmt.Errorf("invalid proxy URL: %w", err)
	}
	scheme := strings.ToLower(u.Scheme)
	switch scheme {
	case "http", "https":
		transport.Proxy = http.ProxyURL(u)
	case "socks5", "socks5h":
		var auth *xnetproxy.Auth
		if u.User != nil {
			pw, _ := u.User.Password()
			auth = &xnetproxy.Auth{User: u.User.Username(), Password: pw}
		}
		dialer, err := xnetproxy.SOCKS5("tcp", u.Host, auth, xnetproxy.Direct)
		if err != nil {
			return nil, fmt.Errorf("unsupported socks proxy: %w", err)
		}
		ctxDialer, ok := dialer.(xnetproxy.ContextDialer)
		if ok {
			transport.DialContext = ctxDialer.DialContext
		} else {
			transport.DialContext = func(ctx context.Context, network, address string) (net.Conn, error) {
				type result struct {
					conn net.Conn
					err  error
				}
				ch := make(chan result, 1)
				go func() { c, e := dialer.Dial(network, address); ch <- result{c, e} }()
				select {
				case <-ctx.Done():
					return nil, ctx.Err()
				case r := <-ch:
					return r.conn, r.err
				}
			}
		}
	default:
		return nil, fmt.Errorf("unsupported proxy scheme %q (supported: http, https, socks5, socks5h)", scheme)
	}
	return transport, nil
}

type mailTMProvider struct {
	chatGPTRegisterBaseMailProvider
}

func (p *mailTMProvider) Name() string { return "mailtm" }
func (p *mailTMProvider) CreateMailbox(ctx context.Context, username string) (*tempMailbox, error) {
	base := p.apiBase("https://api.mail.tm")
	headers := map[string]string{}
	if p.entry.APIKey != "" {
		headers["Authorization"] = "Bearer " + p.entry.APIKey
	}
	data, _, _, err := p.requestJSON(ctx, http.MethodGet, base+"/domains", headers, nil, nil, 200)
	if err != nil {
		return nil, err
	}
	items := itemsFromAny(mapAny(data)["hydra:member"], "hydra:member")
	if len(items) == 0 {
		items = itemsFromAny(data, "hydra:member", "data", "domains")
	}
	if len(items) == 0 {
		return nil, fmt.Errorf("no domains available")
	}
	domain := stringAny(items[0], "domain", "name")
	if domain == "" {
		return nil, fmt.Errorf("mail.tm domain missing")
	}
	address := registerFirstNonEmpty(username, randomMailboxName()) + "@" + domain
	password := chatGPTRegisterRandomPassword(12)
	_, _, status, err := p.requestJSON(ctx, http.MethodPost, base+"/accounts", headers, nil, map[string]string{"address": address, "password": password}, 200, 201)
	if err != nil {
		return nil, fmt.Errorf("create mailbox HTTP %d: %w", status, err)
	}
	return &tempMailbox{Email: address, Password: password, Provider: p.Name(), ProviderRef: p.ProviderRef(), Label: p.entry.Label}, nil
}
func (p *mailTMProvider) FetchLatestMessage(ctx context.Context, mailbox *tempMailbox) (*chatGPTRegisterMailMessage, error) {
	base := p.apiBase("https://api.mail.tm")
	if mailbox.Token == "" {
		data, _, _, err := p.requestJSON(ctx, http.MethodPost, base+"/token", nil, nil, map[string]string{"address": mailbox.Email, "password": mailbox.Password}, 200)
		if err != nil {
			return nil, err
		}
		mailbox.Token = stringAny(mapAny(data), "token")
	}
	if mailbox.Token == "" {
		return nil, fmt.Errorf("empty mail token")
	}
	data, _, _, err := p.requestJSON(ctx, http.MethodGet, base+"/messages", map[string]string{"Authorization": "Bearer " + mailbox.Token}, nil, nil, 200)
	if err != nil {
		return nil, err
	}
	items := itemsFromAny(data, "hydra:member", "data", "messages")
	if len(items) == 0 {
		return nil, nil
	}
	item := items[0]
	id := strings.TrimPrefix(stringAny(item, "id", "@id"), "/messages/")
	if id != "" {
		if detail, _, _, err := p.requestJSON(ctx, http.MethodGet, base+"/messages/"+id, map[string]string{"Authorization": "Bearer " + mailbox.Token}, nil, nil, 200); err == nil {
			if m := mapAny(detail); m != nil {
				item = m
			}
		}
	}
	text, html := extractContent(item)
	return &chatGPTRegisterMailMessage{Provider: p.Name(), Mailbox: mailbox.Email, MessageID: id, Subject: stringAny(item, "subject"), Sender: stringAny(item, "from", "sender"), TextContent: text, HTMLContent: html, ReceivedAt: parseMessageTime(firstMessageTimeValue(item)), Raw: item}, nil
}

type cloudflareTempEmailProvider struct {
	chatGPTRegisterBaseMailProvider
}

func (p *cloudflareTempEmailProvider) Name() string { return "cloudflare_temp_email" }
func (p *cloudflareTempEmailProvider) CreateMailbox(ctx context.Context, username string) (*tempMailbox, error) {
	domain, err := nextRegisterDomain(p.entry.Domain)
	if err != nil {
		return nil, err
	}
	data, _, _, err := p.requestJSON(ctx, http.MethodPost, p.entry.APIBase+"/admin/new_address", map[string]string{"x-admin-auth": p.entry.AdminPassword}, nil, map[string]any{"enablePrefix": true, "name": registerFirstNonEmpty(username, randomMailboxName()), "domain": domain}, 200)
	if err != nil {
		return nil, err
	}
	m := mapAny(data)
	address := stringAny(m, "address")
	token := stringAny(m, "jwt")
	if address == "" || token == "" {
		return nil, fmt.Errorf("CloudflareTempMail 缺少 address 或 jwt")
	}
	return &tempMailbox{Email: address, Token: token, Provider: p.Name(), ProviderRef: p.ProviderRef(), Label: p.entry.Label}, nil
}
func (p *cloudflareTempEmailProvider) FetchLatestMessage(ctx context.Context, mailbox *tempMailbox) (*chatGPTRegisterMailMessage, error) {
	data, _, _, err := p.requestJSON(ctx, http.MethodGet, p.entry.APIBase+"/api/mails", map[string]string{"Authorization": "Bearer " + mailbox.Token}, map[string]string{"limit": "10", "offset": "0"}, nil, 200)
	if err != nil {
		return nil, err
	}
	items := itemsFromAny(data, "results", "data", "messages")
	for _, item := range items {
		if !messageMatchesEmail(item, mailbox.Email) {
			continue
		}
		text, html := extractContent(item)
		sender := stringAny(item, "from", "sender")
		return &chatGPTRegisterMailMessage{Provider: p.Name(), Mailbox: mailbox.Email, MessageID: stringAny(item, "id", "_id"), Subject: stringAny(item, "subject"), Sender: sender, TextContent: text, HTMLContent: html, ReceivedAt: parseMessageTime(firstMessageTimeValue(item)), Raw: item}, nil
	}
	return nil, nil
}

type tempMailLolProvider struct {
	chatGPTRegisterBaseMailProvider
}

func (p *tempMailLolProvider) Name() string { return "tempmail_lol" }
func (p *tempMailLolProvider) CreateMailbox(ctx context.Context, username string) (*tempMailbox, error) {
	payload := map[string]any{}
	if len(p.entry.Domain) > 0 {
		d := p.entry.Domain[randIndex(len(p.entry.Domain))]
		if strings.HasPrefix(d, "*.") {
			payload["domain"] = randomSubdomainLabel() + "." + strings.TrimPrefix(d, "*.")
			payload["prefix"] = randomMailboxName()
		} else {
			payload["domain"] = d
		}
	}
	if username != "" && payload["prefix"] == nil {
		payload["prefix"] = username
	}
	headers := map[string]string{}
	if p.entry.APIKey != "" {
		headers["Authorization"] = "Bearer " + p.entry.APIKey
	}
	data, _, _, err := p.requestJSON(ctx, http.MethodPost, p.apiBase("https://api.tempmail.lol/v2")+"/inbox/create", headers, nil, payload, 200, 201)
	if err != nil {
		return nil, err
	}
	m := mapAny(data)
	address := stringAny(m, "address")
	token := stringAny(m, "token")
	if address == "" || token == "" {
		return nil, fmt.Errorf("TempMail.lol 缺少 address 或 token")
	}
	return &tempMailbox{Email: address, Token: token, Provider: p.Name(), ProviderRef: p.ProviderRef(), Label: p.entry.Label}, nil
}
func (p *tempMailLolProvider) FetchLatestMessage(ctx context.Context, mailbox *tempMailbox) (*chatGPTRegisterMailMessage, error) {
	data, _, _, err := p.requestJSON(ctx, http.MethodGet, p.apiBase("https://api.tempmail.lol/v2")+"/inbox", nil, map[string]string{"token": mailbox.Token}, nil, 200)
	if err != nil {
		return nil, err
	}
	item := latestItem(itemsFromAny(data, "emails", "messages"))
	if item == nil {
		return nil, nil
	}
	text, html := extractContent(item)
	return &chatGPTRegisterMailMessage{Provider: p.Name(), Mailbox: mailbox.Email, MessageID: stringAny(item, "id", "token"), Subject: stringAny(item, "subject"), Sender: stringAny(item, "from", "from_address"), TextContent: text, HTMLContent: html, ReceivedAt: parseMessageTime(firstMessageTimeValue(item)), Raw: item}, nil
}

type inbucketProvider struct {
	chatGPTRegisterBaseMailProvider
}

func (p *inbucketProvider) Name() string { return "inbucket" }
func (p *inbucketProvider) CreateMailbox(ctx context.Context, username string) (*tempMailbox, error) {
	domain, err := nextRegisterDomain(p.entry.Domain)
	if err != nil {
		return nil, err
	}
	if p.entry.RandomSubdomain == nil || *p.entry.RandomSubdomain {
		domain = randomSubdomainLabel() + "." + domain
	}
	local := registerFirstNonEmpty(username, randomMailboxName())
	return &tempMailbox{Email: local + "@" + domain, MailboxName: local, Provider: p.Name(), ProviderRef: p.ProviderRef(), Label: p.entry.Label}, nil
}
func (p *inbucketProvider) FetchLatestMessage(ctx context.Context, mailbox *tempMailbox) (*chatGPTRegisterMailMessage, error) {
	data, _, _, err := p.requestJSON(ctx, http.MethodGet, p.entry.APIBase+"/api/v1/mailbox/"+url.PathEscape(mailbox.MailboxName), nil, nil, nil, 200)
	if err != nil {
		return nil, err
	}
	items := itemsFromAny(data, "data")
	item := latestItem(items)
	if item == nil {
		return nil, nil
	}
	id := stringAny(item, "id")
	detail, _, _, err := p.requestJSON(ctx, http.MethodGet, p.entry.APIBase+"/api/v1/mailbox/"+url.PathEscape(mailbox.MailboxName)+"/"+url.PathEscape(id), nil, nil, nil, 200)
	if err != nil {
		return nil, err
	}
	m := mapAny(detail)
	body := mapAny(m["body"])
	header := mapAny(m["header"])
	normalized := map[string]any{"to": header["To"]}
	if !messageMatchesEmail(normalized, mailbox.Email) {
		return nil, nil
	}
	return &chatGPTRegisterMailMessage{Provider: p.Name(), Mailbox: mailbox.Email, MessageID: id, Subject: stringAny(m, "subject"), Sender: stringAny(m, "from"), TextContent: stringAny(body, "text"), HTMLContent: stringAny(body, "html"), ReceivedAt: parseMessageTime(firstMessageTimeValue(m)), Raw: m}, nil
}

type moEmailProvider struct {
	chatGPTRegisterBaseMailProvider
}

func (p *moEmailProvider) Name() string { return "moemail" }
func (p *moEmailProvider) CreateMailbox(ctx context.Context, username string) (*tempMailbox, error) {
	domain, err := nextRegisterDomain(p.entry.Domain)
	if err != nil {
		return nil, err
	}
	data, _, _, err := p.requestJSON(ctx, http.MethodPost, p.entry.APIBase+"/api/emails/generate", map[string]string{"X-API-Key": p.entry.APIKey}, nil, map[string]any{"name": registerFirstNonEmpty(username, randomMailboxName()), "expiryTime": p.entry.ExpiryTime, "domain": domain}, 200, 201)
	if err != nil {
		return nil, err
	}
	m := mapAny(data)
	address := stringAny(m, "email")
	id := stringAny(m, "id", "email_id")
	if address == "" || id == "" {
		return nil, fmt.Errorf("MoEmail 缺少 email 或 id")
	}
	return &tempMailbox{Email: address, EmailID: id, Provider: p.Name(), ProviderRef: p.ProviderRef(), Label: p.entry.Label}, nil
}
func (p *moEmailProvider) FetchLatestMessage(ctx context.Context, mailbox *tempMailbox) (*chatGPTRegisterMailMessage, error) {
	headers := map[string]string{"X-API-Key": p.entry.APIKey}
	data, _, _, err := p.requestJSON(ctx, http.MethodGet, p.entry.APIBase+"/api/emails/"+url.PathEscape(mailbox.EmailID), headers, nil, nil, 200)
	if err != nil {
		return nil, err
	}
	item := latestItem(itemsFromAny(mapAny(data)["messages"], "messages"))
	if item == nil {
		return nil, nil
	}
	id := stringAny(item, "id", "message_id", "_id")
	detail := any(item)
	if id != "" {
		if d, _, _, e := p.requestJSON(ctx, http.MethodGet, p.entry.APIBase+"/api/emails/"+url.PathEscape(mailbox.EmailID)+"/"+url.PathEscape(id), headers, nil, nil, 200); e == nil {
			detail = d
		}
	}
	m := mapAny(detail)
	if mm := mapAny(m["message"]); mm != nil {
		m = mm
	}
	text, html := extractContent(m)
	return &chatGPTRegisterMailMessage{Provider: p.Name(), Mailbox: mailbox.Email, MessageID: id, Subject: registerFirstNonEmpty(stringAny(m, "subject"), stringAny(item, "subject")), Sender: stringAny(m, "from", "sender"), TextContent: text, HTMLContent: html, ReceivedAt: parseMessageTime(firstMessageTimeValue(m)), Raw: detail}, nil
}

type cloudMailGenProvider struct {
	chatGPTRegisterBaseMailProvider
}

func (p *cloudMailGenProvider) Name() string { return "cloudmail_gen" }
func (p *cloudMailGenProvider) CreateMailbox(ctx context.Context, username string) (*tempMailbox, error) {
	domain, err := nextRegisterDomain(p.entry.Domain)
	if err != nil {
		return nil, err
	}
	if len(p.entry.Subdomain) > 0 {
		domain = p.entry.Subdomain[randIndex(len(p.entry.Subdomain))] + "." + domain
	}
	local := registerFirstNonEmpty(username)
	if local == "" && p.entry.EmailPrefix != "" {
		local = p.entry.EmailPrefix + "_" + randomLower(6)
	}
	if local == "" {
		local = randomMailboxName()
	}
	return &tempMailbox{Email: local + "@" + domain, Provider: p.Name(), ProviderRef: p.ProviderRef(), Label: p.entry.Label}, nil
}
func (p *cloudMailGenProvider) token(ctx context.Context) (string, error) {
	data, _, _, err := p.requestJSON(ctx, http.MethodPost, p.entry.APIBase+"/api/public/genToken", nil, nil, map[string]string{"email": p.entry.AdminEmail, "password": p.entry.AdminPassword}, 200)
	if err != nil {
		return "", err
	}
	m := mapAny(data)
	dm := mapAny(m["data"])
	token := stringAny(dm, "token")
	if token == "" {
		return "", fmt.Errorf("CloudMailGen genToken 返回异常")
	}
	return token, nil
}
func (p *cloudMailGenProvider) FetchLatestMessage(ctx context.Context, mailbox *tempMailbox) (*chatGPTRegisterMailMessage, error) {
	token, err := p.token(ctx)
	if err != nil {
		return nil, err
	}
	data, _, _, err := p.requestJSON(ctx, http.MethodPost, p.entry.APIBase+"/api/public/emailList", map[string]string{"Authorization": token}, nil, map[string]any{"toEmail": mailbox.Email, "size": 20, "timeSort": "desc"}, 200)
	if err != nil {
		return nil, err
	}
	m := mapAny(data)
	item := latestItem(itemsFromAny(m["data"], "data"))
	if item == nil || !messageMatchesEmail(item, mailbox.Email) {
		return nil, nil
	}
	text, html := extractContent(item)
	return &chatGPTRegisterMailMessage{Provider: p.Name(), Mailbox: mailbox.Email, MessageID: stringAny(item, "id", "_id", "messageId"), Subject: stringAny(item, "subject"), Sender: stringAny(item, "from", "sender"), TextContent: text, HTMLContent: html, ReceivedAt: parseMessageTime(firstMessageTimeValue(item)), Raw: item}, nil
}

type ddgMailProvider struct {
	chatGPTRegisterBaseMailProvider
}

func (p *ddgMailProvider) Name() string { return "ddg_mail" }
func (p *ddgMailProvider) CreateMailbox(ctx context.Context, username string) (*tempMailbox, error) {
	data, _, _, err := p.requestJSON(ctx, http.MethodPost, "https://quack.duckduckgo.com/api/email/addresses", map[string]string{"Authorization": "Bearer " + p.entry.DDGToken}, nil, map[string]any{}, 200, 201)
	if err != nil {
		return nil, err
	}
	part := stringAny(mapAny(data), "address")
	if part == "" {
		return nil, fmt.Errorf("DDG API 返回无 address 字段")
	}
	if p.entry.CFInboxJWT == "" {
		return nil, fmt.Errorf("DDGMail 需要 cf_inbox_jwt")
	}
	return &tempMailbox{Email: part + "@duck.com", Token: p.entry.CFInboxJWT, Provider: p.Name(), ProviderRef: p.ProviderRef(), Label: p.entry.Label}, nil
}
func (p *ddgMailProvider) cfHeaders(method string) map[string]string {
	h := map[string]string{}
	key := p.entry.CFAPIKey
	mode := p.entry.CFAuthMode
	if key != "" {
		if mode == "x-api-key" {
			h["X-API-Key"] = key
		} else if mode != "none" && mode != "query-key" {
			h["Authorization"] = "Bearer " + key
		}
	}
	if p.entry.AdminPassword != "" && method == http.MethodPost {
		h["x-admin-auth"] = p.entry.AdminPassword
	}
	return h
}
func (p *ddgMailProvider) FetchLatestMessage(ctx context.Context, mailbox *tempMailbox) (*chatGPTRegisterMailMessage, error) {
	params := map[string]string{"limit": "30", "offset": "0"}
	if p.entry.CFAPIKey != "" && p.entry.CFAuthMode == "query-key" {
		params["key"] = p.entry.CFAPIKey
	}
	h := p.cfHeaders(http.MethodGet)
	h["Authorization"] = "Bearer " + mailbox.Token
	base := registerFirstNonEmpty(p.entry.CFAPIBase, p.entry.APIBase)
	data, _, _, err := p.requestJSON(ctx, http.MethodGet, base+p.entry.CFMessagesPath, h, params, nil, 200)
	if err != nil {
		return nil, err
	}
	for _, item := range itemsFromAny(data, "results", "hydra:member", "data", "messages") {
		raw := stringAny(item, "raw")
		if raw != "" && !strings.Contains(strings.ToLower(raw), strings.ToLower(mailbox.Email)) {
			continue
		}
		text, html := extractContent(item)
		return &chatGPTRegisterMailMessage{Provider: p.Name(), Mailbox: mailbox.Email, MessageID: stringAny(item, "id", "msgid", "_id"), Subject: stringAny(item, "subject"), Sender: stringAny(item, "from", "sender", "source"), TextContent: text, HTMLContent: html, ReceivedAt: parseMessageTime(firstMessageTimeValue(item)), Raw: item}, nil
	}
	return nil, nil
}

type duckMailProvider struct {
	chatGPTRegisterBaseMailProvider
}

func (p *duckMailProvider) Name() string { return "duckmail" }
func (p *duckMailProvider) CreateMailbox(ctx context.Context, username string) (*tempMailbox, error) {
	domain := registerFirstNonEmpty(p.entry.DefaultDomain, "duckmail.sbs")
	password := chatGPTRegisterRandomPassword(12)
	address := registerFirstNonEmpty(username, randomMailboxName()) + "@" + domain
	payload := map[string]string{"address": address, "password": password}
	account, _, _, err := p.requestJSON(ctx, http.MethodPost, p.apiBase("https://api.duckmail.sbs")+"/accounts", map[string]string{"Authorization": "Bearer " + p.entry.APIKey}, nil, payload, 200, 201, 204)
	if err != nil {
		return nil, err
	}
	tokenData, _, _, err := p.requestJSON(ctx, http.MethodPost, p.apiBase("https://api.duckmail.sbs")+"/token", map[string]string{"Authorization": "Bearer " + p.entry.APIKey}, nil, payload, 200, 201)
	if err != nil {
		return nil, err
	}
	return &tempMailbox{Email: address, Password: password, Token: stringAny(mapAny(tokenData), "token"), ID: stringAny(mapAny(account), "id"), Provider: p.Name(), ProviderRef: p.ProviderRef(), Label: p.entry.Label}, nil
}
func (p *duckMailProvider) FetchLatestMessage(ctx context.Context, mailbox *tempMailbox) (*chatGPTRegisterMailMessage, error) {
	h := map[string]string{"Authorization": "Bearer " + mailbox.Token}
	data, _, _, err := p.requestJSON(ctx, http.MethodGet, p.apiBase("https://api.duckmail.sbs")+"/messages", h, map[string]string{"page": "1"}, nil, 200)
	if err != nil {
		return nil, err
	}
	item := latestItem(itemsFromAny(data, "hydra:member", "member", "data"))
	if item == nil {
		return nil, nil
	}
	id := strings.TrimPrefix(stringAny(item, "id", "@id"), "/messages/")
	if id != "" {
		if detail, _, _, e := p.requestJSON(ctx, http.MethodGet, p.apiBase("https://api.duckmail.sbs")+"/messages/"+url.PathEscape(id), h, nil, nil, 200); e == nil {
			item = mapAny(detail)
		}
	}
	text, html := extractContent(item)
	return &chatGPTRegisterMailMessage{Provider: p.Name(), Mailbox: mailbox.Email, MessageID: id, Subject: stringAny(item, "subject"), Sender: stringAny(item, "from"), TextContent: text, HTMLContent: html, ReceivedAt: parseMessageTime(firstMessageTimeValue(item)), Raw: item}, nil
}

type gptMailProvider struct {
	chatGPTRegisterBaseMailProvider
}

func (p *gptMailProvider) Name() string { return "gptmail" }
func (p *gptMailProvider) request(ctx context.Context, method, path string, params map[string]string, payload any) (any, error) {
	data, _, _, err := p.requestJSON(ctx, method, p.apiBase("https://mail.chatgpt.org.uk")+path, map[string]string{"X-API-Key": p.entry.APIKey}, params, payload, 200)
	if err != nil {
		return nil, err
	}
	if m := mapAny(data); m != nil {
		if d, ok := m["data"]; ok {
			return d, nil
		}
	}
	return data, nil
}
func (p *gptMailProvider) CreateMailbox(ctx context.Context, username string) (*tempMailbox, error) {
	payload := map[string]any{}
	if username != "" {
		payload["prefix"] = username
	}
	if p.entry.DefaultDomain != "" {
		payload["domain"] = p.entry.DefaultDomain
	}
	method := http.MethodGet
	var body any
	if len(payload) > 0 {
		method = http.MethodPost
		body = payload
	}
	data, err := p.request(ctx, method, "/api/generate-email", nil, body)
	if err != nil {
		return nil, err
	}
	return &tempMailbox{Email: stringAny(mapAny(data), "email"), Provider: p.Name(), ProviderRef: p.ProviderRef(), Label: p.entry.Label}, nil
}
func (p *gptMailProvider) FetchLatestMessage(ctx context.Context, mailbox *tempMailbox) (*chatGPTRegisterMailMessage, error) {
	data, err := p.request(ctx, http.MethodGet, "/api/emails", map[string]string{"email": mailbox.Email}, nil)
	if err != nil {
		return nil, err
	}
	item := latestItem(itemsFromAny(data, "emails"))
	if item == nil {
		return nil, nil
	}
	id := stringAny(item, "id")
	if id != "" {
		if detail, e := p.request(ctx, http.MethodGet, "/api/email/"+url.PathEscape(id), nil, nil); e == nil {
			item = mapAny(detail)
		}
	}
	return &chatGPTRegisterMailMessage{Provider: p.Name(), Mailbox: mailbox.Email, MessageID: id, Subject: stringAny(item, "subject"), Sender: stringAny(item, "from_address"), TextContent: stringAny(item, "content"), HTMLContent: stringAny(item, "html_content"), ReceivedAt: parseMessageTime(firstMessageTimeValue(item)), Raw: item}, nil
}

type yydsMailProvider struct {
	chatGPTRegisterBaseMailProvider
}

func (p *yydsMailProvider) Name() string { return "yyds_mail" }
func (p *yydsMailProvider) request(ctx context.Context, method, path, token string, params map[string]string, payload any) (any, error) {
	h := map[string]string{"X-API-Key": p.entry.APIKey}
	if token != "" {
		h = map[string]string{"Authorization": "Bearer " + token}
	}
	data, _, _, err := p.requestJSON(ctx, method, p.apiBase("https://maliapi.215.im/v1")+path, h, params, payload, 200, 201, 204)
	if err != nil {
		return nil, err
	}
	if m := mapAny(data); m != nil {
		if b, ok := m["success"].(bool); ok && !b {
			return nil, fmt.Errorf("YYDSMail 请求失败: %s", stringAny(m, "errorCode", "error"))
		}
		if d, ok := m["data"]; ok {
			return d, nil
		}
	}
	return data, nil
}
func (p *yydsMailProvider) CreateMailbox(ctx context.Context, username string) (*tempMailbox, error) {
	payload := map[string]any{"localPart": registerFirstNonEmpty(username, randomMailboxName())}
	if len(p.entry.Domain) > 0 {
		d, _ := nextRegisterDomain(p.entry.Domain)
		payload["domain"] = d
	}
	if len(p.entry.Subdomain) > 0 {
		payload["subdomain"] = p.entry.Subdomain[0]
	}
	path := "/accounts"
	if p.entry.Wildcard {
		path = "/accounts/wildcard"
	}
	data, err := p.request(ctx, http.MethodPost, path, "", nil, payload)
	if err != nil {
		return nil, err
	}
	m := mapAny(data)
	address := stringAny(m, "address", "email")
	token := stringAny(m, "token", "temp_token", "tempToken", "access_token")
	if address == "" || token == "" {
		return nil, fmt.Errorf("YYDSMail 缺少 address 或 token")
	}
	return &tempMailbox{Email: address, Token: token, ID: stringAny(m, "id"), Provider: p.Name(), ProviderRef: p.ProviderRef(), Label: p.entry.Label}, nil
}
func (p *yydsMailProvider) FetchLatestMessage(ctx context.Context, mailbox *tempMailbox) (*chatGPTRegisterMailMessage, error) {
	data, err := p.request(ctx, http.MethodGet, "/messages", mailbox.Token, map[string]string{"address": mailbox.Email}, nil)
	if err != nil {
		return nil, err
	}
	item := latestItem(itemsFromAny(data, "items", "messages", "data"))
	if item == nil {
		return nil, nil
	}
	id := stringAny(item, "id", "message_id")
	if id != "" {
		if detail, e := p.request(ctx, http.MethodGet, "/messages/"+url.PathEscape(id), mailbox.Token, map[string]string{"address": mailbox.Email}, nil); e == nil {
			item = mapAny(detail)
		}
	}
	text, html := extractContent(item)
	return &chatGPTRegisterMailMessage{Provider: p.Name(), Mailbox: mailbox.Email, MessageID: id, Subject: stringAny(item, "subject"), Sender: stringAny(item, "from", "sender"), TextContent: text, HTMLContent: html, ReceivedAt: parseMessageTime(firstMessageTimeValue(item)), Raw: item}, nil
}

func randIndex(n int) int {
	if n <= 1 {
		return 0
	}
	return int(time.Now().UnixNano() % int64(n))
}

var _ = json.Valid
