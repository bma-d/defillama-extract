// =============================================================================
// COMPREHENSIVE GUIDE: Rate Limit Circumvention in Go
// Techniques: Proxy Rotation, User-Agent Rotation, Header Randomization
// =============================================================================

package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// =============================================================================
// PART 1: FREE PROXY SERVICES
// =============================================================================
//
// List of Free Proxy API Services (with endpoints):
//
// 1. ProxyScrape (Best for bulk lists)
//    - HTTP:   https://api.proxyscrape.com/v2/?request=displayproxies&protocol=http&timeout=10000&country=all&ssl=all&anonymity=all
//    - SOCKS4: https://api.proxyscrape.com/v2/?request=displayproxies&protocol=socks4&timeout=10000&country=all
//    - SOCKS5: https://api.proxyscrape.com/v2/?request=displayproxies&protocol=socks5&timeout=10000&country=all
//
// 2. GimmeProxy (Single proxy per request - JSON API)
//    - Endpoint: https://gimmeproxy.com/api/getProxy
//    - Params: ?get=true&anonymityLevel=1&protocol=http
//
// 3. GetProxyList (REST API with filtering)
//    - Endpoint: https://api.getproxylist.com/proxy
//    - Returns rotating proxy on each request
//
// 4. PubProxy (Simple REST API)
//    - Endpoint: http://pubproxy.com/api/proxy
//    - Params: ?limit=5&format=json&type=http&level=anonymous
//
// 5. Proxifly (GitHub + API)
//    - GitHub: https://github.com/proxifly/free-proxy-list
//    - Raw lists: https://raw.githubusercontent.com/proxifly/free-proxy-list/main/proxies/protocols/http/data.txt
//    - API: https://proxifly.dev/ (requires free API key for higher limits)
//
// 6. Free-Proxy-List.net (Web scraping required)
//    - URL: https://free-proxy-list.net/
//
// =============================================================================

// =============================================================================
// PART 2: USER-AGENT STRINGS (Updated 2024-2025)
// =============================================================================

// UserAgents contains a comprehensive list of realistic browser user agents
var UserAgents = []string{
	// Chrome on Windows (most common)
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 11.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",

	// Chrome on macOS
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_0) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 13_6) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36",

	// Chrome on Linux
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Ubuntu; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Safari/537.36",

	// Firefox on Windows
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:133.0) Gecko/20100101 Firefox/133.0",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:132.0) Gecko/20100101 Firefox/132.0",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:131.0) Gecko/20100101 Firefox/131.0",

	// Firefox on macOS
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:133.0) Gecko/20100101 Firefox/133.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14.0; rv:132.0) Gecko/20100101 Firefox/132.0",

	// Firefox on Linux
	"Mozilla/5.0 (X11; Linux x86_64; rv:133.0) Gecko/20100101 Firefox/133.0",
	"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:132.0) Gecko/20100101 Firefox/132.0",

	// Safari on macOS
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 14_0) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 13_6) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/16.6 Safari/605.1.15",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Safari/605.1.15",

	// Edge on Windows
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36 Edg/131.0.0.0",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Safari/537.36 Edg/130.0.0.0",

	// Mobile - Chrome on Android
	"Mozilla/5.0 (Linux; Android 14; SM-S918B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (Linux; Android 13; Pixel 7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/130.0.0.0 Mobile Safari/537.36",
	"Mozilla/5.0 (Linux; Android 14; SM-G991B) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/129.0.0.0 Mobile Safari/537.36",

	// Mobile - Safari on iOS
	"Mozilla/5.0 (iPhone; CPU iPhone OS 18_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/18.0 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (iPhone; CPU iPhone OS 17_6 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.6 Mobile/15E148 Safari/604.1",
	"Mozilla/5.0 (iPad; CPU OS 17_0 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Mobile/15E148 Safari/604.1",
}

// =============================================================================
// PART 3: PROXY STRUCTURES AND FETCHING
// =============================================================================

// Proxy represents a proxy server with its details
type Proxy struct {
	IP       string
	Port     string
	Protocol string // http, https, socks4, socks5
	Country  string
	Speed    float64
	Working  bool
}

