package api

import (
	"net/http"
	"strings"
	"testing"
)

func TestRandomUserAgent(t *testing.T) {
	hr := NewHeaderRandomizer()

	// Get 10 random user agents and verify they're valid
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		ua := hr.RandomUserAgent()
		if ua == "" {
			t.Error("RandomUserAgent returned empty string")
		}
		// Verify it's from our list
		found := false
		for _, validUA := range UserAgents {
			if ua == validUA {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("RandomUserAgent returned unknown UA: %s", ua)
		}
		seen[ua] = true
	}

	// With 100 tries and multiple UAs, we should see variation
	if len(seen) < 2 {
		t.Errorf("Expected more variation in UAs, only saw %d unique UAs", len(seen))
	}
}

func TestApplyHeadersForAPI_SetsRequiredHeaders(t *testing.T) {
	hr := NewHeaderRandomizer()
	req, _ := http.NewRequest("GET", "https://api.llama.fi/oracles", nil)

	hr.ApplyHeadersForAPI(req)

	// Check required headers are set
	// Note: Accept-Encoding is NOT set to let Go handle compression automatically
	requiredHeaders := []string{
		"User-Agent",
		"Accept",
		"Accept-Language",
		"Sec-Fetch-Site",
		"Sec-Fetch-Mode",
		"Sec-Fetch-Dest",
		"Origin",
		"Referer",
	}

	for _, header := range requiredHeaders {
		if req.Header.Get(header) == "" {
			t.Errorf("Expected header %s to be set, but was empty", header)
		}
	}

	// Verify Accept-Encoding is NOT set (Go's http.Client handles it)
	if req.Header.Get("Accept-Encoding") != "" {
		t.Error("Accept-Encoding should NOT be set - Go's http.Client handles compression")
	}

	// Check sec-ch-ua headers specifically (lowercase keys in map)
	secChHeaders := []string{"sec-ch-ua", "sec-ch-ua-mobile", "sec-ch-ua-platform"}
	for _, header := range secChHeaders {
		if _, ok := req.Header[header]; !ok {
			t.Errorf("Expected header %s to be set in map", header)
		}
	}
}

func TestApplyHeadersForAPI_SetsCorrectSecFetchHeaders(t *testing.T) {
	hr := NewHeaderRandomizer()
	req, _ := http.NewRequest("GET", "https://api.llama.fi/oracles", nil)

	hr.ApplyHeadersForAPI(req)

	// Verify Sec-Fetch headers for API/CORS requests
	if got := req.Header.Get("Sec-Fetch-Site"); got != "cross-site" {
		t.Errorf("Sec-Fetch-Site = %q, expected %q", got, "cross-site")
	}
	if got := req.Header.Get("Sec-Fetch-Mode"); got != "cors" {
		t.Errorf("Sec-Fetch-Mode = %q, expected %q", got, "cors")
	}
	if got := req.Header.Get("Sec-Fetch-Dest"); got != "empty" {
		t.Errorf("Sec-Fetch-Dest = %q, expected %q", got, "empty")
	}
}

func TestApplyHeadersForAPI_SetsOriginAndReferer(t *testing.T) {
	hr := NewHeaderRandomizer()
	req, _ := http.NewRequest("GET", "https://api.llama.fi/oracles", nil)

	hr.ApplyHeadersForAPI(req)

	if got := req.Header.Get("Origin"); got != "https://defillama.com" {
		t.Errorf("Origin = %q, expected %q", got, "https://defillama.com")
	}
	if got := req.Header.Get("Referer"); got != "https://defillama.com/" {
		t.Errorf("Referer = %q, expected %q", got, "https://defillama.com/")
	}
}

