package service

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha3"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/google/uuid"
)

const (
	chatGPTWebBaseURL           = "https://chatgpt.com"
	chatGPTWebClientVersion     = "prod-be885abbfcfe7b1f511e88b3003d9ee44757fbad"
	chatGPTWebClientBuildNumber = "5955942"
	chatGPTWebDefaultPowScript  = "https://chatgpt.com/backend-api/sentinel/sdk.js"
	chatGPTWebDefaultUserAgent  = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36 Edg/143.0.0.0"
)

// ChatGPTWebClient implements native ChatGPT Web backend-api conversation.
type ChatGPTWebClient struct {
	service     *OpenAIGatewayService
	account     *Account
	accessToken string
	proxyURL    string

	userAgent     string
	deviceID      string
	sessionID     string
	scriptSources []string
	dataBuild     string
}

type chatGPTWebRequirements struct {
	Token          string
	ProofToken     string
	TurnstileToken string
	SOToken        string
}

func newChatGPTWebClient(service *OpenAIGatewayService, account *Account, accessToken string) *ChatGPTWebClient {
	userAgent := strings.TrimSpace(account.GetChatGPTWebUserAgent())
	if userAgent == "" {
		userAgent = chatGPTWebDefaultUserAgent
	}
	deviceID := strings.TrimSpace(account.GetChatGPTWebDeviceID())
	if deviceID == "" {
		deviceID = uuid.NewString()
	}
	sessionID := strings.TrimSpace(account.GetChatGPTWebSessionID())
	if sessionID == "" {
		sessionID = uuid.NewString()
	}
	proxyURL := ""
	if account != nil && account.Proxy != nil {
		proxyURL = account.Proxy.URL()
	}
	return &ChatGPTWebClient{
		service:     service,
		account:     account,
		accessToken: accessToken,
		proxyURL:    proxyURL,
		userAgent:   userAgent,
		deviceID:    deviceID,
		sessionID:   sessionID,
	}
}

func (c *ChatGPTWebClient) do(req *http.Request) (*http.Response, error) {
	if c == nil || c.service == nil || c.service.httpUpstream == nil {
		return nil, fmt.Errorf("chatgpt web http upstream is not configured")
	}
	return c.service.httpUpstream.Do(req, c.proxyURL, c.account.ID, c.account.Concurrency)
}

func (c *ChatGPTWebClient) bootstrap(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, chatGPTWebBaseURL+"/", nil)
	if err != nil {
		return err
	}
	for key, value := range c.bootstrapHeaders() {
		req.Header.Set(key, value)
	}
	req = req.WithContext(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI))
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4<<20))
	if resp.StatusCode >= 400 {
		return fmt.Errorf("chatgpt bootstrap failed: HTTP %d", resp.StatusCode)
	}
	c.scriptSources, c.dataBuild = parseChatGPTWebPowResources(string(body))
	if len(c.scriptSources) == 0 {
		c.scriptSources = []string{chatGPTWebDefaultPowScript}
	}
	return nil
}

func (c *ChatGPTWebClient) bootstrapHeaders() map[string]string {
	return map[string]string{
		"User-Agent":                c.userAgent,
		"Accept":                    "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
		"Accept-Language":           "zh-CN,zh;q=0.9,en;q=0.8",
		"Sec-Ch-Ua":                 `"Microsoft Edge";v="143", "Chromium";v="143", "Not A(Brand";v="24"`,
		"Sec-Ch-Ua-Mobile":          "?0",
		"Sec-Ch-Ua-Platform":        `"Windows"`,
		"Sec-Fetch-Dest":            "document",
		"Sec-Fetch-Mode":            "navigate",
		"Sec-Fetch-Site":            "none",
		"Sec-Fetch-User":            "?1",
		"Upgrade-Insecure-Requests": "1",
	}
}

