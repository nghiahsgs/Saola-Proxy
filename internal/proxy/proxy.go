// Package proxy provides an HTTPS MITM proxy that sanitizes PII in API requests
// to api.anthropic.com and rehydrates placeholders in responses.
package proxy

import (
	"bytes"
	"crypto/tls"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/elazarl/goproxy"
	"github.com/nguyennghia/saola-proxy/internal/audit"
	"github.com/nguyennghia/saola-proxy/internal/sanitizer"
	"github.com/nguyennghia/saola-proxy/internal/scanner"
)

// ProxyServer is an HTTPS MITM proxy that sanitizes PII in API requests.
type ProxyServer struct {
	listenAddr string
	sanitizer  *sanitizer.Sanitizer
	rehydrator *sanitizer.Rehydrator
	proxy      *goproxy.ProxyHttpServer
}

// NewProxyServer creates a ProxyServer that listens on addr and applies
// PII sanitization to traffic targeting api.anthropic.com.
// If ca is provided, it is used for MITM certificate signing.
// If registry, table, and session are provided, a dashboard is served at http://localhost:<port>/.
func NewProxyServer(addr string, san *sanitizer.Sanitizer, reh *sanitizer.Rehydrator, ca *tls.Certificate, reg *scanner.PatternRegistry, table *sanitizer.MappingTable, session *audit.Session) *ProxyServer {
	p := &ProxyServer{
		listenAddr: addr,
		sanitizer:  san,
		rehydrator: reh,
		proxy:      goproxy.NewProxyHttpServer(),
	}
	// Use custom CA for MITM if provided.
	if ca != nil {
		goproxy.GoproxyCa = *ca
		goproxy.OkConnect = &goproxy.ConnectAction{Action: goproxy.ConnectMitm, TLSConfig: goproxy.TLSConfigFromCA(ca)}
		goproxy.MitmConnect = &goproxy.ConnectAction{Action: goproxy.ConnectMitm, TLSConfig: goproxy.TLSConfigFromCA(ca)}
	}
	// Serve dashboard for direct (non-proxy) requests.
	if reg != nil && table != nil && session != nil {
		p.proxy.NonproxyHandler = newDashboardHandler(reg, table, session)
	}
	p.setup()
	return p
}

// setup registers MITM and request/response handlers for api.anthropic.com.
func (p *ProxyServer) setup() {
	isAnthropic := goproxy.DstHostIs("api.anthropic.com")

	// MITM only Anthropic HTTPS traffic; everything else tunnels transparently.
	p.proxy.OnRequest(isAnthropic).HandleConnect(goproxy.AlwaysMitm)

	// Sanitize outbound request bodies.
	p.proxy.OnRequest(isAnthropic).DoFunc(p.handleRequest)

	// Rehydrate inbound response bodies.
	p.proxy.OnResponse(isAnthropic).DoFunc(p.handleResponse)
}

// handleRequest reads the request body, sanitizes PII, and replaces the body.
func (p *ProxyServer) handleRequest(req *http.Request, ctx *goproxy.ProxyCtx) (*http.Request, *http.Response) {
	if req.Body == nil || req.ContentLength == 0 {
		return req, nil
	}

	body, err := io.ReadAll(req.Body)
	_ = req.Body.Close()
	if err != nil {
		log.Printf("saola proxy: read request body: %v", err)
		req.Body = io.NopCloser(bytes.NewReader(body))
		return req, nil
	}

	sanitized := p.sanitizer.Sanitize(string(body))

	sanitizedBytes := []byte(sanitized)
	req.Body = io.NopCloser(bytes.NewReader(sanitizedBytes))
	req.ContentLength = int64(len(sanitizedBytes))

	if len(sanitized) != len(body) {
		log.Printf("saola proxy: sanitized request body (%d → %d bytes)", len(body), len(sanitized))
	}

	return req, nil
}

// handleResponse rehydrates placeholders in response bodies.
// For SSE streaming (text/event-stream), wraps body with streaming reader.
// For regular responses, reads full body and rehydrates at once.
func (p *ProxyServer) handleResponse(resp *http.Response, ctx *goproxy.ProxyCtx) *http.Response {
	if resp == nil || resp.Body == nil {
		return resp
	}

	ct := resp.Header.Get("Content-Type")

	// SSE streaming: wrap body with streaming rehydrator (on-the-fly).
	if strings.Contains(ct, "text/event-stream") {
		log.Printf("saola proxy: rehydrating SSE stream")
		resp.Body = newStreamingReader(resp.Body, p.rehydrator.Rehydrate)
		resp.ContentLength = -1
		resp.Header.Del("Content-Length")
		return resp
	}

	// Regular response: read full body, rehydrate, replace.
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		log.Printf("saola proxy: read response body: %v", err)
		resp.Body = io.NopCloser(bytes.NewReader(body))
		return resp
	}

	rehydrated := p.rehydrator.Rehydrate(string(body))

	rehydratedBytes := []byte(rehydrated)
	resp.Body = io.NopCloser(bytes.NewReader(rehydratedBytes))
	resp.ContentLength = int64(len(rehydratedBytes))

	if len(rehydrated) != len(body) {
		log.Printf("saola proxy: rehydrated response body (%d \u2192 %d bytes)", len(body), len(rehydrated))
	}

	return resp
}

// Handler returns the underlying http.Handler for use in tests (e.g. httptest.NewServer).
func (p *ProxyServer) Handler() http.Handler {
	return p.proxy
}

// Start begins listening and serving proxy requests. Blocking call.
func (p *ProxyServer) Start() error {
	log.Printf("saola proxy: listening on %s", p.listenAddr)
	return http.ListenAndServe(p.listenAddr, p.proxy)
}
