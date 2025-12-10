# Rate Limit Bypass Guide for Go

## Free Proxy Services

| Service | API Endpoint | Format | Protocol | Notes |
|---------|-------------|--------|----------|-------|
| **ProxyScrape** | `https://api.proxyscrape.com/v2/?request=displayproxies&protocol=http&timeout=10000&country=all&ssl=all&anonymity=elite` | Plain text (ip:port) | HTTP, SOCKS4, SOCKS5 | Best for bulk lists, ~5000+ proxies |
| **GimmeProxy** | `https://gimmeproxy.com/api/getProxy` | JSON | HTTP/HTTPS | Returns 1 proxy per request |
| **GetProxyList** | `https://api.getproxylist.com/proxy` | JSON | HTTP | Rotating proxy, good filtering |
| **PubProxy** | `http://pubproxy.com/api/proxy?limit=5&format=json&type=http` | JSON | HTTP | Up to 5 per request |
| **Proxifly** | `https://raw.githubusercontent.com/proxifly/free-proxy-list/main/proxies/protocols/http/data.txt` | Plain text | HTTP, SOCKS4, SOCKS5 | Updated every 5 min on GitHub |

### Quick API Examples

```bash
# ProxyScrape - Get HTTP proxies
curl "https://api.proxyscrape.com/v2/?request=displayproxies&protocol=http&timeout=10000&anonymity=elite"

# ProxyScrape - Get SOCKS5 proxies  
curl "https://api.proxyscrape.com/v2/?request=displayproxies&protocol=socks5&timeout=10000"

# GimmeProxy - Get single proxy (JSON)
curl "https://gimmeproxy.com/api/getProxy?get=true&anonymityLevel=1&protocol=http"

# PubProxy - Get 5 proxies (JSON)
curl "http://pubproxy.com/api/proxy?limit=5&format=json&type=http&level=anonymous"

# Proxifly GitHub - Raw list
curl "https://raw.githubusercontent.com/proxifly/free-proxy-list/main/proxies/protocols/http/data.txt"
```

---

## Quick Start Code Snippets

### 1. Minimal Proxy Rotation

```go
package main

import (
    "fmt"
    "io"
    "net/http"
    "net/url"
    "strings"
    "sync"
)

type ProxyPool struct {
    proxies []string
    index   int
    mu      sync.Mutex
}

func (p *ProxyPool) LoadFromProxyScrape() error {
    resp, _ := http.Get("https://api.proxyscrape.com/v2/?request=displayproxies&protocol=http&timeout=10000&anonymity=elite")
    body, _ := io.ReadAll(resp.Body)
    defer resp.Body.Close()
    
    for _, line := range strings.Split(string(body), "\n") {
        if line = strings.TrimSpace(line); line != "" {
            p.proxies = append(p.proxies, "http://"+line)
        }
    }
    return nil
}

func (p *ProxyPool) Next() string {
    p.mu.Lock()
    defer p.mu.Unlock()
    proxy := p.proxies[p.index%len(p.proxies)]
    p.index++
    return proxy
}

func (p *ProxyPool) Client() *http.Client {
    proxyURL, _ := url.Parse(p.Next())
    return &http.Client{
        Transport: &http.Transport{Proxy: http.ProxyURL(proxyURL)},
    }
}
```

### 2. User-Agent Rotation

```go
package main

import (
    "math/rand"
    "net/http"
)

var userAgents = []string{
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
    "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:133.0) Gecko/20100101 Firefox/133.0",
    "Mozilla/5.0 (Macintosh; Intel Mac OS X 14_0) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.0 Safari/605.1.15",
    "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36",
}

func RandomUA() string {
    return userAgents[rand.Intn(len(userAgents))]
}

func MakeRequest(url string) (*http.Response, error) {
    req, _ := http.NewRequest("GET", url, nil)
    req.Header.Set("User-Agent", RandomUA())
    return http.DefaultClient.Do(req)
}
```

### 3. Full Header Randomization

```go
package main

import (
    "math/rand"
    "net/http"
    "strings"
)

func RandomHeaders(req *http.Request) {
    ua := RandomUA()
    req.Header.Set("User-Agent", ua)
    
    // Accept headers
    accepts := []string{
        "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8",
        "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8",
        "*/*",
    }
    req.Header.Set("Accept", accepts[rand.Intn(len(accepts))])
    
    // Accept-Language
    langs := []string{"en-US,en;q=0.9", "en-GB,en;q=0.9", "en-US,en;q=0.9,es;q=0.8"}
    req.Header.Set("Accept-Language", langs[rand.Intn(len(langs))])
    
    // Accept-Encoding
    req.Header.Set("Accept-Encoding", "gzip, deflate, br")
    
    // Connection
    req.Header.Set("Connection", "keep-alive")
    
    // Chrome-specific headers
    if strings.Contains(ua, "Chrome") {
        req.Header.Set("Sec-Fetch-Dest", "document")
        req.Header.Set("Sec-Fetch-Mode", "navigate")
        req.Header.Set("Sec-Fetch-Site", "none")
        req.Header.Set("Sec-Fetch-User", "?1")
        req.Header.Set("Sec-Ch-Ua", `"Google Chrome";v="131", "Chromium";v="131"`)
        req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
        req.Header.Set("Sec-Ch-Ua-Platform", `"Windows"`)
    }
    
    // Random optional headers
    if rand.Float32() > 0.5 {
        req.Header.Set("Upgrade-Insecure-Requests", "1")
    }
    if rand.Float32() > 0.7 {
        req.Header.Set("DNT", "1")
    }
}
```

### 4. Combined: All Three Techniques

```go
package main

import (
    "io"
    "math/rand"
    "net/http"
    "net/url"
    "time"
)

func main() {
    rand.Seed(time.Now().UnixNano())
    
    // Load proxies
    pool := &ProxyPool{}
    pool.LoadFromProxyScrape()
    
    // Make request
    targetURL := "https://httpbin.org/ip"
    
    req, _ := http.NewRequest("GET", targetURL, nil)
    RandomHeaders(req)  // Apply randomized headers
    
    client := pool.Client()  // Get client with next proxy
    client.Timeout = 30 * time.Second
    
    resp, err := client.Do(req)
    if err != nil {
        // Try next proxy on failure
        client = pool.Client()
        resp, _ = client.Do(req)
    }
    
    body, _ := io.ReadAll(resp.Body)
    println(string(body))
}
```

---

## Best Practices

1. **Add random delays** between requests (1-5 seconds + jitter)
2. **Mark dead proxies** and remove them from rotation
3. **Match headers to User-Agent** (Chrome UA → Chrome headers)
4. **Respect Retry-After** headers when receiving 429s
5. **Test proxies** before using them in production
6. **Rotate everything together** - proxy + UA + headers for each request

## Warnings

- Free proxies are **unreliable** and **slow** - expect 50%+ failure rates
- Many free proxies are **transparent** (they expose your real IP)
- Some may be **honeypots** - never send sensitive data through free proxies
- For production use, consider paid services like BrightData, Oxylabs, or ScraperAPI
