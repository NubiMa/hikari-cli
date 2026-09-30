// Package hermes implements the Hikari Provider interface for Hermes.
//
// Hermes is an agent provider with tool calling, memory, and agent execution.
//
// TODO: Replace stub implementations with actual Hermes API calls once the
// API contract is confirmed.
package hermes

import (
	"context"
	"fmt"
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

func (p *Provider) Models(_ context.Context) ([]provider.Model, error) {
	return nil, fmt.Errorf("hermes: model listing not supported")
}

// Chat sends a message to Hermes and streams the response.
// TODO: Implement once Hermes API spec is available.
func (p *Provider) Chat(_ context.Context, _ []provider.Message, _ string) (<-chan provider.Event, error) {
	return nil, fmt.Errorf("hermes: Chat() not yet implemented — awaiting API specification")
}
