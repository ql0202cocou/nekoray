package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"time"

	box "github.com/sagernet/sing-box"
	M "github.com/sagernet/sing/common/metadata"
)

// nekoDialContext replaces boxapi.DialContext
func nekoDialContext(ctx context.Context, b *box.Box, network, addr string) (net.Conn, error) {
	if b == nil {
		return nil, fmt.Errorf("box instance is nil")
	}
	outbound := b.Outbound().Default()
	return outbound.DialContext(ctx, network, M.ParseSocksaddr(addr))
}

// nekoCreateProxyHttpClient replaces boxapi.CreateProxyHttpClient
func nekoCreateProxyHttpClient(b *box.Box) *http.Client {
	if b == nil {
		return &http.Client{}
	}
	outbound := b.Outbound().Default()
	return &http.Client{
		Transport: &http.Transport{
			ForceAttemptHTTP2:   true,
			TLSHandshakeTimeout: 10 * time.Second,
			DialContext: func(ctx context.Context, network, addr string) (net.Conn, error) {
				return outbound.DialContext(ctx, network, M.ParseSocksaddr(addr))
			},
			TLSClientConfig: &tls.Config{},
		},
	}
}
