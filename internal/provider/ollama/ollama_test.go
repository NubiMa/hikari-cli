package ollama_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/NubiMa/hikari-cli/internal/config"
	"github.com/NubiMa/hikari-cli/internal/provider"
	"github.com/NubiMa/hikari-cli/internal/provider/ollama"
)

// newTestProvider creates an Ollama provider pointed at a test server.
func newTestProvider(t *testing.T, endpoint string) provider.Provider {
	t.Helper()
	p, err := ollama.New("test", config.ProviderConfig{
		Endpoint: endpoint,
		Model:    "llama3.2",
	})
	if err != nil {
		t.Fatalf("creating provider: %v", err)
	}
	return p
}

func TestCapabilities(t *testing.T) {
	p, _ := ollama.New("test", config.ProviderConfig{Endpoint: "http://localhost"})
	caps := p.Capabilities()
	if !caps.Chat {
		t.Error("expected Chat capability")
	}
	if !caps.Streaming {
		t.Error("expected Streaming capability")
	}
	if !caps.Models {
		t.Error("expected Models capability")
	}
	if caps.ToolCalling {
		t.Error("Ollama should NOT have ToolCalling capability")
	}
	if caps.AgentExecution {
		t.Error("Ollama should NOT have AgentExecution capability")
	}
}

func TestConnect(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/tags" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]any{"models": []any{}})
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	p := newTestProvider(t, srv.URL)
	ctx := context.Background()
	if err := p.Connect(ctx); err != nil {
		t.Fatalf("Connect() error: %v", err)
	}
}

func TestConnectFailure(t *testing.T) {
	p := newTestProvider(t, "http://127.0.0.1:1") // nothing listening
	ctx := context.Background()
	if err := p.Connect(ctx); err == nil {
		t.Fatal("expected Connect() to fail on unreachable server")
	}
}

func TestModels(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]any{
			"models": []map[string]any{
				{"name": "llama3.2", "size": int64(4_000_000_000)},
				{"name": "mistral", "size": int64(7_000_000_000)},
			},
		})
	}))
	defer srv.Close()

	p := newTestProvider(t, srv.URL)
	models, err := p.Models(context.Background())
	if err != nil {
		t.Fatalf("Models() error: %v", err)
	}
	if len(models) != 2 {
		t.Fatalf("expected 2 models, got %d", len(models))
	}
	if models[0].Name != "llama3.2" {
		t.Errorf("expected llama3.2, got %q", models[0].Name)
	}
}

func TestChat(t *testing.T) {
	// Simulate Ollama streaming NDJSON response
	streamBody := strings.Join([]string{
		`{"message":{"role":"assistant","content":"Hello"},"done":false}`,
		`{"message":{"role":"assistant","content":", world!"},"done":false}`,
		`{"message":{"role":"assistant","content":""},"done":true}`,
	}, "\n")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/chat" {
			w.Header().Set("Content-Type", "application/x-ndjson")
			w.Write([]byte(streamBody))
		}
	}))
	defer srv.Close()

	p := newTestProvider(t, srv.URL)

	ch, err := p.Chat(context.Background(), nil, "Hi")
	if err != nil {
		t.Fatalf("Chat() error: %v", err)
	}

	var tokens []string
	var gotDone bool
	for evt := range ch {
		switch evt.Type {
		case provider.EventToken:
			tokens = append(tokens, evt.Content)
		case provider.EventDone:
			gotDone = true
		case provider.EventError:
			t.Fatalf("unexpected error event: %v", evt.Error)
		}
	}

	if !gotDone {
		t.Error("expected EventDone")
	}
	full := strings.Join(tokens, "")
	if full != "Hello, world!" {
		t.Errorf("expected 'Hello, world!', got %q", full)
	}
}

func TestChatNoModel(t *testing.T) {
	p, _ := ollama.New("test", config.ProviderConfig{
		Endpoint: "http://localhost",
		Model:    "", // no model set
	})
	_, err := p.Chat(context.Background(), nil, "Hi")
	if err == nil {
		t.Fatal("expected error when no model is configured")
	}
}
