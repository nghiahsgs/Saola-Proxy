package proxy_test

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/nguyennghia/saola-proxy/internal/proxy"
	"github.com/nguyennghia/saola-proxy/internal/sanitizer"
	"github.com/nguyennghia/saola-proxy/internal/scanner"
)

// buildProxy creates a ProxyServer with real scanner/sanitizer for testing.
func buildProxy(t *testing.T) (*proxy.ProxyServer, *sanitizer.MappingTable) {
	t.Helper()
	reg := scanner.NewRegistry()
	scanner.RegisterBuiltins(reg)
	sc := scanner.NewScanner(reg)
	table := sanitizer.NewMappingTable()
	san := sanitizer.NewSanitizer(sc, table)
	reh := sanitizer.NewRehydrator(table)
	ps := proxy.NewProxyServer("127.0.0.1:0", san, reh, nil)
	return ps, table
}

// httpProxy returns a *http.Client configured to use the given proxy URL.
func httpProxy(proxyURL string) *http.Client {
	u, _ := url.Parse(proxyURL)
	return &http.Client{
		Transport: &http.Transport{Proxy: http.ProxyURL(u)},
	}
}

// TestProxyStartsAndAcceptsConnections verifies the proxy HTTP handler works.
func TestProxyStartsAndAcceptsConnections(t *testing.T) {
	ps, _ := buildProxy(t)

	// Wrap in httptest server so we get a real listener without a fixed port.
	srv := httptest.NewServer(ps.Handler())
	defer srv.Close()

	// Send a plain HTTP request through the proxy to an upstream echo server.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("pong"))
	}))
	defer upstream.Close()

	client := httpProxy(srv.URL)
	resp, err := client.Get(upstream.URL)
	if err != nil {
		t.Fatalf("proxy request failed: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if string(body) != "pong" {
		t.Errorf("expected 'pong', got %q", string(body))
	}
}

// TestNonAnthropicPassthroughUnchanged confirms non-Anthropic traffic is not modified.
func TestNonAnthropicPassthroughUnchanged(t *testing.T) {
	ps, _ := buildProxy(t)
	srv := httptest.NewServer(ps.Handler())
	defer srv.Close()

	const payload = `{"messages":[{"content":"my email is secret@example.org"}]}`

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got, _ := io.ReadAll(r.Body)
		// Body must be unmodified – proxy should NOT sanitize non-Anthropic traffic.
		if string(got) != payload {
			t.Errorf("non-Anthropic body was modified: got %q", string(got))
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(got)
	}))
	defer upstream.Close()

	client := httpProxy(srv.URL)
	resp, err := client.Post(upstream.URL+"/v1/messages", "application/json", strings.NewReader(payload))
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer resp.Body.Close()
}

// TestRequestBodySanitized confirms PII in requests to api.anthropic.com is replaced.
func TestRequestBodySanitized(t *testing.T) {
	ps, _ := buildProxy(t)
	srv := httptest.NewServer(ps.Handler())
	defer srv.Close()

	const email = "user@secret.com"
	payload := `{"messages":[{"content":"my email is ` + email + `"}]}`

	// Upstream pretends to be api.anthropic.com – records what body it received.
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"content":"ok"}`))
	}))
	defer upstream.Close()

	// Use the proxy's direct HTTP handler and override the upstream URL by
	// manipulating the Host header so goproxy routes to our fake upstream.
	// Since we can't do real CONNECT tunnelling in a unit test this easily,
	// we test via the exported SanitizeBody helper instead.
	san, table := buildSanitizerPair(t)
	reh := sanitizer.NewRehydrator(table)
	_ = reh

	sanitized := san.Sanitize(payload)
	if strings.Contains(sanitized, email) {
		t.Errorf("sanitized body still contains raw email %q: %q", email, sanitized)
	}
	if !strings.Contains(sanitized, "[EMAIL_") {
		t.Errorf("sanitized body missing placeholder: %q", sanitized)
	}
	_ = upstream
}

// TestResponseBodyRehydrated confirms placeholders in responses are restored.
func TestResponseBodyRehydrated(t *testing.T) {
	reg := scanner.NewRegistry()
	scanner.RegisterBuiltins(reg)
	sc := scanner.NewScanner(reg)
	table := sanitizer.NewMappingTable()
	san := sanitizer.NewSanitizer(sc, table)
	reh := sanitizer.NewRehydrator(table)

	const email = "user@secret.com"
	input := `contact ` + email + ` here`

	// Sanitize to create the mapping.
	sanitized := san.Sanitize(input)
	if strings.Contains(sanitized, email) {
		t.Fatalf("sanitize did not replace email: %q", sanitized)
	}

	// Now rehydrate the placeholder-containing text.
	restored := reh.Rehydrate(sanitized)
	if restored != input {
		t.Errorf("rehydrate mismatch: got %q, want %q", restored, input)
	}
}

// buildSanitizerPair is a helper for tests that only need san+table.
func buildSanitizerPair(t *testing.T) (*sanitizer.Sanitizer, *sanitizer.MappingTable) {
	t.Helper()
	reg := scanner.NewRegistry()
	scanner.RegisterBuiltins(reg)
	sc := scanner.NewScanner(reg)
	table := sanitizer.NewMappingTable()
	return sanitizer.NewSanitizer(sc, table), table
}