// ProxyRotator manages a pool of proxies with thread-safe rotation
type ProxyRotator struct {
	proxies      []Proxy
	currentIndex int
	mu           sync.Mutex
	lastUpdate   time.Time
}

// NewProxyRotator creates a new proxy rotator
func NewProxyRotator() *ProxyRotator {
	return &ProxyRotator{
		proxies: make([]Proxy, 0),
	}
}

// FetchFromProxyScrape fetches proxies from ProxyScrape API
func (pr *ProxyRotator) FetchFromProxyScrape(protocol string) error {
	urls := map[string]string{
		"http":   "https://api.proxyscrape.com/v2/?request=displayproxies&protocol=http&timeout=10000&country=all&ssl=all&anonymity=elite",
		"socks4": "https://api.proxyscrape.com/v2/?request=displayproxies&protocol=socks4&timeout=10000&country=all",
		"socks5": "https://api.proxyscrape.com/v2/?request=displayproxies&protocol=socks5&timeout=10000&country=all",
	}

	apiURL, ok := urls[protocol]
	if !ok {
		return fmt.Errorf("unsupported protocol: %s", protocol)
	}

	resp, err := http.Get(apiURL)
	if err != nil {
		return fmt.Errorf("failed to fetch proxies: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	lines := strings.Split(string(body), "\n")
	pr.mu.Lock()
	defer pr.mu.Unlock()

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) == 2 {
			pr.proxies = append(pr.proxies, Proxy{
				IP:       parts[0],
				Port:     parts[1],
				Protocol: protocol,
				Working:  true,
			})
		}
	}

	pr.lastUpdate = time.Now()
	return nil
}

// GimmeProxyResponse represents the JSON response from GimmeProxy
type GimmeProxyResponse struct {
	IP             string `json:"ip"`
	Port           string `json:"port"`
	Protocol       string `json:"protocol"`
	Country        string `json:"country"`
	AnonymityLevel int    `json:"anonymityLevel"`
}

// FetchFromGimmeProxy fetches a single proxy from GimmeProxy API
func (pr *ProxyRotator) FetchFromGimmeProxy() error {
	resp, err := http.Get("https://gimmeproxy.com/api/getProxy?get=true&anonymityLevel=1&protocol=http")
	if err != nil {
		return fmt.Errorf("failed to fetch from GimmeProxy: %w", err)
	}
	defer resp.Body.Close()

	var proxyResp GimmeProxyResponse
	if err := json.NewDecoder(resp.Body).Decode(&proxyResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	pr.mu.Lock()
	defer pr.mu.Unlock()

	pr.proxies = append(pr.proxies, Proxy{
		IP:       proxyResp.IP,
		Port:     proxyResp.Port,
		Protocol: proxyResp.Protocol,
		Country:  proxyResp.Country,
		Working:  true,
	})

	return nil
}

// PubProxyResponse represents the JSON response from PubProxy
type PubProxyResponse struct {
	Data []struct {
		IP      string `json:"ip"`
		Port    string `json:"port"`
		Type    string `json:"type"`
		Country string `json:"country"`
		Speed   string `json:"speed"`
	} `json:"data"`
}

// FetchFromPubProxy fetches proxies from PubProxy API
func (pr *ProxyRotator) FetchFromPubProxy(limit int) error {
	url := fmt.Sprintf("http://pubproxy.com/api/proxy?limit=%d&format=json&type=http&level=anonymous", limit)

	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("failed to fetch from PubProxy: %w", err)
	}
	defer resp.Body.Close()

	var proxyResp PubProxyResponse
	if err := json.NewDecoder(resp.Body).Decode(&proxyResp); err != nil {
		return fmt.Errorf("failed to decode response: %w", err)
	}

	pr.mu.Lock()
	defer pr.mu.Unlock()

	for _, p := range proxyResp.Data {
		pr.proxies = append(pr.proxies, Proxy{
			IP:       p.IP,
			Port:     p.Port,
			Protocol: p.Type,
			Country:  p.Country,
			Working:  true,
		})
	}

	return nil
}

