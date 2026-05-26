package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (s *OpenAIGatewayService) forwardAsChatGPTWebChatCompletions(
	ctx context.Context,
	c *gin.Context,
	account *Account,
	body []byte,
	defaultMappedModel string,
) (*OpenAIForwardResult, error) {
	startTime := time.Now()

	var chatReq apicompat.ChatCompletionsRequest
	if err := json.Unmarshal(body, &chatReq); err != nil {
		return nil, fmt.Errorf("parse chat completions request: %w", err)
	}

	originalModel := chatReq.Model
	billingModel := resolveOpenAIForwardModel(account, originalModel, defaultMappedModel)
	upstreamModel := billingModel

	token, _, err := s.GetAccessToken(ctx, account)
	if err != nil {
		return nil, fmt.Errorf("get access token: %w", err)
	}

	webClient := newChatGPTWebClient(s, account, token)

	clientStream := chatReq.Stream
	completionID := "chatcmpl-" + uuid.NewString()
	created := time.Now().Unix()
	responseModel := originalModel
	if responseModel == "" {
		responseModel = upstreamModel
	}

	if clientStream {
		return s.handleChatGPTWebStreamingResponse(ctx, c, webClient, chatReq.Messages, upstreamModel, responseModel, billingModel, completionID, created, startTime)
	}

	return s.handleChatGPTWebBufferedResponse(ctx, c, webClient, chatReq.Messages, upstreamModel, responseModel, billingModel, completionID, created, startTime)
}

func (s *OpenAIGatewayService) handleChatGPTWebStreamingResponse(
	ctx context.Context,
	c *gin.Context,
	webClient *ChatGPTWebClient,
	messages []apicompat.ChatMessage,
	upstreamModel string,
	responseModel string,
	billingModel string,
	completionID string,
	created int64,
	startTime time.Time,
) (*OpenAIForwardResult, error) {
	writeChatSSEHeaders(c)
	startTimeToFirstToken := time.Now()
	var firstTokenMs *int

	sentRole := false
	var fullContent strings.Builder

	err := webClient.StreamConversation(ctx, messages, upstreamModel, func(delta string) error {
		if firstTokenMs == nil {
			ms := int(time.Since(startTimeToFirstToken).Milliseconds())
			firstTokenMs = &ms
		}

		chunk := apicompat.ChatCompletionsChunk{
			ID:      completionID,
			Object:  "chat.completion.chunk",
			Created: created,
			Model:   responseModel,
			Choices: []apicompat.ChatChunkChoice{{
				Index: 0,
				Delta: apicompat.ChatDelta{
					Content: &delta,
				},
			}},
		}
		if !sentRole {
			sentRole = true
			role := "assistant"
			chunk.Choices[0].Delta.Role = role
		}
		_, _ = fullContent.WriteString(delta)

		sseData, err := apicompat.ChatChunkToSSE(chunk)
		if err != nil {
			return err
		}
		return writeChatSSEEvent(c, sseData)
	})
	if err != nil {
		return nil, err
	}

	if !sentRole {
		empty := ""
		chunk := apicompat.ChatCompletionsChunk{
			ID:      completionID,
			Object:  "chat.completion.chunk",
			Created: created,
			Model:   responseModel,
			Choices: []apicompat.ChatChunkChoice{{
				Index: 0,
				Delta: apicompat.ChatDelta{Role: "assistant", Content: &empty},
			}},
		}
		sseData, _ := apicompat.ChatChunkToSSE(chunk)
		_ = writeChatSSEEvent(c, sseData)
	}

	stopReason := "stop"
	finChunk := apicompat.ChatCompletionsChunk{
		ID:      completionID,
		Object:  "chat.completion.chunk",
		Created: created,
		Model:   responseModel,
		Choices: []apicompat.ChatChunkChoice{{
			Index:        0,
			Delta:        apicompat.ChatDelta{},
			FinishReason: &stopReason,
		}},
	}
	sseData, _ := apicompat.ChatChunkToSSE(finChunk)
	_ = writeChatSSEEvent(c, sseData)
	_ = writeChatSSEEvent(c, "[DONE]")

	inputTokens := 0
	for _, m := range messages {
		inputTokens += estimateChatGPTWebTokens(chatGPTWebExtractText(m.Content))
	}
	outputTokens := estimateChatGPTWebTokens(fullContent.String())

	result := &OpenAIForwardResult{
		RequestID:     completionID,
		Usage:         OpenAIUsage{InputTokens: inputTokens, OutputTokens: outputTokens},
		Model:         responseModel,
		BillingModel:  billingModel,
		UpstreamModel: upstreamModel,
		Stream:        true,
		Duration:      time.Since(startTime),
		FirstTokenMs:  firstTokenMs,
	}
	return result, nil
}

func (s *OpenAIGatewayService) handleChatGPTWebBufferedResponse(
	ctx context.Context,
	c *gin.Context,
	webClient *ChatGPTWebClient,
	messages []apicompat.ChatMessage,
	upstreamModel string,
	responseModel string,
	billingModel string,
	completionID string,
	created int64,
	startTime time.Time,
) (*OpenAIForwardResult, error) {
	var fullContent strings.Builder
	err := webClient.StreamConversation(ctx, messages, upstreamModel, func(delta string) error {
		_, _ = fullContent.WriteString(delta)
		return nil
	})
	if err != nil {
		return nil, err
	}

	content := fullContent.String()
	inputTokens := 0
	for _, m := range messages {
		inputTokens += estimateChatGPTWebTokens(chatGPTWebExtractText(m.Content))
	}
	outputTokens := estimateChatGPTWebTokens(content)

	resp := apicompat.ChatCompletionsResponse{
		ID:      completionID,
		Object:  "chat.completion",
		Created: created,
		Model:   responseModel,
		Choices: []apicompat.ChatChoice{{
			Index: 0,
			Message: apicompat.ChatMessage{
				Role:    "assistant",
				Content: json.RawMessage(chatGPTWebJSONString(content)),
			},
			FinishReason: "stop",
		}},
		Usage: &apicompat.ChatUsage{
			PromptTokens:     inputTokens,
			CompletionTokens: outputTokens,
			TotalTokens:      inputTokens + outputTokens,
		},
	}
	c.JSON(200, resp)

	result := &OpenAIForwardResult{
		RequestID:     completionID,
		Usage:         OpenAIUsage{InputTokens: inputTokens, OutputTokens: outputTokens},
		Model:         responseModel,
		BillingModel:  billingModel,
		UpstreamModel: upstreamModel,
		Stream:        false,
		Duration:      time.Since(startTime),
	}
	return result, nil
}

func writeChatSSEHeaders(c *gin.Context) {
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Status(200)
}

func writeChatSSEEvent(c *gin.Context, data string) error {
	if _, err := c.Writer.Write([]byte("data: " + data + "\n\n")); err != nil {
		return err
	}
	c.Writer.Flush()
	return nil
}

func chatGPTWebJSONString(s string) string {
	b, err := json.Marshal(s)
	if err != nil {
		return `""`
	}
	return string(b)
}
