// Package hermes implements the Hikari Provider interface for Hermes.
//
// Hermes is an agent provider with tool calling, memory, and agent execution.
//
// TODO: Replace stub implementations with actual Hermes API calls once the
// API contract is confirmed.
package hermes

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/NubiMa/hikari-cli/internal/config"
	"github.com/NubiMa/hikari-cli/internal/provider"
)

const defaultTimeout = 120 * time.Second

// Provider implements provider.Provider for Hermes.
type Provider struct {
	name         string
	cfg          config.ProviderConfig
	client       *http.Client
	streamClient *http.Client
	endpoint     string
}

// New creates a HermesProvider from a ProviderConfig.
func New(name string, cfg config.ProviderConfig) (provider.Provider, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("hermes provider %q: endpoint is required", name)
	}

	timeout := defaultTimeout
	if cfg.TimeoutSeconds > 0 {
		timeout = time.Duration(cfg.TimeoutSeconds) * time.Second
	}

	return &Provider{
		name:         name,
		cfg:          cfg,
		client:       &http.Client{Timeout: timeout},
		streamClient: &http.Client{Timeout: 0},
		endpoint:     strings.TrimRight(cfg.Endpoint, "/"),
	}, nil
}

func (p *Provider) Name() string { return fmt.Sprintf("Hermes (%s)", p.name) }

func (p *Provider) Capabilities() provider.Capabilities {
	return provider.HermesCapabilities()
}

// Connect validates that the Hermes server is reachable.
// TODO: Replace with actual Hermes health-check endpoint.
func (p *Provider) Connect(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.endpoint+"/health", nil)
	if err != nil {
		return fmt.Errorf("hermes connect: %w", err)
	}
	if p.cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+p.cfg.Token)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("hermes connect: cannot reach %s: %w", p.endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("hermes connect: server returned %d", resp.StatusCode)
	}
	return nil
}

func (p *Provider) Close() error { return nil }

func (p *Provider) Status(ctx context.Context) (provider.ProviderStatus, error) {
	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.endpoint+"/health", nil)
	if err != nil {
		return provider.ProviderStatus{}, err
	}
	if p.cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+p.cfg.Token)
	}

	resp, err := p.client.Do(req)
	if err != nil {
		msg := err.Error()
		details := []string{"Make sure the Hermes server is running and the endpoint is correct."}
		if strings.Contains(msg, "connection refused") {
			msg = "connection refused (is Hermes running?)"
		} else if strings.Contains(msg, "no such host") {
			msg = "hostname not found (check endpoint in config)"
		} else if strings.Contains(msg, "timeout") || strings.Contains(msg, "deadline exceeded") {
			msg = "timed out (server slow or unreachable)"
		}
		return provider.ProviderStatus{
			Connected: false,
			Endpoint:  p.endpoint,
			Message:   msg,
			Details:   details,
		}, nil
	}
	defer resp.Body.Close()

	latency := time.Since(start)
	details := []string{fmt.Sprintf("HTTP %d from %s/health", resp.StatusCode, p.endpoint)}
	switch {
	case resp.StatusCode == http.StatusUnauthorized:
		details = append(details, "Authentication failed — check your token in config.toml")
		return provider.ProviderStatus{
			Connected: false, Latency: latency, Endpoint: p.endpoint,
			Message: "auth failed (HTTP 401)", Details: details,
		}, nil
	case resp.StatusCode == http.StatusForbidden:
		details = append(details, "Access denied — your token may not have the required permissions")
		return provider.ProviderStatus{
			Connected: false, Latency: latency, Endpoint: p.endpoint,
			Message: "forbidden (HTTP 403)", Details: details,
		}, nil
	case resp.StatusCode >= 400:
		return provider.ProviderStatus{
			Connected: false, Latency: latency, Endpoint: p.endpoint,
			Message: fmt.Sprintf("server error (HTTP %d)", resp.StatusCode), Details: details,
		}, nil
	}

	hasToken := p.cfg.Token != ""
	details = append(details, fmt.Sprintf("Auth token configured: %v", hasToken))
	return provider.ProviderStatus{
		Connected: true,
		Latency:   latency,
		Endpoint:  p.endpoint,
		Message:   "online",
		Details:   details,
	}, nil
}

// Models lists models available on Hermes if supported by the server.
func (p *Provider) Models(ctx context.Context) ([]provider.Model, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, resolveModelsURL(p.endpoint), nil)
	if err != nil {
		return nil, fmt.Errorf("hermes models: %w", err)
	}
	p.setAuthHeader(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("hermes models: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("hermes models: server returned HTTP %d", resp.StatusCode)
	}

	var mResp struct {
		Data []struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&mResp); err != nil {
		return nil, fmt.Errorf("hermes models: decoding response: %w", err)
	}

	models := make([]provider.Model, 0, len(mResp.Data))
	for _, m := range mResp.Data {
		name := m.Name
		if name == "" {
			name = m.ID
		}
		models = append(models, provider.Model{
			ID:   m.ID,
			Name: name,
		})
	}
	return models, nil
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model    string        `json:"model,omitempty"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream"`
}