// FetchFromProxifly fetches proxies from Proxifly GitHub raw list
func (pr *ProxyRotator) FetchFromProxifly(protocol string) error {
	urls := map[string]string{
		"http":   "https://raw.githubusercontent.com/proxifly/free-proxy-list/main/proxies/protocols/http/data.txt",
		"socks4": "https://raw.githubusercontent.com/proxifly/free-proxy-list/main/proxies/protocols/socks4/data.txt",
		"socks5": "https://raw.githubusercontent.com/proxifly/free-proxy-list/main/proxies/protocols/socks5/data.txt",
	}

	apiURL, ok := urls[protocol]
	if !ok {
		return fmt.Errorf("unsupported protocol: %s", protocol)
	}

	resp, err := http.Get(apiURL)
	if err != nil {
		return fmt.Errorf("failed to fetch from Proxifly: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	lines := strings.Split(string(body), "\n")
	pr.mu.Lock()
	defer pr.mu.Unlock()

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, ":")
		if len(parts) == 2 {
			pr.proxies = append(pr.proxies, Proxy{
				IP:       parts[0],
				Port:     parts[1],
				Protocol: protocol,
				Working:  true,
			})
		}
	}

	return nil
}

// GetNext returns the next proxy in rotation
func (pr *ProxyRotator) GetNext() *Proxy {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if len(pr.proxies) == 0 {
		return nil
	}

	proxy := &pr.proxies[pr.currentIndex]
	pr.currentIndex = (pr.currentIndex + 1) % len(pr.proxies)
	return proxy
}

// GetRandom returns a random proxy from the pool
func (pr *ProxyRotator) GetRandom() *Proxy {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	if len(pr.proxies) == 0 {
		return nil
	}

	return &pr.proxies[rand.Intn(len(pr.proxies))]
}

// MarkDead marks a proxy as non-working
func (pr *ProxyRotator) MarkDead(ip string, port string) {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	for i := range pr.proxies {
		if pr.proxies[i].IP == ip && pr.proxies[i].Port == port {
			pr.proxies[i].Working = false
			break
		}
	}
}

// RemoveDead removes all non-working proxies
func (pr *ProxyRotator) RemoveDead() {
	pr.mu.Lock()
	defer pr.mu.Unlock()

	working := make([]Proxy, 0)
	for _, p := range pr.proxies {
		if p.Working {
			working = append(working, p)
		}
	}
	pr.proxies = working
}

// Count returns the number of proxies in the pool
func (pr *ProxyRotator) Count() int {
	pr.mu.Lock()
	defer pr.mu.Unlock()
	return len(pr.proxies)
}

// =============================================================================
// PART 4: HEADER RANDOMIZATION
// =============================================================================

// HeaderRandomizer generates realistic, randomized HTTP headers
type HeaderRandomizer struct {
	acceptLanguages []string
	acceptEncodings []string
	accepts         []string
	connections     []string
	cacheControls   []string
	secFetchDest    []string
	secFetchMode    []string
	secFetchSite    []string
	secFetchUser    []string
}

// NewHeaderRandomizer creates a new header randomizer with default values
func NewHeaderRandomizer() *HeaderRandomizer {
	return &HeaderRandomizer{
		acceptLanguages: []string{
			"en-US,en;q=0.9",
			"en-US,en;q=0.9,es;q=0.8",
			"en-GB,en;q=0.9,en-US;q=0.8",
			"en-US,en;q=0.9,fr;q=0.8",
			"en-US,en;q=0.9,de;q=0.8",
			"en-CA,en;q=0.9",
			"en-AU,en;q=0.9",
			"en;q=0.9",
		},
		acceptEncodings: []string{
			"gzip, deflate, br",
			"gzip, deflate, br, zstd",
			"gzip, deflate",
			"br, gzip, deflate",
		},
		accepts: []string{
			"text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8",
			"text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
			"text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
			"application/json,text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
			"*/*",
		},
		connections: []string{
			"keep-alive",
			"Keep-Alive",
		},
		cacheControls: []string{
			"no-cache",
			"max-age=0",
			"no-store",
		},
		secFetchDest: []string{
			"document",
			"empty",
		},
		secFetchMode: []string{
			"navigate",
			"cors",
			"no-cors",
		},
		secFetchSite: []string{
			"none",
			"same-origin",
			"same-site",
			"cross-site",
		},
		secFetchUser: []string{
			"?1",
		},
	}
}

