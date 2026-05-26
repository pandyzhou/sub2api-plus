package service

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/Wei-Shaw/sub2api/internal/pkg/apicompat"
)

func TestParseChatGPTWebPowResources(t *testing.T) {
	tests := []struct {
		name        string
		html        string
		wantSources int
		wantBuild   string
	}{
		{
			name:        "empty html returns default script",
			html:        "<html></html>",
			wantSources: 1,
			wantBuild:   "",
		},
		{
			name:        "extracts script sources and data-build from src",
			html:        `<html><head><script src="/c/abc123/_next/static/script1.js"></script><script src="/other.js"></script></head></html>`,
			wantSources: 2,
			wantBuild:   "c/abc123/_",
		},
		{
			name:        "extracts data-build from html attribute",
			html:        `<html data-build="v1.2.3"><script src="/other.js"></script></html>`,
			wantSources: 1,
			wantBuild:   "v1.2.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sources, build := parseChatGPTWebPowResources(tt.html)
			if len(sources) != tt.wantSources {
				t.Errorf("sources count = %d, want %d", len(sources), tt.wantSources)
			}
			if build != tt.wantBuild {
				t.Errorf("dataBuild = %q, want %q", build, tt.wantBuild)
			}
		})
	}
}

func TestBuildChatGPTWebLegacyRequirementsToken(t *testing.T) {
	token := buildChatGPTWebLegacyRequirementsToken("Mozilla/5.0 Test", nil, "")
	if !strings.HasPrefix(token, "gAAAAAC") {
		t.Errorf("legacy token should start with gAAAAAC, got: %s", token)
	}
	if len(token) < 10 {
		t.Errorf("legacy token too short: %d chars", len(token))
	}
}

func TestBuildChatGPTWebProofToken(t *testing.T) {
	token, err := buildChatGPTWebProofToken("0.123456", "0fffff", "Mozilla/5.0 Test", nil, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.HasPrefix(token, "gAAAAAB") {
		t.Errorf("proof token should start with gAAAAAB, got: %s", token)
	}
}

func TestBuildChatGPTWebPowConfig(t *testing.T) {
	config := buildChatGPTWebPowConfig("Mozilla/5.0 Test", nil, "")
	if len(config) != 18 {
		t.Fatalf("config length = %d, want 18", len(config))
	}
	// Verify user agent is at index 4
	if ua, ok := config[4].(string); !ok || ua != "Mozilla/5.0 Test" {
		t.Errorf("config[4] = %v, want 'Mozilla/5.0 Test'", config[4])
	}
}

func TestChatGPTWebPowGenerate(t *testing.T) {
	config := buildChatGPTWebPowConfig("Mozilla/5.0 Test", nil, "")
	answer, solved := chatGPTWebPowGenerate("0.123456", "0fffff", config, 500000)
	if !solved {
		t.Error("expected PoW to be solved for easy difficulty 0fffff")
	}
	if answer == "" {
		t.Error("expected non-empty answer")
	}
}

func TestChatGPTWebConversationMessages(t *testing.T) {
	messages := []apicompat.ChatMessage{
		{Role: "system", Content: json.RawMessage(`"You are helpful"`)},
		{Role: "user", Content: json.RawMessage(`"Hello"`)},
		{Role: "assistant", Content: json.RawMessage(`"Hi there"`)},
	}
	result := chatGPTWebConversationMessages(messages)
	if len(result) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(result))
	}
	for i, msg := range result {
		if _, ok := msg["id"]; !ok {
			t.Errorf("message %d missing id", i)
		}
		author, _ := msg["author"].(map[string]any)
		content, _ := msg["content"].(map[string]any)
		if author == nil || content == nil {
			t.Errorf("message %d missing author or content", i)
		}
	}
}

func TestChatGPTWebExtractText(t *testing.T) {
	tests := []struct {
		name string
		raw  json.RawMessage
		want string
	}{
		{"null", json.RawMessage("null"), ""},
		{"string", json.RawMessage(`"hello world"`), "hello world"},
		{"empty string", json.RawMessage(`""`), ""},
		{"text parts", json.RawMessage(`[{"type":"text","text":"part1"},{"type":"text","text":"part2"}]`), "part1part2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := chatGPTWebExtractText(tt.raw)
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestScanChatGPTWebConversationDeltas(t *testing.T) {
	// Simulate ChatGPT Web SSE with a complete assistant message
	events := "data: {\"message\":{\"author\":{\"role\":\"assistant\"},\"content\":{\"content_type\":\"text\",\"parts\":[\"Hello world\"]}}}\n\n"
	r := strings.NewReader(events)

	var deltas []string
	err := scanChatGPTWebConversationDeltas(r, func(delta string) error {
		deltas = append(deltas, delta)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deltas) == 0 {
		t.Fatal("expected at least one delta")
	}
	if !strings.Contains(strings.Join(deltas, ""), "Hello world") {
		t.Errorf("expected 'Hello world' in deltas, got: %v", deltas)
	}
}

func TestScanChatGPTWebConversationDeltas_IgnoresDone(t *testing.T) {
	events := "data: [DONE]\n\n"
	r := strings.NewReader(events)

	var deltas []string
	err := scanChatGPTWebConversationDeltas(r, func(delta string) error {
		deltas = append(deltas, delta)
		return nil
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(deltas) != 0 {
		t.Errorf("expected no deltas for [DONE], got: %v", deltas)
	}
}

func TestEstimateChatGPTWebTokens(t *testing.T) {
	tests := []struct {
		text string
		min  int
	}{
		{"", 0},
		{"hello", 2},
		{"这是一个测试文本用于估算 token 数量", 4},
	}
	for _, tt := range tests {
		got := estimateChatGPTWebTokens(tt.text)
		if got < tt.min {
			t.Errorf("estimateChatGPTWebTokens(%q) = %d, want >= %d", tt.text, got, tt.min)
		}
	}
}

func TestStringMapVal(t *testing.T) {
	m := map[string]any{"key": " value ", "num": 42, "nil": nil}
	if got := stringMapVal(m, "key"); got != "value" {
		t.Errorf("stringMapVal(key) = %q, want 'value'", got)
	}
	if got := stringMapVal(m, "missing"); got != "" {
		t.Errorf("stringMapVal(missing) = %q, want ''", got)
	}
	if got := stringMapVal(nil, "key"); got != "" {
		t.Errorf("stringMapVal(nil) = %q, want ''", got)
	}
}

func TestBoolMapVal(t *testing.T) {
	m := map[string]any{"yes": true, "no": false, "str": "true"}
	if !boolMapVal(m, "yes") {
		t.Error("expected true for 'yes'")
	}
	if boolMapVal(m, "no") {
		t.Error("expected false for 'no'")
	}
	if boolMapVal(m, "str") {
		t.Error("expected false for string value")
	}
	if boolMapVal(nil, "key") {
		t.Error("expected false for nil map")
	}
}
