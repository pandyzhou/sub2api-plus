package service

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ChatGPTImageClient implements pure Go ChatGPT image generation
// Matches chatgpt2api's exact request flow without Python dependency
type ChatGPTImageClient struct {
	accessToken  string
	refreshToken string
	proxyURL     string
	client       *http.Client
	deviceID     string
	sessionID    string
	userAgent    string
}

// NewChatGPTImageClient creates a new image generation client
func NewChatGPTImageClient(accessToken, refreshToken, proxyURL string) *ChatGPTImageClient {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
	}

	if proxyURL != "" {
		proxyURLParsed, err := url.Parse(proxyURL)
		if err == nil {
			transport.Proxy = http.ProxyURL(proxyURLParsed)
		}
	}

	client := &http.Client{
		Transport: transport,
		Timeout:   300 * time.Second,
	}

	return &ChatGPTImageClient{
		accessToken:  accessToken,
		refreshToken: refreshToken,
		proxyURL:     proxyURL,
		client:       client,
		deviceID:     uuid.New().String(),
		sessionID:    uuid.New().String(),
		userAgent:    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/143.0.0.0 Safari/537.36 Edg/143.0.0.0",
	}
}

// buildHeaders creates standard headers for ChatGPT requests
func (c *ChatGPTImageClient) buildHeaders() map[string]string {
	return map[string]string{
		"User-Agent":                  c.userAgent,
		"Origin":                      "https://chatgpt.com",
		"Referer":                     "https://chatgpt.com/",
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
		"OAI-Client-Version":          "prod-be885abbfcfe7b1f511e88b3003d9ee44757fbad",
		"OAI-Client-Build-Number":     "5955942",
		"Authorization":               fmt.Sprintf("Bearer %s", c.accessToken),
	}
}

// getChatRequirements gets the final requirements token
func (c *ChatGPTImageClient) getChatRequirements(ctx context.Context) (string, string, error) {
	reqToken := BuildLegacyRequirementsToken(c.userAgent)

	payload := map[string]any{
		"p": reqToken,
	}

	payloadBytes, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://chatgpt.com/backend-api/sentinel/chat-requirements", bytes.NewReader(payloadBytes))
	if err != nil {
		return "", "", err
	}

	headers := c.buildHeaders()
	headers["X-OpenAI-Target-Path"] = "/backend-api/sentinel/chat-requirements"
	headers["X-OpenAI-Target-Route"] = "/backend-api/sentinel/chat-requirements"
	headers["Content-Type"] = "application/json"
	headers["Accept"] = "application/json"

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", "", fmt.Errorf("chat-requirements failed: %d %s", resp.StatusCode, string(body))
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", "", err
	}

	token := ""
	proofToken := ""
	if t, ok := result["token"].(string); ok {
		token = t
	}
	if p, ok := result["proof_token"].(string); ok {
		proofToken = p
	}

	return token, proofToken, nil
}

// getConduitToken prepares the conversation
func (c *ChatGPTImageClient) getConduitToken(ctx context.Context, prompt, model, sentinelToken, proofToken string) (string, error) {
	payload := map[string]any{
		"action":                "next",
		"fork_from_shared_post": false,
		"parent_message_id":     uuid.New().String(),
		"model":                 model,
		"client_prepare_state":  "success",
		"timezone_offset_min":   -480,
		"timezone":              "Asia/Shanghai",
		"conversation_mode":     map[string]string{"kind": "primary_assistant"},
		"system_hints":          []string{"picture_v2"},
		"partial_query": map[string]any{
			"id":      uuid.New().String(),
			"author":  map[string]string{"role": "user"},
			"content": map[string]any{"content_type": "text", "parts": []string{prompt}},
		},
		"supports_buffering":     true,
		"supported_encodings":    []string{"v1"},
		"client_contextual_info": map[string]string{"app_name": "chatgpt.com"},
	}

	payloadBytes, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://chatgpt.com/backend-api/f/conversation/prepare", bytes.NewReader(payloadBytes))
	if err != nil {
		return "", err
	}

	headers := c.buildHeaders()
	headers["X-OpenAI-Target-Path"] = "/backend-api/f/conversation/prepare"
	headers["X-OpenAI-Target-Route"] = "/backend-api/f/conversation/prepare"
	headers["Content-Type"] = "application/json"
	headers["Accept"] = "application/json"
	headers["OpenAI-Sentinel-Chat-Requirements-Token"] = sentinelToken
	if proofToken != "" {
		headers["OpenAI-Sentinel-Proof-Token"] = proofToken
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("conduit failed: %d %s", resp.StatusCode, string(body))
	}

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	conduitToken := ""
	if t, ok := result["conduit_token"].(string); ok {
		conduitToken = t
	}

	return conduitToken, nil
}