// randomChoice returns a random element from a slice
func randomChoice(choices []string) string {
	return choices[rand.Intn(len(choices))]
}

// RandomUserAgent returns a random user agent string
func RandomUserAgent() string {
	return UserAgents[rand.Intn(len(UserAgents))]
}

// GenerateHeaders returns a map of randomized HTTP headers
func (hr *HeaderRandomizer) GenerateHeaders() map[string]string {
	ua := RandomUserAgent()

	headers := map[string]string{
		"User-Agent":      ua,
		"Accept":          randomChoice(hr.accepts),
		"Accept-Language": randomChoice(hr.acceptLanguages),
		"Accept-Encoding": randomChoice(hr.acceptEncodings),
		"Connection":      randomChoice(hr.connections),
		"Cache-Control":   randomChoice(hr.cacheControls),
	}

	// Add Sec-Fetch headers (Chrome/Edge specific - only add for Chrome UAs)
	if strings.Contains(ua, "Chrome") && !strings.Contains(ua, "Mobile") {
		headers["Sec-Fetch-Dest"] = randomChoice(hr.secFetchDest)
		headers["Sec-Fetch-Mode"] = randomChoice(hr.secFetchMode)
		headers["Sec-Fetch-Site"] = randomChoice(hr.secFetchSite)
		headers["Sec-Fetch-User"] = randomChoice(hr.secFetchUser)
		headers["Sec-Ch-Ua"] = generateSecChUa(ua)
		headers["Sec-Ch-Ua-Mobile"] = "?0"
		headers["Sec-Ch-Ua-Platform"] = extractPlatform(ua)
	}

	// Randomly add optional headers
	if rand.Float32() > 0.5 {
		headers["Upgrade-Insecure-Requests"] = "1"
	}
	if rand.Float32() > 0.7 {
		headers["DNT"] = "1"
	}

	return headers
}

// generateSecChUa generates Sec-Ch-Ua header based on Chrome version in UA
func generateSecChUa(ua string) string {
	// Extract Chrome version from UA
	if strings.Contains(ua, "Chrome/131") {
		return `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`
	} else if strings.Contains(ua, "Chrome/130") {
		return `"Google Chrome";v="130", "Chromium";v="130", "Not_A Brand";v="24"`
	} else if strings.Contains(ua, "Chrome/129") {
		return `"Google Chrome";v="129", "Chromium";v="129", "Not_A Brand";v="24"`
	}
	return `"Google Chrome";v="131", "Chromium";v="131", "Not_A Brand";v="24"`
}

// extractPlatform extracts platform from user agent
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

// =============================================================================
// PART 5: SMART HTTP CLIENT
// =============================================================================

// SmartClient combines proxy rotation, user-agent rotation, and header randomization
type SmartClient struct {
	proxyRotator     *ProxyRotator
	headerRandomizer *HeaderRandomizer
	timeout          time.Duration
	retryCount       int
	retryDelay       time.Duration
}

// NewSmartClient creates a new smart HTTP client
func NewSmartClient() *SmartClient {
	return &SmartClient{
		proxyRotator:     NewProxyRotator(),
		headerRandomizer: NewHeaderRandomizer(),
		timeout:          30 * time.Second,
		retryCount:       3,
		retryDelay:       2 * time.Second,
	}
}

// SetTimeout sets the request timeout
func (sc *SmartClient) SetTimeout(d time.Duration) {
	sc.timeout = d
}

// SetRetryCount sets the number of retries
func (sc *SmartClient) SetRetryCount(count int) {
	sc.retryCount = count
}

// LoadProxies loads proxies from multiple sources
func (sc *SmartClient) LoadProxies(sources ...string) error {
	for _, source := range sources {
		var err error
		switch source {
		case "proxyscrape":
			err = sc.proxyRotator.FetchFromProxyScrape("http")
		case "proxyscrape-socks5":
			err = sc.proxyRotator.FetchFromProxyScrape("socks5")
		case "gimmeproxy":
			err = sc.proxyRotator.FetchFromGimmeProxy()
		case "pubproxy":
			err = sc.proxyRotator.FetchFromPubProxy(5)
		case "proxifly":
			err = sc.proxyRotator.FetchFromProxifly("http")
		default:
			continue
		}
		if err != nil {
			fmt.Printf("Warning: failed to load from %s: %v\n", source, err)
		}
	}

	fmt.Printf("Loaded %d proxies total\n", sc.proxyRotator.Count())
	return nil
}

