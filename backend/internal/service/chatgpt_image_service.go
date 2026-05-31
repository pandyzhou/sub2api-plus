package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"os/exec"
	"time"
)

// ChatGPTImageService uses pure Go implementation for ChatGPT image generation.
// No Python dependency - all requests are made directly from Go.
type ChatGPTImageService struct {
	poolService   *ChatGPTAccountPoolService
	storage       *ChatGPTImageStorage
	proxyURL      string
	tokenProvider *OpenAITokenProvider
}

// SetTokenProvider injects the OpenAI token provider for automatic token refresh.
func (s *ChatGPTImageService) SetTokenProvider(tp *OpenAITokenProvider) {
	s.tokenProvider = tp
}

type ChatGPTImageGenerateInput struct {
	Prompt string `json:"prompt"`
	Model  string `json:"model"`
	N      int    `json:"n"`
	Size   string `json:"size"`
}

type ChatGPTImageGenerateOutput struct {
	Created int64              `json:"created"`
	Data    []ChatGPTImageData `json:"data"`
}

type ChatGPTImageData struct {
	URL           string `json:"url,omitempty"`
	B64JSON       string `json:"b64_json,omitempty"`
	RevisedPrompt string `json:"revised_prompt,omitempty"`
}

func NewChatGPTImageService(poolService *ChatGPTAccountPoolService, storage *ChatGPTImageStorage, proxyURL string) *ChatGPTImageService {
	return &ChatGPTImageService{
		poolService: poolService,
		storage:     storage,
		proxyURL:    proxyURL,
	}
}

