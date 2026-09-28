// Package ollama implements the Hikari Provider interface for Ollama.
//
// Ollama API reference: https://github.com/ollama/ollama/blob/main/docs/api.md
//
// Ollama is a model provider: it supports streaming chat completions and model
// listing but does not have agent, tool-calling, or memory capabilities.
package ollama

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/NubiMa/hikari-cli/internal/config"
	"github.com/NubiMa/hikari-cli/internal/provider"
)

const (
	defaultTimeout = 120 * time.Second
	endpointChat   = "/api/chat"
	endpointTags   = "/api/tags"
)

// ---------------------------------------------------------------------------
// Provider struct
// ---------------------------------------------------------------------------

// Provider implements provider.Provider for Ollama.
type Provider struct {
	name     string
	cfg      config.ProviderConfig
	client   *http.Client
	endpoint string // normalised (no trailing slash)
}

// New creates an OllamaProvider from a ProviderConfig.
// This is the constructor registered with the provider Registry.
func New(name string, cfg config.ProviderConfig) (provider.Provider, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("ollama provider %q: endpoint is required", name)
	}

	timeout := defaultTimeout
	if cfg.TimeoutSeconds > 0 {
		timeout = time.Duration(cfg.TimeoutSeconds) * time.Second
	}

	transport := &http.Transport{}
	if cfg.TLSSkipVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // user-opted
	}

	client := &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}

	return &Provider{
		name:     name,
		cfg:      cfg,
		client:   client,
		endpoint: strings.TrimRight(cfg.Endpoint, "/"),
	}, nil
}

// ---------------------------------------------------------------------------
// Provider interface implementation
// ---------------------------------------------------------------------------

func (p *Provider) Name() string {
	return fmt.Sprintf("Ollama (%s)", p.name)
}

func (p *Provider) Capabilities() provider.Capabilities {
	return provider.OllamaCapabilities()
}

// Connect validates that the Ollama server is reachable by listing models.
func (p *Provider) Connect(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.endpoint+endpointTags, nil)
	if err != nil {
		return fmt.Errorf("ollama connect: building request: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("ollama connect: cannot reach %s: %w", p.endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("ollama connect: server returned %d", resp.StatusCode)
	}
	return nil
}

// Close is a no-op for Ollama (stateless HTTP).
func (p *Provider) Close() error { return nil }

// Status returns the health of the Ollama connection.
func (p *Provider) Status(ctx context.Context) (provider.ProviderStatus, error) {
	start := time.Now()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.endpoint+endpointTags, nil)
	if err != nil {
		return provider.ProviderStatus{}, err
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return provider.ProviderStatus{
			Connected: false,
			Message:   err.Error(),
		}, nil
	}
	defer resp.Body.Close()

	latency := time.Since(start)
	if resp.StatusCode == http.StatusOK {
		return provider.ProviderStatus{
			Connected: true,
			Latency:   latency,
			Message:   "connected",
		}, nil
	}
	return provider.ProviderStatus{
		Connected: false,
		Latency:   latency,
		Message:   fmt.Sprintf("HTTP %d", resp.StatusCode),
	}, nil
}

// ---------------------------------------------------------------------------
// Models
// ---------------------------------------------------------------------------

// tagsResponse mirrors the Ollama GET /api/tags response.
type tagsResponse struct {
	Models []struct {
		Name string `json:"name"`
		Size int64  `json:"size"`
	} `json:"models"`
}

// Models returns the list of locally available models on the Ollama server.
func (p *Provider) Models(ctx context.Context) ([]provider.Model, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.endpoint+endpointTags, nil)
	if err != nil {
		return nil, fmt.Errorf("ollama models: %w", err)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ollama models: server returned %d", resp.StatusCode)
	}

	var tags tagsResponse
	if err := json.NewDecoder(resp.Body).Decode(&tags); err != nil {
		return nil, fmt.Errorf("ollama models: decoding response: %w", err)
	}

	models := make([]provider.Model, 0, len(tags.Models))
	for _, m := range tags.Models {
		models = append(models, provider.Model{
			ID:   m.Name,
			Name: m.Name,
			Size: m.Size,
		})
	}
	return models, nil
}

// ---------------------------------------------------------------------------
// Chat
// ---------------------------------------------------------------------------

// chatRequest mirrors the Ollama POST /api/chat request body.
type chatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatChunk mirrors a single streamed JSON line from Ollama.
type chatChunk struct {
	Message ollamaMessage `json:"message"`
	Done    bool          `json:"done"`
	Error   string        `json:"error,omitempty"`
}

// Chat sends a conversation to Ollama and streams the response as Events.
// The caller must drain the returned channel until it is closed.
func (p *Provider) Chat(ctx context.Context, history []provider.Message, message string) (<-chan provider.Event, error) {
	model := p.cfg.Model
	if model == "" {
		return nil, fmt.Errorf("ollama: no model configured for provider %q; set model in config.toml or use /model", p.name)
	}

	// Build message list: history + new user message.
	msgs := make([]ollamaMessage, 0, len(history)+1)
	for _, h := range history {
		msgs = append(msgs, ollamaMessage{Role: string(h.Role), Content: h.Content})
	}
	msgs = append(msgs, ollamaMessage{Role: "user", Content: message})

	body := chatRequest{
		Model:    model,
		Messages: msgs,
		Stream:   true,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("ollama chat: marshalling request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, p.endpoint+endpointChat, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("ollama chat: building request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("ollama chat: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		resp.Body.Close()
		return nil, fmt.Errorf("ollama chat: server returned %d", resp.StatusCode)
	}

	events := make(chan provider.Event, 64)

	go func() {
		defer close(events)
		defer resp.Body.Close()

		scanner := bufio.NewScanner(resp.Body)
		for scanner.Scan() {
			line := scanner.Bytes()
			if len(line) == 0 {
				continue
			}

			var chunk chatChunk
			if err := json.Unmarshal(line, &chunk); err != nil {
				select {
				case events <- provider.Event{Type: provider.EventError, Error: fmt.Errorf("ollama: decoding chunk: %w", err)}:
				case <-ctx.Done():
				}
				return
			}

			if chunk.Error != "" {
				select {
				case events <- provider.Event{Type: provider.EventError, Error: fmt.Errorf("ollama: %s", chunk.Error)}:
				case <-ctx.Done():
				}
				return
			}

			if chunk.Message.Content != "" {
				select {
				case events <- provider.Event{Type: provider.EventToken, Content: chunk.Message.Content}:
				case <-ctx.Done():
					return
				}
			}

			if chunk.Done {
				select {
				case events <- provider.Event{Type: provider.EventDone}:
				case <-ctx.Done():
				}
				return
			}
		}

		if err := scanner.Err(); err != nil {
			if ctx.Err() == nil {
				select {
				case events <- provider.Event{Type: provider.EventError, Error: fmt.Errorf("ollama: reading stream: %w", err)}:
				default:
				}
			}
		}
	}()

	return events, nil
}
