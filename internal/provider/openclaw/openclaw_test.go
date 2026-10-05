package openclaw

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/NubiMa/hikari-cli/internal/config"
	"github.com/NubiMa/hikari-cli/internal/provider"
)

func TestOpenClawChatStreaming(t *testing.T) {
	// Mock OpenClaw server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/chat/completions" || r.URL.Path == "/chat/completions" {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)

			flusher, ok := w.(http.Flusher)
			if !ok {
				t.Fatal("expected http.Flusher")
			}

			// Send tokens
			fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"Hello \"}}]}\n\n")
			flusher.Flush()
			time.Sleep(10 * time.Millisecond)

			fmt.Fprintf(w, "data: {\"choices\":[{\"delta\":{\"content\":\"from OpenClaw!\"}}]}\n\n")
			flusher.Flush()

			fmt.Fprintf(w, "data: [DONE]\n\n")
			flusher.Flush()
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	cfg := config.ProviderConfig{
		Type:     "openclaw",
		Endpoint: server.URL,
		Token:    "test-token",
	}

	prov, err := New("openclaw-test", cfg)
	if err != nil {
		t.Fatalf("New provider failed: %v", err)
	}

	ctx := context.Background()
	events, err := prov.Chat(ctx, nil, "Hi")
	if err != nil {
		t.Fatalf("Chat() failed: %v", err)
	}

	var output string
	for e := range events {
		if e.Type == provider.EventToken {
			output += e.Content
		}
	}

	expected := "Hello from OpenClaw!"
	if output != expected {
		t.Errorf("expected output %q, got %q", expected, output)
	}
}

func TestOpenClawStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	cfg := config.ProviderConfig{
		Type:     "openclaw",
		Endpoint: server.URL,
		Token:    "test-token",
	}

	prov, err := New("openclaw-test", cfg)
	if err != nil {
		t.Fatalf("New provider failed: %v", err)
	}

	status, err := prov.Status(context.Background())
	if err != nil {
		t.Fatalf("Status() failed: %v", err)
	}
	if !status.Connected {
		t.Errorf("expected status Connected = true")
	}
}
