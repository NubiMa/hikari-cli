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
	name     string
	cfg      config.ProviderConfig
	client   *http.Client
	endpoint string
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
		name:     name,
		cfg:      cfg,
		client:   &http.Client{Timeout: timeout},
		endpoint: strings.TrimRight(cfg.Endpoint, "/"),
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
		return provider.ProviderStatus{Connected: false, Message: err.Error()}, nil
	}
	defer resp.Body.Close()

	return provider.ProviderStatus{
		Connected: resp.StatusCode < 400,
		Latency:   time.Since(start),
		Message:   fmt.Sprintf("HTTP %d", resp.StatusCode),
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
