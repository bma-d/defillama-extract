package api

import (
	"context"
	"crypto/tls"
	"math/rand"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	utls "github.com/refraction-networking/utls"
	"golang.org/x/net/http2"
)

// UserAgents contains realistic browser user agents (2024-2025).
// Using only Chrome UAs for consistency with TLS fingerprint.
var UserAgents = []string{
	// Chrome on Windows (most common)
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",

	// Chrome on macOS
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 13_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36",

	// Chrome on Linux
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
}

// HeaderRandomizer generates realistic randomized HTTP headers.
type HeaderRandomizer struct {
	acceptLanguages []string
	rng             *rand.Rand
	mu              sync.Mutex
}

// NewHeaderRandomizer creates a new header randomizer.
func NewHeaderRandomizer() *HeaderRandomizer {
	return &HeaderRandomizer{
		acceptLanguages: []string{
			"en-US,en;q=0.9",
			"en-US,en;q=0.9,es;q=0.8",
			"en-GB,en;q=0.9,en-US;q=0.8",
			"en-US,en;q=0.9,fr;q=0.8",
			"en-US,en;q=0.9,de;q=0.8",
		},
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

// utlsRoundTripper is an http.RoundTripper that uses utls for TLS with HTTP/2 support.
type utlsRoundTripper struct {
	dialer        *net.Dialer
	h2Transport   *http2.Transport
	h1Transport   *http.Transport
	tlsConfigSpec utls.ClientHelloID
}

// NewUtlsRoundTripper creates a new round tripper with Chrome TLS fingerprint.
func NewUtlsRoundTripper() *utlsRoundTripper {
	dialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}

	rt := &utlsRoundTripper{
		dialer:        dialer,
		tlsConfigSpec: utls.HelloChrome_131,
	}

	// HTTP/2 transport for HTTPS
	rt.h2Transport = &http2.Transport{
		DialTLSContext: rt.dialTLSContext,
	}

	// HTTP/1.1 transport for non-TLS (plain HTTP)
	rt.h1Transport = &http.Transport{
		DialContext:           dialer.DialContext,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    false,
	}

	return rt
}

// dialTLSContext creates a TLS connection with Chrome's fingerprint for HTTP/2.
func (rt *utlsRoundTripper) dialTLSContext(ctx context.Context, network, addr string, _ *tls.Config) (net.Conn, error) {
	conn, err := rt.dialer.DialContext(ctx, network, addr)
	if err != nil {
		return nil, err
	}

	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		host = addr
	}

	tlsConn := utls.UClient(conn, &utls.Config{
		ServerName:         host,
		InsecureSkipVerify: false,
		NextProtos:         []string{"h2", "http/1.1"},
	}, rt.tlsConfigSpec)

	if err := tlsConn.HandshakeContext(ctx); err != nil {
		conn.Close()
		return nil, err
	}

	return tlsConn, nil
}

// RoundTrip implements http.RoundTripper.
func (rt *utlsRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	if req.URL.Scheme == "https" {
		return rt.h2Transport.RoundTrip(req)
	}
	return rt.h1Transport.RoundTrip(req)
}

// NewBrowserTransport creates an HTTP transport configured to mimic Chrome browser TLS fingerprint.
// It uses utls library to impersonate Chrome 131's TLS fingerprint with HTTP/2 support.
func NewBrowserTransport() http.RoundTripper {
	return NewUtlsRoundTripper()
}

// NewStandardTransport creates a standard HTTP transport without TLS fingerprinting.
// Used for testing and non-HTTPS connections.
func NewStandardTransport() *http.Transport {
	return &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
			MaxVersion: tls.VersionTLS13,
		},
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          100,
		MaxIdleConnsPerHost:   10,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
		DisableCompression:    false,
	}
}

// randomChoice returns a random element from a slice.
func (hr *HeaderRandomizer) randomChoice(choices []string) string {
	hr.mu.Lock()
	defer hr.mu.Unlock()
	return choices[hr.rng.Intn(len(choices))]
}

// RandomUserAgent returns a random user agent string.
func (hr *HeaderRandomizer) RandomUserAgent() string {
	hr.mu.Lock()
	defer hr.mu.Unlock()
	return UserAgents[hr.rng.Intn(len(UserAgents))]
}