func (c *ChatGPTWebClient) headers(path string, extra map[string]string) map[string]string {
	h := map[string]string{
		"User-Agent":                  c.userAgent,
		"Origin":                      chatGPTWebBaseURL,
		"Referer":                     chatGPTWebBaseURL + "/",
		"Accept-Language":             "zh-CN,zh;q=0.9,en;q=0.8,en-US;q=0.7",
		"Cache-Control":               "no-cache",
		"Pragma":                      "no-cache",
		"Priority":                    "u=1, i",
		"Sec-Ch-Ua":                   `"Microsoft Edge";v="143", "Chromium";v="143", "Not A(Brand";v="24"`,
		"Sec-Ch-Ua-Arch":              `"x86"`,
		"Sec-Ch-Ua-Bitness":           `"64"`,
		"Sec-Ch-Ua-Full-Version":      `"143.0.3650.96"`,
		"Sec-Ch-Ua-Full-Version-List": `"Microsoft Edge";v="143.0.3650.96", "Chromium";v="143.0.7499.147", "Not A(Brand";v="24.0.0.0"`,
		"Sec-Ch-Ua-Mobile":            "?0",
		"Sec-Ch-Ua-Model":             `""`,
		"Sec-Ch-Ua-Platform":          `"Windows"`,
		"Sec-Ch-Ua-Platform-Version":  `"19.0.0"`,
		"Sec-Fetch-Dest":              "empty",
		"Sec-Fetch-Mode":              "cors",
		"Sec-Fetch-Site":              "same-origin",
		"OAI-Device-Id":               c.deviceID,
		"OAI-Session-Id":              c.sessionID,
		"OAI-Language":                "zh-CN",
		"OAI-Client-Version":          chatGPTWebClientVersion,
		"OAI-Client-Build-Number":     chatGPTWebClientBuildNumber,
		"X-OpenAI-Target-Path":        path,
		"X-OpenAI-Target-Route":       path,
	}
	if strings.TrimSpace(c.accessToken) != "" {
		h["Authorization"] = "Bearer " + c.accessToken
	}
	for k, v := range extra {
		h[k] = v
	}
	return h
}

func (c *ChatGPTWebClient) getChatRequirements(ctx context.Context) (*chatGPTWebRequirements, error) {
	path := "/backend-api/sentinel/chat-requirements"
	p := buildChatGPTWebLegacyRequirementsToken(c.userAgent, c.scriptSources, c.dataBuild)
	payload, _ := json.Marshal(map[string]any{"p": p})
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chatGPTWebBaseURL+path, bytes.NewReader(payload))
	if err != nil {
		return nil, err
	}
	for k, v := range c.headers(path, map[string]string{"Content-Type": "application/json"}) {
		req.Header.Set(k, v)
	}
	req = req.WithContext(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI))
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("chat requirements failed: HTTP %d: %s", resp.StatusCode, truncateString(string(body), 300))
	}
	var data map[string]any
	if err := json.Unmarshal(body, &data); err != nil {
		return nil, err
	}
	reqInfo := &chatGPTWebRequirements{Token: stringMapVal(data, "token"), SOToken: stringMapVal(data, "so_token")}
	if proof, _ := data["proofofwork"].(map[string]any); boolMapVal(proof, "required") {
		token, err := buildChatGPTWebProofToken(stringMapVal(proof, "seed"), stringMapVal(proof, "difficulty"), c.userAgent, c.scriptSources, c.dataBuild)
		if err != nil {
			return nil, err
		}
		reqInfo.ProofToken = token
	}
	if strings.TrimSpace(reqInfo.Token) == "" {
		return nil, fmt.Errorf("missing chat requirements token")
	}
	return reqInfo, nil
}

// StreamConversation sends a ChatGPT Web conversation and streams text deltas via onDelta.
func (c *ChatGPTWebClient) StreamConversation(ctx context.Context, messages []apicompat.ChatMessage, model string, onDelta func(string) error) error {
	if err := c.bootstrap(ctx); err != nil {
		return err
	}
	reqInfo, err := c.getChatRequirements(ctx)
	if err != nil {
		return err
	}
	path := "/backend-api/conversation"
	payload, err := json.Marshal(c.buildConversationPayload(messages, model))
	if err != nil {
		return err
	}
	extra := map[string]string{
		"Accept":       "text/event-stream",
		"Content-Type": "application/json",
		"OpenAI-Sentinel-Chat-Requirements-Token": reqInfo.Token,
	}
	if reqInfo.ProofToken != "" {
		extra["OpenAI-Sentinel-Proof-Token"] = reqInfo.ProofToken
	}
	if reqInfo.TurnstileToken != "" {
		extra["OpenAI-Sentinel-Turnstile-Token"] = reqInfo.TurnstileToken
	}
	if reqInfo.SOToken != "" {
		extra["OpenAI-Sentinel-SO-Token"] = reqInfo.SOToken
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, chatGPTWebBaseURL+path, bytes.NewReader(payload))
	if err != nil {
		return err
	}
	for k, v := range c.headers(path, extra) {
		req.Header.Set(k, v)
	}
	req = req.WithContext(WithHTTPUpstreamProfile(req.Context(), HTTPUpstreamProfileOpenAI))
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
		return fmt.Errorf("chatgpt conversation failed: HTTP %d: %s", resp.StatusCode, truncateString(string(body), 300))
	}
	return scanChatGPTWebConversationDeltas(resp.Body, onDelta)
}

