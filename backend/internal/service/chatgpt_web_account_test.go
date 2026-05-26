package service

import "testing"

func TestAccountOpenAIBackendMode(t *testing.T) {
	tests := []struct {
		name      string
		account   *Account
		wantMode  OpenAIBackendMode
		wantWeb   bool
	}{
		{
			name:     "non-openai platform returns any",
			account:  &Account{Platform: PlatformAnthropic, Type: AccountTypeOAuth},
			wantMode: OpenAIBackendModeAny,
		},
		{
			name:     "openai oauth with no extra defaults to codex",
			account:  &Account{Platform: PlatformOpenAI, Type: AccountTypeOAuth},
			wantMode: OpenAIBackendModeCodex,
		},
		{
			name: "openai oauth with codex mode",
			account: &Account{
				Platform: PlatformOpenAI, Type: AccountTypeOAuth,
				Extra: map[string]any{"openai_backend_mode": "codex"},
			},
			wantMode: OpenAIBackendModeCodex,
		},
		{
			name: "openai oauth with chatgpt_web mode",
			account: &Account{
				Platform: PlatformOpenAI, Type: AccountTypeOAuth,
				Extra: map[string]any{"openai_backend_mode": "chatgpt_web"},
			},
			wantMode: OpenAIBackendModeChatGPTWeb,
			wantWeb:  true,
		},
		{
			name: "openai apikey with chatgpt_web is not web mode (needs oauth)",
			account: &Account{
				Platform: PlatformOpenAI, Type: AccountTypeAPIKey,
				Extra: map[string]any{"openai_backend_mode": "chatgpt_web"},
			},
			wantMode: OpenAIBackendModeChatGPTWeb,
			wantWeb:  false, // IsOpenAIChatGPTWebMode requires OAuth
		},
		{
			name: "backward compat: mode in credentials",
			account: &Account{
				Platform: PlatformOpenAI, Type: AccountTypeOAuth,
				Credentials: map[string]any{"openai_backend_mode": "chatgpt_web"},
			},
			wantMode: OpenAIBackendModeChatGPTWeb,
			wantWeb:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.account.OpenAIBackendMode()
			if got != tt.wantMode {
				t.Errorf("OpenAIBackendMode() = %q, want %q", got, tt.wantMode)
			}
			if gotWeb := tt.account.IsOpenAIChatGPTWebMode(); gotWeb != tt.wantWeb {
				t.Errorf("IsOpenAIChatGPTWebMode() = %v, want %v", gotWeb, tt.wantWeb)
			}
		})
	}
}

func TestAccountMatchesOpenAIBackendMode(t *testing.T) {
	codexAccount := &Account{
		Platform: PlatformOpenAI, Type: AccountTypeOAuth,
		Extra: map[string]any{"openai_backend_mode": "codex"},
	}
	webAccount := &Account{
		Platform: PlatformOpenAI, Type: AccountTypeOAuth,
		Extra: map[string]any{"openai_backend_mode": "chatgpt_web"},
	}

	if !codexAccount.MatchesOpenAIBackendMode(OpenAIBackendModeCodex) {
		t.Error("codex account should match codex mode")
	}
	if codexAccount.MatchesOpenAIBackendMode(OpenAIBackendModeChatGPTWeb) {
		t.Error("codex account should not match chatgpt_web mode")
	}
	if !codexAccount.MatchesOpenAIBackendMode(OpenAIBackendModeAny) {
		t.Error("codex account should match any mode")
	}

	if !webAccount.MatchesOpenAIBackendMode(OpenAIBackendModeChatGPTWeb) {
		t.Error("web account should match chatgpt_web mode")
	}
	if webAccount.MatchesOpenAIBackendMode(OpenAIBackendModeCodex) {
		t.Error("web account should not match codex mode")
	}
	if !webAccount.MatchesOpenAIBackendMode(OpenAIBackendModeAny) {
		t.Error("web account should match any mode")
	}
}

func TestNormalizeOpenAIBackendMode(t *testing.T) {
	tests := []struct {
		input string
		want  OpenAIBackendMode
	}{
		{"", OpenAIBackendModeCodex},
		{"codex", OpenAIBackendModeCodex},
		{"chatgpt_web", OpenAIBackendModeChatGPTWeb},
		{"CHATGPT_WEB", OpenAIBackendModeChatGPTWeb},
		{" ChatGPT_Web ", OpenAIBackendModeChatGPTWeb},
		{"unknown", OpenAIBackendModeCodex},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeOpenAIBackendMode(tt.input)
			if got != tt.want {
				t.Errorf("normalizeOpenAIBackendMode(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
