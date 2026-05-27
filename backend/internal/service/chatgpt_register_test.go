package service

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

type chatGPTRegisterSettingRepoStub struct {
	values map[string]string
}

func (r *chatGPTRegisterSettingRepoStub) Get(_ context.Context, key string) (*Setting, error) {
	return &Setting{Key: key, Value: r.values[key]}, nil
}

func (r *chatGPTRegisterSettingRepoStub) GetValue(_ context.Context, key string) (string, error) {
	return r.values[key], nil
}

func (r *chatGPTRegisterSettingRepoStub) Set(_ context.Context, key, value string) error {
	if r.values == nil {
		r.values = map[string]string{}
	}
	r.values[key] = value
	return nil
}

func (r *chatGPTRegisterSettingRepoStub) GetMultiple(_ context.Context, keys []string) (map[string]string, error) {
	out := map[string]string{}
	for _, key := range keys {
		out[key] = r.values[key]
	}
	return out, nil
}

func (r *chatGPTRegisterSettingRepoStub) SetMultiple(_ context.Context, settings map[string]string) error {
	if r.values == nil {
		r.values = map[string]string{}
	}
	for key, value := range settings {
		r.values[key] = value
	}
	return nil
}

func (r *chatGPTRegisterSettingRepoStub) GetAll(_ context.Context) (map[string]string, error) {
	out := map[string]string{}
	for key, value := range r.values {
		out[key] = value
	}
	return out, nil
}

func (r *chatGPTRegisterSettingRepoStub) Delete(_ context.Context, key string) error {
	delete(r.values, key)
	return nil
}

func TestChatGPTRegisterService_ReturnsAndPersistsMailConfig(t *testing.T) {
	repo := &chatGPTRegisterSettingRepoStub{values: map[string]string{}}
	svc := NewChatGPTRegisterService(repo, nil)

	result := svc.Update(map[string]any{
		"mail_provider": "mailtm",
		"mail_api_base": "https://mail.example.test",
		"mail_api_key":  "secret-key",
	})

	register, ok := result["register"].(map[string]any)
	if !ok {
		t.Fatalf("register payload missing or wrong type: %#v", result["register"])
	}
	if got := register["mail_provider"]; got != "mailtm" {
		t.Fatalf("mail_provider = %v, want mailtm", got)
	}
	if got := register["mail_api_base"]; got != "https://mail.example.test" {
		t.Fatalf("mail_api_base = %v, want custom base", got)
	}
	if got := register["mail_api_key"]; got != "secret-key" {
		t.Fatalf("mail_api_key = %v, want secret-key", got)
	}

	var stored ChatGPTRegisterConfig
	if err := json.Unmarshal([]byte(repo.values[chatGPTRegisterSettingKey]), &stored); err != nil {
		t.Fatalf("stored config is not json: %v", err)
	}
	if stored.MailProvider != "mailtm" || stored.MailAPIBase != "https://mail.example.test" || stored.MailAPIKey != "secret-key" {
		t.Fatalf("stored mail config mismatch: %#v", stored)
	}
	if len(stored.Mail.Providers) != 1 || stored.Mail.Providers[0].Type != "mailtm" || stored.Mail.Providers[0].APIBase != "https://mail.example.test" || stored.Mail.Providers[0].APIKey != "secret-key" {
		t.Fatalf("stored migrated providers mismatch: %#v", stored.Mail.Providers)
	}
}

func TestChatGPTRegisterService_CreateTempEmailUsesConfiguredMailAPIBase(t *testing.T) {
	var sawAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("Authorization")
		switch r.URL.Path {
		case "/domains":
			_, _ = w.Write([]byte(`{"hydra:member":[{"domain":"example.test"}]}`))
		case "/accounts":
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(`{}`))
		default:
			t.Fatalf("unexpected mail API path: %s", r.URL.Path)
		}
	}))
	defer server.Close()

	svc := NewChatGPTRegisterService(nil, nil)
	mailbox, err := svc.createTempEmail(ChatGPTRegisterConfig{
		MailProvider: "mailtm",
		MailAPIBase:  server.URL,
		MailAPIKey:   "mail-api-key",
	})
	if err != nil {
		t.Fatalf("createTempEmail returned error: %v", err)
	}
	if mailbox.Email == "" || mailbox.Password == "" {
		t.Fatalf("mailbox not populated: %#v", mailbox)
	}
	if sawAuth != "Bearer mail-api-key" {
		t.Fatalf("Authorization header = %q, want bearer API key", sawAuth)
	}
}