// createHTTPClient creates an http.Client with the given proxy
func (sc *SmartClient) createHTTPClient(proxy *Proxy) *http.Client {
	transport := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		DialContext: (&net.Dialer{
			Timeout:   10 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
	}

	if proxy != nil {
		var proxyURL *url.URL
		var err error

		switch proxy.Protocol {
		case "http", "https":
			proxyURL, err = url.Parse(fmt.Sprintf("http://%s:%s", proxy.IP, proxy.Port))
		case "socks4":
			proxyURL, err = url.Parse(fmt.Sprintf("socks4://%s:%s", proxy.IP, proxy.Port))
		case "socks5":
			proxyURL, err = url.Parse(fmt.Sprintf("socks5://%s:%s", proxy.IP, proxy.Port))
		default:
			proxyURL, err = url.Parse(fmt.Sprintf("http://%s:%s", proxy.IP, proxy.Port))
		}

		if err == nil {
			transport.Proxy = http.ProxyURL(proxyURL)
		}
	}

	return &http.Client{
		Transport: transport,
		Timeout:   sc.timeout,
	}
}

// Do performs an HTTP request with proxy rotation, header randomization, and retry logic
func (sc *SmartClient) Do(method, targetURL string, body io.Reader) (*http.Response, error) {
	var lastErr error

	for attempt := 0; attempt <= sc.retryCount; attempt++ {
		// Get next proxy (can be nil if no proxies loaded - direct connection)
		proxy := sc.proxyRotator.GetNext()

		// Create client
		client := sc.createHTTPClient(proxy)

		// Create request
		req, err := http.NewRequest(method, targetURL, body)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}

		// Apply randomized headers
		headers := sc.headerRandomizer.GenerateHeaders()
		for key, value := range headers {
			req.Header.Set(key, value)
		}

		// Execute request
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			if proxy != nil {
				sc.proxyRotator.MarkDead(proxy.IP, proxy.Port)
				fmt.Printf("Proxy %s:%s failed, marked as dead\n", proxy.IP, proxy.Port)
			}
			time.Sleep(sc.retryDelay)
			continue
		}

		// Check for rate limiting
		if resp.StatusCode == 429 {
			resp.Body.Close()
			retryAfter := resp.Header.Get("Retry-After")
			fmt.Printf("Rate limited (429). Retry-After: %s. Switching proxy...\n", retryAfter)

			if proxy != nil {
				sc.proxyRotator.MarkDead(proxy.IP, proxy.Port)
			}

			// Add jitter to delay
			jitter := time.Duration(rand.Intn(3000)) * time.Millisecond
			time.Sleep(sc.retryDelay + jitter)
			continue
		}

		return resp, nil
	}

	return nil, fmt.Errorf("all retries exhausted: %w", lastErr)
}

// Get performs a GET request
func (sc *SmartClient) Get(targetURL string) (*http.Response, error) {
	return sc.Do("GET", targetURL, nil)
}

// =============================================================================
// PART 6: PARALLEL WORKER POOL
// =============================================================================

// WorkerPool manages concurrent requests with proxy rotation
type WorkerPool struct {
	client     *SmartClient
	numWorkers int
	jobs       chan string
	results    chan *WorkResult
	ctx        context.Context
	cancel     context.CancelFunc
}

// WorkResult represents the result of a single request
type WorkResult struct {
	URL        string
	StatusCode int
	Body       []byte
	Error      error
	Duration   time.Duration
}

