// Package openclaw implements the Hikari Provider interface for OpenClaw.
//
// OpenClaw is an agent provider. It supports streaming, tool calling, memory,
// and autonomous agent execution.
//
// TODO: Replace stub implementations with actual OpenClaw API calls once the
// API contract (endpoints, authentication scheme, request/response schema) is
// confirmed. The interface and capability declarations are already correct.
package openclaw

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/nubiv/hikari/internal/config"
	"github.com/nubiv/hikari/internal/provider"
)

const defaultTimeout = 120 * time.Second

// Provider implements provider.Provider for OpenClaw.
type Provider struct {
	name     string
	cfg      config.ProviderConfig
	client   *http.Client
	endpoint string
}

// New creates an OpenClawProvider from a ProviderConfig.
func New(name string, cfg config.ProviderConfig) (provider.Provider, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("openclaw provider %q: endpoint is required", name)
	}
	if cfg.Token == "" {
		return nil, fmt.Errorf("openclaw provider %q: token is required (or set HIKARI_%s_TOKEN)", name,
			strings.ToUpper(strings.ReplaceAll(name, "-", "_")))
	}

	timeout := defaultTimeout
	if cfg.TimeoutSeconds > 0 {
		timeout = time.Duration(cfg.TimeoutSeconds) * time.Second
	}

	return &Provider{
		name:     name,
		cfg:      cfg,
		client:   &http.Client{Timeout: timeout},
		endpoint: strings.TrimRight(cfg.Endpoint, "/"),
	}, nil
}

func (p *Provider) Name() string { return fmt.Sprintf("OpenClaw (%s)", p.name) }

func (p *Provider) Capabilities() provider.Capabilities {
	return provider.OpenClawCapabilities()
}

// Connect validates that the OpenClaw server is reachable.
// TODO: Replace with actual health-check endpoint once API is known.
func (p *Provider) Connect(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, p.endpoint+"/health", nil)
	if err != nil {
		return fmt.Errorf("openclaw connect: %w", err)
	}
	p.setAuthHeader(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("openclaw connect: cannot reach %s: %w", p.endpoint, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("openclaw connect: server returned %d", resp.StatusCode)
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
	p.setAuthHeader(req)

	resp, err := p.client.Do(req)
	if err != nil {
		return provider.ProviderStatus{Connected: false, Message: err.Error()}, nil
	}
	defer resp.Body.Close()

	return provider.ProviderStatus{
		Connected: resp.StatusCode < 400,
		Latency:   time.Since(start),
		Message:   fmt.Sprintf("HTTP %d", resp.StatusCode),
	}, nil
}

// Models returns an error because OpenClaw manages its own models.
func (p *Provider) Models(_ context.Context) ([]provider.Model, error) {
	return nil, fmt.Errorf("openclaw: model listing not supported (OpenClaw manages its own models)")
}

// Chat sends a message to OpenClaw and streams the response.
// TODO: Implement actual OpenClaw chat API once endpoint/schema is known.
func (p *Provider) Chat(_ context.Context, _ []provider.Message, _ string) (<-chan provider.Event, error) {
	return nil, fmt.Errorf("openclaw: Chat() not yet implemented — awaiting API specification")
}

// setAuthHeader attaches the bearer token to the request.
func (p *Provider) setAuthHeader(req *http.Request) {
	if p.cfg.Token != "" {
		req.Header.Set("Authorization", "Bearer "+p.cfg.Token)
	}
}