// generateImage starts the SSE conversation and extracts the image file ID
func (c *ChatGPTImageClient) generateImage(ctx context.Context, prompt, model, sentinelToken, proofToken, conduitToken string) (string, error) {
	payload := map[string]any{
		"action": "next",
		"messages": []map[string]any{
			{
				"id":          uuid.New().String(),
				"author":      map[string]string{"role": "user"},
				"create_time": float64(time.Now().Unix()),
				"content":     map[string]any{"content_type": "text", "parts": []string{prompt}},
				"metadata": map[string]any{
					"developer_mode_connector_ids": []string{},
					"selected_github_repos":        []string{},
					"selected_all_github_repos":    false,
					"system_hints":                 []string{"picture_v2"},
					"serialization_metadata":       map[string]any{"custom_symbol_offsets": []any{}},
				},
			},
		},
		"parent_message_id":        uuid.New().String(),
		"model":                    model,
		"client_prepare_state":     "sent",
		"timezone_offset_min":      -480,
		"timezone":                 "Asia/Shanghai",
		"conversation_mode":        map[string]string{"kind": "primary_assistant"},
		"enable_message_followups": true,
		"system_hints":             []string{"picture_v2"},
		"supports_buffering":       true,
		"supported_encodings":      []string{"v1"},
		"client_contextual_info": map[string]any{
			"is_dark_mode":      false,
			"time_since_loaded": 1200,
			"page_height":       1072,
			"page_width":        1724,
			"pixel_ratio":       1.2,
			"screen_height":     1440,
			"screen_width":      2560,
			"app_name":          "chatgpt.com",
		},
		"paragen_cot_summary_display_override": "allow",
		"force_parallel_switch":                "auto",
	}

	payloadBytes, _ := json.Marshal(payload)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://chatgpt.com/backend-api/f/conversation", bytes.NewReader(payloadBytes))
	if err != nil {
		return "", err
	}

	headers := c.buildHeaders()
	headers["X-OpenAI-Target-Path"] = "/backend-api/f/conversation"
	headers["X-OpenAI-Target-Route"] = "/backend-api/f/conversation"
	headers["Content-Type"] = "application/json"
	headers["Accept"] = "text/event-stream"
	headers["OpenAI-Sentinel-Chat-Requirements-Token"] = sentinelToken
	headers["X-Oai-Turn-Trace-Id"] = uuid.New().String()
	if proofToken != "" {
		headers["OpenAI-Sentinel-Proof-Token"] = proofToken
	}
	if conduitToken != "" {
		headers["X-Conduit-Token"] = conduitToken
	}

	for k, v := range headers {
		req.Header.Set(k, v)
	}

	resp, err := c.client.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("conversation failed: %d %s", resp.StatusCode, string(body))
	}

	// Parse SSE stream
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var event map[string]any
		if err := json.Unmarshal([]byte(data), &event); err != nil {
			continue
		}

		// Extract file ID from image_asset_pointer
		if msg, ok := event["message"].(map[string]any); ok {
			if content, ok := msg["content"].(map[string]any); ok {
				if parts, ok := content["parts"].([]any); ok {
					for _, part := range parts {
						if partMap, ok := part.(map[string]any); ok {
							if partMap["content_type"] == "image_asset_pointer" {
								if assetPointer, ok := partMap["asset_pointer"].(string); ok {
									fileID := strings.TrimPrefix(assetPointer, "file-service://")
									if fileID != "" {
										return fileID, nil
									}
								}
							}
						}
					}
				}
			}
		}
	}

	return "", fmt.Errorf("no image file ID found in response")
}

// GenerateImage is the main entry point for image generation
func (c *ChatGPTImageClient) GenerateImage(ctx context.Context, prompt, model string) (string, error) {
	// 1. Get chat requirements token
	sentinelToken, proofToken, err := c.getChatRequirements(ctx)
	if err != nil {
		return "", fmt.Errorf("get chat requirements: %w", err)
	}

	// 2. Get conduit token
	conduitToken, err := c.getConduitToken(ctx, prompt, model, sentinelToken, proofToken)
	if err != nil {
		return "", fmt.Errorf("get conduit token: %w", err)
	}

	// 3. Generate image and get file ID
	fileID, err := c.generateImage(ctx, prompt, model, sentinelToken, proofToken, conduitToken)
	if err != nil {
		return "", fmt.Errorf("generate image: %w", err)
	}

	return fileID, nil
}

// DownloadImage downloads the generated image from ChatGPT file service
func (c *ChatGPTImageClient) DownloadImage(ctx context.Context, fileID string) ([]byte, error) {
	// ChatGPT uses a signed URL format for file downloads
	downloadURL := fmt.Sprintf("https://files.oaiusercontent.com/file-%s", fileID)

	req, err := http.NewRequestWithContext(ctx, "GET", downloadURL, nil)
	if err != nil {
		return nil, err
	}

	// Add minimal headers for file download
	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Referer", "https://chatgpt.com/")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("download failed: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}

// DownloadImageFromURL downloads an image from a direct URL
func (c *ChatGPTImageClient) DownloadImageFromURL(ctx context.Context, imageURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", imageURL, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", c.userAgent)
	req.Header.Set("Authorization", "Bearer "+c.accessToken)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("download from URL failed: %d", resp.StatusCode)
	}

	return io.ReadAll(resp.Body)
}
