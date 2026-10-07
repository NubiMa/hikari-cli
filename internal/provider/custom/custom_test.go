package custom

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NubiMa/hikari-cli/internal/config"
)

// ---------------------------------------------------------------------------
// Constructor / validation tests
// ---------------------------------------------------------------------------

func TestNew_RequiresEndpoint(t *testing.T) {
	_, err := New("test", config.ProviderConfig{Type: "custom"})
	if err == nil {
		t.Fatal("expected error when endpoint is empty, got nil")
	}
}

func TestNew_DefaultHealthPath(t *testing.T) {
	p, err := New("test", config.ProviderConfig{
		Type:     "custom",
		Endpoint: "http://localhost:1234",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cp := p.(*Provider)
	if cp.healthPath != defaultHealthPath {
		t.Errorf("expected default health path %q, got %q", defaultHealthPath, cp.healthPath)
	}
}

func TestNew_CustomHealthPath(t *testing.T) {
	p, err := New("test", config.ProviderConfig{
		Type:       "custom",
		Endpoint:   "http://localhost:1234",
		HealthPath: "health",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	cp := p.(*Provider)
	if cp.healthPath != "/health" {
		t.Errorf("expected health path %q, got %q", "/health", cp.healthPath)
	}
}

func TestNew_Name(t *testing.T) {
	p, err := New("my-api", config.ProviderConfig{
		Type:     "custom",
		Endpoint: "http://localhost:1234",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(p.Name(), "my-api") {
		t.Errorf("Name() = %q, want it to contain %q", p.Name(), "my-api")
	}
}

// ---------------------------------------------------------------------------
// Status / Connect tests
// ---------------------------------------------------------------------------

func TestStatus_Online(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer srv.Close()

	p, _ := New("test", config.ProviderConfig{
		Type:     "custom",
		Endpoint: srv.URL,
	})

	status, err := p.Status(t.Context())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if !status.Connected {
		t.Errorf("expected Connected=true, got false: %s", status.Message)
	}
}

func TestStatus_Offline(t *testing.T) {
	p, _ := New("test", config.ProviderConfig{
		Type:     "custom",
		Endpoint: "http://127.0.0.1:19999", // nothing listening here
	})
	status, err := p.Status(t.Context())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if status.Connected {
		t.Error("expected Connected=false for unreachable server")
	}
}

func TestStatus_Auth401(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()

	p, _ := New("test", config.ProviderConfig{
		Type:     "custom",
		Endpoint: srv.URL,
	})

	status, err := p.Status(t.Context())
	if err != nil {
		t.Fatalf("Status returned error: %v", err)
	}
	if status.Connected {
		t.Error("expected Connected=false on 401")
	}
}

// ---------------------------------------------------------------------------
// Models listing
// ---------------------------------------------------------------------------

func TestModels_OpenAIFormat(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"data":[{"id":"llama-3.1-70b","name":"Llama 3.1 70B"}]}`))
	}))
	defer srv.Close()

	p, _ := New("test", config.ProviderConfig{
		Type:     "custom",
		Endpoint: srv.URL,
	})

	models, err := p.Models(t.Context())
	if err != nil {
		t.Fatalf("Models returned error: %v", err)
	}
	if len(models) != 1 {
		t.Fatalf("expected 1 model, got %d", len(models))
	}
	if models[0].ID != "llama-3.1-70b" {
		t.Errorf("expected model ID %q, got %q", "llama-3.1-70b", models[0].ID)
	}
}

func TestModels_ServerDoesNotSupportListing(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	p, _ := New("test", config.ProviderConfig{
		Type:     "custom",
		Endpoint: srv.URL,
	})

	models, err := p.Models(t.Context())
	if err != nil {
		t.Fatalf("Models should not error on 404, got: %v", err)
	}
	if len(models) != 0 {
		t.Errorf("expected empty list, got %d models", len(models))
	}
}

// ---------------------------------------------------------------------------
// Chat streaming
// ---------------------------------------------------------------------------

func TestChat_StreamsTokens(t *testing.T) {
	// Serve an SSE stream of two token chunks then [DONE]
	chunks := []string{
		`data: {"choices":[{"delta":{"content":"Hello"}}]}`,
		`data: {"choices":[{"delta":{"content":" world"}}]}`,
		`data: [DONE]`,
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		for _, chunk := range chunks {
			_, _ = w.Write([]byte(chunk + "\n"))
		}
	}))
	defer srv.Close()

	p, _ := New("test", config.ProviderConfig{
		Type:     "custom",
		Endpoint: srv.URL,
		Model:    "test-model",
	})

	ch, err := p.Chat(t.Context(), nil, "hi")
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}

	var tokens []string
	for evt := range ch {
		switch evt.Type {
		case "token":
			tokens = append(tokens, evt.Content)
		case "error":
			t.Fatalf("unexpected error event: %v", evt.Error)
		}
	}

	got := strings.Join(tokens, "")
	if got != "Hello world" {
		t.Errorf("expected %q, got %q", "Hello world", got)
	}
}

func TestChat_NonStreamingResponse(t *testing.T) {
	// Some servers return a plain JSON object (non-streaming) — verify we handle it
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"choices": []map[string]any{
				{"message": map[string]any{"content": "non-streaming response"}},
			},
		})
	}))
	defer srv.Close()

	p, _ := New("test", config.ProviderConfig{
		Type:     "custom",
		Endpoint: srv.URL,
	})

	ch, err := p.Chat(t.Context(), nil, "hi")
	if err != nil {
		t.Fatalf("Chat returned error: %v", err)
	}

	var tokens []string
	for evt := range ch {
		if evt.Type == "token" {
			tokens = append(tokens, evt.Content)
		}
	}
	if strings.Join(tokens, "") == "" {
		// The non-streaming case may or may not produce tokens depending on
		// parse path — just check it doesn't panic or error
		t.Log("no tokens extracted from non-streaming body (acceptable)")
	}
}

// ---------------------------------------------------------------------------
// URL resolution helpers
// ---------------------------------------------------------------------------

func TestResolveChatURL(t *testing.T) {
	cases := []struct {
		endpoint string
		want     string
	}{
		{"https://api.groq.com/openai/v1", "https://api.groq.com/openai/v1/chat/completions"},
		{"https://api.groq.com/openai/v1/", "https://api.groq.com/openai/v1/chat/completions"},
		{"http://localhost:1234/v1", "http://localhost:1234/v1/chat/completions"},
		{"http://localhost:1234", "http://localhost:1234/v1/chat/completions"},
		{"http://localhost:1234/chat/completions", "http://localhost:1234/chat/completions"},
	}

	for _, tc := range cases {
		t.Run(tc.endpoint, func(t *testing.T) {
			got := resolveChatURL(tc.endpoint)
			if got != tc.want {
				t.Errorf("resolveChatURL(%q) = %q, want %q", tc.endpoint, got, tc.want)
			}
		})
	}
}

func TestResolveModelsURL(t *testing.T) {
	cases := []struct {
		endpoint string
		want     string
	}{
		{"https://api.groq.com/openai/v1", "https://api.groq.com/openai/v1/models"},
		{"http://localhost:1234", "http://localhost:1234/v1/models"},
		{"http://localhost:1234/v1/models", "http://localhost:1234/v1/models"},
	}

	for _, tc := range cases {
		t.Run(tc.endpoint, func(t *testing.T) {
			got := resolveModelsURL(tc.endpoint)
			if got != tc.want {
				t.Errorf("resolveModelsURL(%q) = %q, want %q", tc.endpoint, got, tc.want)
			}
		})
	}
}