// NewWorkerPool creates a new worker pool
func NewWorkerPool(client *SmartClient, numWorkers int) *WorkerPool {
	ctx, cancel := context.WithCancel(context.Background())
	return &WorkerPool{
		client:     client,
		numWorkers: numWorkers,
		jobs:       make(chan string, 100),
		results:    make(chan *WorkResult, 100),
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start starts the worker pool
func (wp *WorkerPool) Start() {
	for i := 0; i < wp.numWorkers; i++ {
		go wp.worker(i)
	}
}

// worker processes jobs from the queue
func (wp *WorkerPool) worker(id int) {
	for {
		select {
		case <-wp.ctx.Done():
			return
		case url, ok := <-wp.jobs:
			if !ok {
				return
			}

			start := time.Now()

			// Add random jitter between requests
			jitter := time.Duration(rand.Intn(2000)) * time.Millisecond
			time.Sleep(jitter)

			resp, err := wp.client.Get(url)
			result := &WorkResult{
				URL:      url,
				Duration: time.Since(start),
			}

			if err != nil {
				result.Error = err
			} else {
				result.StatusCode = resp.StatusCode
				body, _ := io.ReadAll(resp.Body)
				resp.Body.Close()
				result.Body = body
			}

			wp.results <- result
		}
	}
}

// Submit submits a URL to be processed
func (wp *WorkerPool) Submit(url string) {
	wp.jobs <- url
}

// Results returns the results channel
func (wp *WorkerPool) Results() <-chan *WorkResult {
	return wp.results
}

// Stop stops the worker pool
func (wp *WorkerPool) Stop() {
	wp.cancel()
	close(wp.jobs)
}

// =============================================================================
// PART 7: USAGE EXAMPLES
// =============================================================================

func main() {
	// Seed random number generator
	rand.Seed(time.Now().UnixNano())

	fmt.Println("=== Rate Limit Bypass Demo ===")

	// Example 1: Simple request with randomized headers only
	fmt.Println("--- Example 1: Headers Only (No Proxies) ---")
	simpleDemo()

	// Example 2: Full setup with proxies
	fmt.Println("\n--- Example 2: With Proxy Rotation ---")
	proxyDemo()

	// Example 3: Worker pool for bulk requests
	fmt.Println("\n--- Example 3: Worker Pool Demo ---")
	workerPoolDemo()
}

func simpleDemo() {
	client := NewSmartClient()

	// Make 3 requests with randomized headers
	for i := 0; i < 3; i++ {
		resp, err := client.Get("https://httpbin.org/headers")
		if err != nil {
			fmt.Printf("Request %d failed: %v\n", i+1, err)
			continue
		}

		fmt.Printf("Request %d: Status %d\n", i+1, resp.StatusCode)
		resp.Body.Close()

		time.Sleep(500 * time.Millisecond)
	}
}

func proxyDemo() {
	client := NewSmartClient()

	// Load proxies from free services
	// Note: Free proxies are often slow/unreliable - this is for demonstration
	fmt.Println("Loading proxies (this may take a moment)...")

	err := client.LoadProxies("proxyscrape", "proxifly")
	if err != nil {
		fmt.Printf("Error loading proxies: %v\n", err)
	}

	if client.proxyRotator.Count() == 0 {
		fmt.Println("No proxies loaded, falling back to direct connection")
	}

	// Make requests through rotating proxies
	for i := 0; i < 3; i++ {
		resp, err := client.Get("https://httpbin.org/ip")
		if err != nil {
			fmt.Printf("Request %d failed: %v\n", i+1, err)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("Request %d: Status %d, Response: %s\n", i+1, resp.StatusCode, strings.TrimSpace(string(body)))
		resp.Body.Close()
	}
}

func workerPoolDemo() {
	client := NewSmartClient()

	// Create worker pool with 3 concurrent workers
	pool := NewWorkerPool(client, 3)
	pool.Start()

	// Submit URLs
	urls := []string{
		"https://httpbin.org/ip",
		"https://httpbin.org/headers",
		"https://httpbin.org/user-agent",
		"https://httpbin.org/get",
		"https://httpbin.org/status/200",
	}

	go func() {
		for _, url := range urls {
			pool.Submit(url)
		}
	}()

	// Collect results
	for i := 0; i < len(urls); i++ {
		result := <-pool.Results()
		if result.Error != nil {
			fmt.Printf("URL: %s - Error: %v\n", result.URL, result.Error)
		} else {
			fmt.Printf("URL: %s - Status: %d - Duration: %v\n", result.URL, result.StatusCode, result.Duration)
		}
	}

	pool.Stop()
}
