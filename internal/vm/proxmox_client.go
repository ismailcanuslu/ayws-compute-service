package vm

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"

	"github.com/ismailcanuslu/ayws-compute-service/config"
	proxmox "github.com/luthermonson/go-proxmox"
)

// ProxmoxClient wraps the go-proxmox client with our config.
type ProxmoxClient struct {
	client *proxmox.Client
	node   string
}

// NewProxmoxClient creates an authenticated Proxmox API client.
// Returns an error if the config is empty (dev mode — real connection not required).
func NewProxmoxClient(cfg *config.ProxmoxConfig) (*ProxmoxClient, error) {
	if cfg.Host == "" || cfg.TokenSecret == "" {
		return nil, fmt.Errorf("proxmox yapılandırması eksik: host veya token_secret boş")
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: cfg.InsecureTLS}, //nolint:gosec
		},
	}

	c := proxmox.NewClient(cfg.Host,
		proxmox.WithHTTPClient(httpClient),
		proxmox.WithAPIToken(cfg.TokenID, cfg.TokenSecret),
	)

	// Bağlantı sağlığını doğrula
	if _, err := c.Version(context.Background()); err != nil {
		return nil, fmt.Errorf("proxmox bağlantısı başarısız: %w", err)
	}

	return &ProxmoxClient{client: c, node: cfg.Node}, nil
}

// Client returns the underlying go-proxmox client for direct use.
func (p *ProxmoxClient) Client() *proxmox.Client { return p.client }

// Node returns the configured Proxmox node name.
func (p *ProxmoxClient) Node() string { return p.node }