func (c *ChatGPTWebClient) buildConversationPayload(messages []apicompat.ChatMessage, model string) map[string]any {
	return map[string]any{
		"action":                        "next",
		"messages":                      chatGPTWebConversationMessages(messages),
		"model":                         strings.TrimSpace(model),
		"parent_message_id":             uuid.NewString(),
		"conversation_mode":             map[string]any{"kind": "primary_assistant"},
		"conversation_origin":           nil,
		"force_paragen":                 false,
		"force_paragen_model_slug":      "",
		"force_rate_limit":              false,
		"force_use_sse":                 true,
		"history_and_training_disabled": true,
		"reset_rate_limits":             false,
		"suggestions":                   []any{},
		"supported_encodings":           []any{},
		"system_hints":                  []any{},
		"timezone":                      "Asia/Shanghai",
		"timezone_offset_min":           -480,
		"variant_purpose":               "comparison_implicit",
		"websocket_request_id":          uuid.NewString(),
		"client_contextual_info": map[string]any{
			"is_dark_mode":      false,
			"time_since_loaded": 120,
			"page_height":       900,
			"page_width":        1400,
			"pixel_ratio":       2,
			"screen_height":     1440,
			"screen_width":      2560,
		},
	}
}

func chatGPTWebConversationMessages(messages []apicompat.ChatMessage) []map[string]any {
	out := make([]map[string]any, 0, len(messages))
	for _, msg := range messages {
		role := strings.TrimSpace(msg.Role)
		if role == "" {
			role = "user"
		}
		text := chatGPTWebExtractText(msg.Content)
		out = append(out, map[string]any{
			"id":      uuid.NewString(),
			"author":  map[string]any{"role": role},
			"content": map[string]any{"content_type": "text", "parts": []string{text}},
		})
	}
	return out
}

func chatGPTWebExtractText(raw json.RawMessage) string {
	if len(raw) == 0 || string(raw) == "null" {
		return ""
	}
	var s string
	if err := json.Unmarshal(raw, &s); err == nil {
		return s
	}
	var parts []apicompat.ChatContentPart
	if err := json.Unmarshal(raw, &parts); err == nil {
		var b strings.Builder
		for _, part := range parts {
			if part.Type == "text" {
				_, _ = b.WriteString(part.Text)
			}
		}
		return b.String()
	}
	return string(raw)
}

// PoW / bootstrap helpers

var chatGPTWebScriptSrcRE = regexp.MustCompile(`<script[^>]+src="([^"]+)"`)
var chatGPTWebDataBuildRE = regexp.MustCompile(`<html[^>]*data-build="([^"]*)"`)
var chatGPTWebCBuildRE = regexp.MustCompile(`c/[^/]*/_`)

func parseChatGPTWebPowResources(html string) ([]string, string) {
	matches := chatGPTWebScriptSrcRE.FindAllStringSubmatch(html, -1)
	sources := make([]string, 0, len(matches))
	dataBuild := ""
	for _, m := range matches {
		if len(m) < 2 || strings.TrimSpace(m[1]) == "" {
			continue
		}
		src := m[1]
		sources = append(sources, src)
		if dataBuild == "" {
			dataBuild = chatGPTWebCBuildRE.FindString(src)
		}
	}
	if dataBuild == "" {
		if m := chatGPTWebDataBuildRE.FindStringSubmatch(html); len(m) > 1 {
			dataBuild = m[1]
		}
	}
	if len(sources) == 0 {
		sources = []string{chatGPTWebDefaultPowScript}
	}
	return sources, dataBuild
}

func buildChatGPTWebLegacyRequirementsToken(userAgent string, scriptSources []string, dataBuild string) string {
	seed := fmt.Sprintf("%f", rand.Float64())
	config := buildChatGPTWebPowConfig(userAgent, scriptSources, dataBuild)
	answer, _ := chatGPTWebPowGenerate(seed, "0fffff", config, 500000)
	return "gAAAAAC" + answer
}

func buildChatGPTWebProofToken(seed, difficulty, userAgent string, scriptSources []string, dataBuild string) (string, error) {
	config := buildChatGPTWebPowConfig(userAgent, scriptSources, dataBuild)
	answer, solved := chatGPTWebPowGenerate(seed, difficulty, config, 500000)
	if !solved {
		return "", fmt.Errorf("failed to solve proof token: difficulty=%s", difficulty)
	}
	return "gAAAAAB" + answer, nil
}