// ApplyHeaders applies browser-like HTTP headers to a request.
// Headers are set in an order that matches Chrome browser behavior.
// Note: We don't set Accept-Encoding to let Go's http.Client handle
// compression automatically (it will add gzip and decompress responses).
func (hr *HeaderRandomizer) ApplyHeaders(req *http.Request) {
	ua := hr.RandomUserAgent()

	// Set Host header (usually automatic but explicit for clarity)
	if req.URL != nil {
		req.Host = req.URL.Host
	}

	// Chrome sends headers in this specific order
	// Using http.Header directly to control order better
	req.Header = make(http.Header)

	// sec-ch-ua headers come first in Chrome
	req.Header["sec-ch-ua"] = []string{generateSecChUa(ua)}
	req.Header["sec-ch-ua-mobile"] = []string{"?0"}
	req.Header["sec-ch-ua-platform"] = []string{extractPlatform(ua)}

	// Upgrade-Insecure-Requests
	req.Header["Upgrade-Insecure-Requests"] = []string{"1"}

	// User-Agent
	req.Header["User-Agent"] = []string{ua}

	// Accept headers
	req.Header["Accept"] = []string{"application/json, text/plain, */*"}
	req.Header["Accept-Language"] = []string{hr.randomChoice(hr.acceptLanguages)}
	// Don't set Accept-Encoding - let Go's http.Client handle it automatically

	// Sec-Fetch headers
	req.Header["Sec-Fetch-Site"] = []string{"cross-site"}
	req.Header["Sec-Fetch-Mode"] = []string{"cors"}
	req.Header["Sec-Fetch-Dest"] = []string{"empty"}

	// Referer - simulating coming from DeFiLlama website
	req.Header["Referer"] = []string{"https://defillama.com/"}
	req.Header["Origin"] = []string{"https://defillama.com"}

	// Priority (Chrome 131+)
	req.Header["Priority"] = []string{"u=1, i"}
}

// ApplyHeadersForAPI applies headers specifically for API/XHR requests.
// Note: We don't set Accept-Encoding to let Go's http.Client handle
// compression automatically (it will add gzip and decompress responses).
func (hr *HeaderRandomizer) ApplyHeadersForAPI(req *http.Request) {
	ua := hr.RandomUserAgent()

	if req.URL != nil {
		req.Host = req.URL.Host
	}

	req.Header = make(http.Header)

	// sec-ch-ua headers
	req.Header["sec-ch-ua"] = []string{generateSecChUa(ua)}
	req.Header["sec-ch-ua-mobile"] = []string{"?0"}
	req.Header["sec-ch-ua-platform"] = []string{extractPlatform(ua)}

	// User-Agent
	req.Header["User-Agent"] = []string{ua}

	// Accept for API requests - prefer JSON
	req.Header["Accept"] = []string{"application/json, text/plain, */*"}
	req.Header["Accept-Language"] = []string{hr.randomChoice(hr.acceptLanguages)}
	// Don't set Accept-Encoding - let Go's http.Client handle it automatically
	// This ensures proper decompression of gzip/deflate responses

	// Sec-Fetch headers for CORS/API requests
	req.Header["Sec-Fetch-Site"] = []string{"cross-site"}
	req.Header["Sec-Fetch-Mode"] = []string{"cors"}
	req.Header["Sec-Fetch-Dest"] = []string{"empty"}

	// Origin and Referer for CORS
	req.Header["Origin"] = []string{"https://defillama.com"}
	req.Header["Referer"] = []string{"https://defillama.com/"}

	// Priority
	req.Header["Priority"] = []string{"u=1, i"}
}

// generateSecChUa generates Sec-Ch-Ua header based on Chrome version in UA.
func generateSecChUa(ua string) string {
	if strings.Contains(ua, "Chrome/131") {
		return `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`
	} else if strings.Contains(ua, "Chrome/130") {
		return `"Google Chrome";v="130", "Chromium";v="130", "Not_A Brand";v="24"`
	} else if strings.Contains(ua, "Chrome/129") {
		return `"Google Chrome";v="129", "Chromium";v="129", "Not_A Brand";v="24"`
	}
	return `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`
}

// extractPlatform extracts platform from user agent for Sec-Ch-Ua-Platform.
func extractPlatform(ua string) string {
	if strings.Contains(ua, "Windows") {
		return `"Windows"`
	} else if strings.Contains(ua, "Macintosh") {
		return `"macOS"`
	} else if strings.Contains(ua, "Linux") {
		return `"Linux"`
	}
	return `"Unknown"`
}
