package admin

import (
	"testing"
	"time"
)

func TestChatGPTRegisterEventTokenGenerationValidationAndExpiry(t *testing.T) {
	token, expiresAt := newChatGPTRegisterEventToken(5 * time.Minute)
	if token == "" {
		t.Fatalf("token is empty")
	}
	if !validateChatGPTRegisterEventToken(token, time.Now()) {
		t.Fatalf("fresh token should validate")
	}
	if validateChatGPTRegisterEventToken(token, expiresAt.Add(time.Second)) {
		t.Fatalf("expired token should not validate")
	}
	if validateChatGPTRegisterEventToken("missing", time.Now()) {
		t.Fatalf("unknown token should not validate")
	}
}