// Chat sends a message history to Hermes and streams response events.
func (p *Provider) Chat(ctx context.Context, history []provider.Message, message string) (<-chan provider.Event, error) {
	model := p.cfg.Model
	if model == "" {
		model = "default"
	}

	msgs := make([]chatMessage, 0, len(history)+1)
	for _, h := range history {
		msgs = append(msgs, chatMessage{
			Role:    string(h.Role),
			Content: h.Content,
		})
	}
	msgs = append(msgs, chatMessage{
		Role:    "user",
		Content: message,
	})

	body := chatRequest{
		Model:    model,
		Messages: msgs,
		Stream:   true,
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("hermes chat: marshalling request: %w", err)
	}

	url := resolveChatURL(p.endpoint)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return nil, fmt.Errorf("hermes chat: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream, application/json")
	p.setAuthHeader(req)

	resp, err := p.streamClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("hermes chat: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		resp.Body.Close()
		return nil, fmt.Errorf("hermes chat: server returned HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(bodyBytes)))
	}

	events := make(chan provider.Event, 64)

	go func() {
		defer close(events)
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)
		for {
			line, err := reader.ReadString('\n')
			if len(line) > 0 {
				line = strings.TrimSpace(line)
				if line != "" {
					parseStreamLine(line, events)
				}
			}
			if err != nil {
				if err != io.EOF && !errors.Is(err, context.Canceled) {
					events <- provider.Event{
						Type:  provider.EventError,
						Error: fmt.Errorf("hermes stream: %w", err),
					}
				}
				break
			}
		}
		events <- provider.Event{Type: provider.EventDone}
	}()

	return events, nil
}

type streamChunk struct {
	Choices []struct {
		Delta struct {
			Content          string `json:"content"`
			ReasoningContent string `json:"reasoning_content"`
			Role             string `json:"role"`
		} `json:"delta"`
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
		Text         string `json:"text"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Content  string `json:"content"`
	Text     string `json:"text"`
	Response string `json:"response"`
	Message  struct {
		Content string `json:"content"`
		Role    string `json:"role"`
	} `json:"message"`
	Error   string `json:"error"`
	Thought string `json:"thought"`
	Status  string `json:"status"`
	Done    bool   `json:"done"`
}

func parseStreamLine(line string, events chan<- provider.Event) {
	if strings.HasPrefix(line, "data:") {
		line = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
	}
	if line == "[DONE]" {
		return
	}
	if !strings.HasPrefix(line, "{") {
		return
	}

	var chunk streamChunk
	if err := json.Unmarshal([]byte(line), &chunk); err != nil {
		return
	}

	if chunk.Error != "" {
		events <- provider.Event{
			Type:  provider.EventError,
			Error: errors.New(chunk.Error),
		}
		return
	}

	if chunk.Thought != "" {
		events <- provider.Event{
			Type:    provider.EventStatus,
			Content: chunk.Thought,
		}
	} else if chunk.Status != "" {
		events <- provider.Event{
			Type:    provider.EventStatus,
			Content: chunk.Status,
		}
	}

	if len(chunk.Choices) > 0 {
		c := chunk.Choices[0]
		if c.Delta.ReasoningContent != "" {
			events <- provider.Event{
				Type:    provider.EventStatus,
				Content: c.Delta.ReasoningContent,
			}
		}
		if c.Delta.Content != "" {
			events <- provider.Event{
				Type:    provider.EventToken,
				Content: c.Delta.Content,
			}
			return
		}
		if c.Text != "" {
			events <- provider.Event{
				Type:    provider.EventToken,
				Content: c.Text,
			}
			return
		}
		if c.Message.Content != "" {
			events <- provider.Event{
				Type:    provider.EventToken,
				Content: c.Message.Content,
			}
			return
		}
	}

	token := chunk.Content
	if token == "" {
		token = chunk.Text
	}
	if token == "" {
		token = chunk.Response
	}
	if token == "" {
		token = chunk.Message.Content
	}
	if token != "" {
		events <- provider.Event{
			Type:    provider.EventToken,
			Content: token,
		}
	}
}

func resolveChatURL(endpoint string) string {
	ep := strings.TrimRight(endpoint, "/")
	if strings.HasSuffix(ep, "/chat/completions") || strings.HasSuffix(ep, "/chat") {
		return ep
	}
	if strings.HasSuffix(ep, "/v1") {
		return ep + "/chat/completions"
	}
	return ep + "/v1/chat/completions"
}

func resolveModelsURL(endpoint string) string {
	ep := strings.TrimRight(endpoint, "/")
	if strings.HasSuffix(ep, "/models") {
		return ep
	}
	if strings.HasSuffix(ep, "/v1") {
		return ep + "/models"
	}
	return ep + "/v1/models"
}

// setAuthHeader attaches the bearer token to the request.
func (p *Provider) setAuthHeader(req *http.Request) {
	if p.cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+p.cfg.Token)
	}
}