func (s *ChatGPTImageService) Generate(ctx context.Context, input ChatGPTImageGenerateInput) (*ChatGPTImageGenerateOutput, error) {
	if input.Prompt == "" {
		return nil, fmt.Errorf("prompt is required")
	}
	model := input.Model
	if model == "" {
		model = "gpt-image-2"
	}

	// 1. 获取可用账号
	acc, rawToken, release, err := s.poolService.GetAvailableImageAccount()
	if err != nil {
		return nil, fmt.Errorf("没有可用的图片生成账号: %w", err)
	}
	defer release()

	// 2. 使用 token provider 获取刷新后的 token（如果可用）
	accessToken := rawToken
	refreshToken := ""
	if s.tokenProvider != nil && acc != nil {
		if refreshedToken, err := s.tokenProvider.GetAccessToken(ctx, acc); err == nil && refreshedToken != "" {
			accessToken = refreshedToken
		} else if err != nil {
			slog.Warn("chatgpt_image: failed to refresh token", "error", err)
		}
		// Get refresh token if available
		if acc.Credentials != nil {
			if rt, ok := acc.Credentials["refresh_token"].(string); ok {
				refreshToken = rt
			}
		}
	}

	slog.Info("chatgpt_image_generate_start", "prompt", input.Prompt, "model", model, "has_access_token", accessToken != "", "has_refresh_token", refreshToken != "")

	// 3. 调用 Python 脚本生成图片（使用 curl_cffi 绕过 Cloudflare）
	scriptPath := "/app/chatgpt_image_proxy.py"
	result, err := callPythonImageScript(ctx, scriptPath, accessToken, refreshToken, input.Prompt, model, s.proxyURL)
	if err != nil {
		slog.Error("chatgpt_image: python script failed", "error", err)
		s.poolService.MarkImageResult(rawToken, false)
		return nil, fmt.Errorf("generate image: %w", err)
	}

	// 4. 解析结果
	slog.Info("chatgpt_image: python result", "success", result.Success, "has_url", result.ImageURL != "", "has_b64", result.ImageB64 != "", "error", result.Error)
	if !result.Success {
		s.poolService.MarkImageResult(rawToken, false)
		return nil, fmt.Errorf("image generation failed: %s", result.Error)
	}

	// 5. 下载图片（如果返回的是 URL）
	var imgBytes []byte
	if result.ImageURL != "" {
		slog.Info("chatgpt_image: downloading from URL", "url", result.ImageURL)
		// 使用 Python 客户端下载（保持一致的 TLS 指纹）
		client := NewChatGPTImageClient(accessToken, refreshToken, s.proxyURL)
		imgBytes, err = client.DownloadImageFromURL(ctx, result.ImageURL)
		if err != nil {
			slog.Error("chatgpt_image: download failed", "error", err)
			s.poolService.MarkImageResult(rawToken, false)
			return nil, fmt.Errorf("download image failed: %w", err)
		}
		slog.Info("chatgpt_image: download success", "size", len(imgBytes))
	} else if result.ImageB64 != "" {
		slog.Info("chatgpt_image: decoding base64", "length", len(result.ImageB64))
		// 直接使用 base64 数据
		imgBytes, err = base64.StdEncoding.DecodeString(result.ImageB64)
		if err != nil {
			slog.Error("chatgpt_image: base64 decode failed", "error", err)
			s.poolService.MarkImageResult(rawToken, false)
			return nil, fmt.Errorf("decode base64 image failed: %w", err)
		}
		slog.Info("chatgpt_image: base64 decode success", "size", len(imgBytes))
	} else {
		slog.Error("chatgpt_image: no image data in result")
		s.poolService.MarkImageResult(rawToken, false)
		return nil, fmt.Errorf("no image data returned")
	}

	// 6. 保存图片
	var publicURL string
	if s.storage != nil {
		slog.Info("chatgpt_image: saving to storage", "size", len(imgBytes))
		_, publicURL, err = s.storage.Save(imgBytes, "webp")
		if err != nil {
			slog.Warn("chatgpt_image: failed to save image", "error", err)
			// Fallback to base64
			publicURL = ""
		} else {
			slog.Info("chatgpt_image: saved to storage", "url", publicURL)
		}
	} else {
		slog.Warn("chatgpt_image: storage is nil, will use base64")
	}

	// 7. 标记成功
	s.poolService.MarkImageResult(rawToken, true)

	// 8. 构建输出
	output := &ChatGPTImageGenerateOutput{
		Created: time.Now().Unix(),
		Data:    make([]ChatGPTImageData, 0, 1),
	}

	if publicURL != "" {
		slog.Info("chatgpt_image: returning URL", "url", publicURL)
		output.Data = append(output.Data, ChatGPTImageData{
			URL:           publicURL,
			RevisedPrompt: input.Prompt,
		})
	} else {
		// Fallback to base64
		slog.Info("chatgpt_image: returning base64", "size", len(imgBytes))
		b64 := base64.StdEncoding.EncodeToString(imgBytes)
		output.Data = append(output.Data, ChatGPTImageData{
			B64JSON:       b64,
			RevisedPrompt: input.Prompt,
		})
	}

	slog.Info("chatgpt_image_generate_success", "data_count", len(output.Data), "has_url", output.Data[0].URL != "", "has_b64", output.Data[0].B64JSON != "")
	return output, nil
}

// pythonImageResult represents the output from the Python image generation script
type pythonImageResult struct {
	Success  bool   `json:"success"`
	ImageURL string `json:"image_url"`
	ImageB64 string `json:"image_b64"`
	Error    string `json:"error"`
}

// callPythonImageScript calls the Python script to generate an image using curl_cffi
func callPythonImageScript(ctx context.Context, scriptPath, accessToken, refreshToken, prompt, model, proxyURL string) (*pythonImageResult, error) {
	// Prepare input JSON
	input := map[string]string{
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"prompt":        prompt,
		"model":         model,
		"proxy":         proxyURL,
	}
	inputJSON, err := json.Marshal(input)
	if err != nil {
		return nil, fmt.Errorf("marshal input: %w", err)
	}

	// Call Python script
	cmd := exec.CommandContext(ctx, "python3", scriptPath)
	cmd.Stdin = bytes.NewReader(inputJSON)
	
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("python script failed: %w (stderr: %s)", err, stderr.String())
	}

	// Parse output
	var result pythonImageResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		return nil, fmt.Errorf("parse python output: %w (output: %s)", err, stdout.String())
	}

	return &result, nil
}