func buildChatGPTWebPowConfig(userAgent string, scriptSources []string, dataBuild string) []any {
	scriptSource := chatGPTWebDefaultPowScript
	if len(scriptSources) > 0 {
		scriptSource = scriptSources[rand.Intn(len(scriptSources))]
	}
	cores := []int{8, 16, 24, 32}
	documentKeys := []string{"_reactListeningo743lnnpvdg", "location"}
	return []any{
		[]int{3000, 4000, 5000}[rand.Intn(3)],
		time.Now().In(time.FixedZone("EST", -5*3600)).Format("Mon Jan 02 2006 15:04:05") + " GMT-0500 (Eastern Standard Time)",
		4294705152,
		0,
		userAgent,
		scriptSource,
		dataBuild,
		"en-US",
		"en-US,es-US,en,es",
		0,
		"webdriver-false",
		documentKeys[rand.Intn(len(documentKeys))],
		"window",
		float64(time.Now().UnixNano()) / 1e6,
		uuid.NewString(),
		"",
		cores[rand.Intn(len(cores))],
		float64(time.Now().UnixMicro()) / 1e3,
	}
}

func chatGPTWebPowGenerate(seed, difficulty string, config []any, limit int) (string, bool) {
	target, err := hex.DecodeString(difficulty)
	if err != nil || len(target) == 0 {
		return "", false
	}
	diffLen := len(difficulty) / 2
	seedBytes := []byte(seed)
	part1Bytes, _ := json.Marshal(config[:3])
	part2Bytes, _ := json.Marshal(config[4:9])
	part3Bytes, _ := json.Marshal(config[10:])
	part1 := strings.TrimSuffix(string(part1Bytes), "]") + ","
	part2 := "," + strings.TrimPrefix(strings.TrimSuffix(string(part2Bytes), "]"), "[") + ","
	part3 := "," + strings.TrimPrefix(string(part3Bytes), "[")
	for i := 0; i < limit; i++ {
		finalJSON := part1 + fmt.Sprintf("%d", i) + part2 + fmt.Sprintf("%d", i>>1) + part3
		encoded := []byte(base64.StdEncoding.EncodeToString([]byte(finalJSON)))
		digest := sha3.Sum512(append(seedBytes, encoded...))
		if bytes.Compare(digest[:diffLen], target) <= 0 {
			return string(encoded), true
		}
	}
	fallback := "wQ8Lk5FbGpA2NcR9dShT6gYjU7VxZ4D" + base64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%q", seed)))
	return fallback, false
}

// SSE delta parser

func scanChatGPTWebConversationDeltas(r io.Reader, onDelta func(string) error) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 8<<20)
	var dataLines []string
	current := ""
	flush := func() error {
		if len(dataLines) == 0 {
			return nil
		}
		payload := strings.TrimSpace(strings.Join(dataLines, "\n"))
		dataLines = dataLines[:0]
		if payload == "" || payload == "[DONE]" {
			return nil
		}
		text := chatGPTWebAssistantText(payload, current)
		if text == "" || text == current {
			return nil
		}
		delta := strings.TrimPrefix(text, current)
		if delta == text && current != "" {
			delta = text
		}
		current = text
		if delta != "" && onDelta != nil {
			return onDelta(delta)
		}
		return nil
	}
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			if err := flush(); err != nil {
				return err
			}
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}
	if err := flush(); err != nil {
		return err
	}
	return scanner.Err()
}

func chatGPTWebAssistantText(payload string, current string) string {
	var event map[string]any
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		return ""
	}
	for _, candidate := range []any{event, event["v"]} {
		m, ok := candidate.(map[string]any)
		if !ok {
			continue
		}
		message, _ := m["message"].(map[string]any)
		author, _ := message["author"].(map[string]any)
		if strings.TrimSpace(fmt.Sprint(author["role"])) != "assistant" {
			continue
		}
		content, _ := message["content"].(map[string]any)
		parts, _ := content["parts"].([]any)
		var b strings.Builder
		for _, part := range parts {
			if s, ok := part.(string); ok {
				_, _ = b.WriteString(s)
			}
		}
		if b.Len() > 0 {
			return b.String()
		}
	}
	if op, _ := event["o"].(string); op == "patch" {
		if path, _ := event["p"].(string); path == "/message/content/parts/0" {
			if ops, ok := event["v"].([]any); ok {
				text := current
				for _, item := range ops {
					m, _ := item.(map[string]any)
					if m["p"] == float64(0) || m["p"] == "" {
						if v, ok := m["v"].(string); ok {
							text += v
						}
					}
				}
				return text
			}
		}
	}
	if v, ok := event["v"].(string); ok && current != "" {
		return current + v
	}
	return ""
}

func stringMapVal(m map[string]any, key string) string {
	if m == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(m[key]))
}

func boolMapVal(m map[string]any, key string) bool {
	if m == nil {
		return false
	}
	v, ok := m[key].(bool)
	return ok && v
}

func estimateChatGPTWebTokens(text string) int {
	text = strings.TrimSpace(text)
	if text == "" {
		return 0
	}
	return len([]rune(text))/4 + 1
}