func TestApplyHeaders_SetsSecChUaHeaders(t *testing.T) {
	hr := NewHeaderRandomizer()
	req, _ := http.NewRequest("GET", "https://example.com", nil)

	hr.ApplyHeaders(req)

	// All UAs are now Chrome, so these should always be set
	// Check directly in map since we use lowercase keys
	secHeaders := []string{
		"sec-ch-ua",
		"sec-ch-ua-mobile",
		"sec-ch-ua-platform",
	}
	for _, header := range secHeaders {
		if _, ok := req.Header[header]; !ok {
			t.Errorf("Expected %s header to be set", header)
		}
	}
}

func TestGenerateSecChUa(t *testing.T) {
	tests := []struct {
		ua       string
		contains string
	}{
		{"Chrome/131.0.0.0", `v="131"`},
		{"Chrome/130.0.0.0", `v="130"`},
		{"Chrome/129.0.0.0", `v="129"`},
		{"Chrome/128.0.0.0", `v="131"`}, // Fallback to 131
	}

	for _, tt := range tests {
		result := generateSecChUa(tt.ua)
		if !strings.Contains(result, tt.contains) {
			t.Errorf("generateSecChUa(%q) = %q, expected to contain %q", tt.ua, result, tt.contains)
		}
	}
}

func TestExtractPlatform(t *testing.T) {
	tests := []struct {
		ua       string
		expected string
	}{
		{"Mozilla/5.0 (Windows NT 10.0; Win64; x64)", `"Windows"`},
		{"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)", `"macOS"`},
		{"Mozilla/5.0 (X11; Linux x86_64)", `"Linux"`},
		{"Mozilla/5.0 (Unknown)", `"Unknown"`},
	}

	for _, tt := range tests {
		result := extractPlatform(tt.ua)
		if result != tt.expected {
			t.Errorf("extractPlatform(%q) = %q, expected %q", tt.ua, result, tt.expected)
		}
	}
}

func TestHeaderRandomizer_Concurrency(t *testing.T) {
	hr := NewHeaderRandomizer()
	done := make(chan bool)

	// Run concurrent header applications
	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 100; j++ {
				req, _ := http.NewRequest("GET", "https://example.com", nil)
				hr.ApplyHeadersForAPI(req)
			}
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestUserAgents_AllValid(t *testing.T) {
	for i, ua := range UserAgents {
		if ua == "" {
			t.Errorf("UserAgents[%d] is empty", i)
		}
		if !strings.Contains(ua, "Mozilla") {
			t.Errorf("UserAgents[%d] doesn't look like a valid UA: %s", i, ua)
		}
		// All should be Chrome UAs for consistent TLS fingerprinting
		if !strings.Contains(ua, "Chrome") {
			t.Errorf("UserAgents[%d] should be Chrome UA for TLS consistency: %s", i, ua)
		}
	}
}

func TestUserAgents_OnlyChromeDesktop(t *testing.T) {
	for i, ua := range UserAgents {
		if strings.Contains(ua, "Mobile") {
			t.Errorf("UserAgents[%d] should not be mobile UA: %s", i, ua)
		}
	}
}

func TestNewBrowserTransport(t *testing.T) {
	transport := NewBrowserTransport()

	if transport == nil {
		t.Fatal("NewBrowserTransport returned nil")
	}

	// Verify it's a RoundTripper
	var _ http.RoundTripper = transport

	// Verify it's our utls round tripper
	rt, ok := transport.(*utlsRoundTripper)
	if !ok {
		t.Fatal("Expected utlsRoundTripper type")
	}

	if rt.h2Transport == nil {
		t.Error("h2Transport should not be nil")
	}

	if rt.h1Transport == nil {
		t.Error("h1Transport should not be nil")
	}
}

func TestApplyHeaders_AcceptsJSON(t *testing.T) {
	hr := NewHeaderRandomizer()
	req, _ := http.NewRequest("GET", "https://api.llama.fi/oracles", nil)

	hr.ApplyHeadersForAPI(req)

	accept := req.Header.Get("Accept")
	if !strings.Contains(accept, "application/json") {
		t.Errorf("Accept header should prefer JSON, got %q", accept)
	}
}
