// cmd/gety/main.go
package main

import (
    "bufio"
    "crypto/tls"
    "flag"
    "fmt"
    "io"
    "log"
    "math/rand"
    "net/http"
    "net/http/cookiejar"
    "net/url"
    "os"
    "regexp"
    "strconv"
    "strings"
    "sync"
    "time"
)

const version = "v0.2.0"

func main() {
    // --- CLI flags ---
    fVersion := flag.Bool("version", false, "Print version and exit")
    fGET := flag.Bool("GET", false, "Use GET method")
    fPOST := flag.Bool("POST", false, "Use POST method")
    fHEAD := flag.Bool("HEAD", false, "Use HEAD method")
    fPUT := flag.Bool("PUT", false, "Use PUT method")

    proxyAddr := flag.String("proxy", "", "Proxy URL (e.g. http://127.0.0.1:8080)")
    timeout := flag.Duration("timeout", 30*time.Second, "Request timeout (e.g. 10s)")
    noFollow := flag.Bool("no-follow", false, "Disable redirect following")
    rl := flag.Int("rl", 0, "Rate limit (seconds) between requests to the same host")
    fc := flag.String("fc", "", "Comma-separated HTTP status codes to include (e.g. 200,403)")
    insecure := flag.Bool("insecure", false, "Disable TLS certificate verification")
    concurrency := flag.Int("c", 10, "Maximum number of concurrent requests")
    burst := flag.Int("burst", 0, "Number of requests in a burst before cooling down")
    burstCooldown := flag.Int("burst-cooldown", 0, "Cooldown (seconds) after each burst")
    matchPattern := flag.String("match", "", "Regex to match in response body")

    var headerFlags []string
    flag.Func("H", "Custom header, can be repeated: 'Header: Value'", func(s string) error {
        headerFlags = append(headerFlags, s)
        return nil
    })

    var cookieFlags []string
    flag.Func("cookie", "Cookie to send, can be repeated: 'name=value'", func(s string) error {
        cookieFlags = append(cookieFlags, s)
        return nil
    })

    flag.Parse()

    if *fVersion {
        fmt.Printf("gety %s\n", version)
        return
    }

    // --- Determine HTTP method ---
    var method string
    switch {
    case *fGET:
        method = http.MethodGet
    case *fPOST:
        method = http.MethodPost
    case *fHEAD:
        method = http.MethodHead
    case *fPUT:
        method = http.MethodPut
    default:
        log.Fatal("Please choose exactly one method: -GET, -POST, -HEAD, or -PUT")
    }

    if *proxyAddr == "" {
        log.Fatal("Proxy URL is required: use -proxy")
    }

    // --- Parse status-code filter ---
    filterCodes := make(map[int]bool)
    if *fc != "" {
        for _, part := range strings.Split(*fc, ",") {
            code, err := strconv.Atoi(strings.TrimSpace(part))
            if err != nil {
                log.Fatalf("Invalid status code in -fc: %v", err)
            }
            filterCodes[code] = true
        }
    }

    // --- Compile body-match regex ---
    var bodyRe *regexp.Regexp
    if *matchPattern != "" {
        re, err := regexp.Compile(*matchPattern)
        if err != nil {
            log.Fatalf("Invalid regex in -match: %v", err)
        }
        bodyRe = re
    }

    // --- Seed for jitter ---
    rand.Seed(time.Now().UnixNano())

    // --- HTTP client & transport setup ---
    transport := &http.Transport{}
    if *insecure {
        transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
    }
    pu, err := url.Parse(*proxyAddr)
    if err != nil {
        log.Fatalf("Invalid proxy URL: %v", err)
    }
    transport.Proxy = http.ProxyURL(pu)

    jar, _ := cookiejar.New(nil)
    client := &http.Client{
        Transport:     transport,
        Timeout:       *timeout,
        CheckRedirect: nil,  // use default
        Jar:           jar,
    }
    if *noFollow {
        client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
            return http.ErrUseLastResponse
        }
    }

    // --- Concurrency and burst controls ---
    sem := make(chan struct{}, *concurrency)
    var wg sync.WaitGroup

    burstCount := 0

    // --- Rate-limit state per host ---
    var mu sync.Mutex
    lastRequest := make(map[string]time.Time)

    // --- Read URLs from stdin ---
    scanner := bufio.NewScanner(os.Stdin)
    for scanner.Scan() {
        rawURL := strings.TrimSpace(scanner.Text())
        if rawURL == "" {
            continue
        }

        // Burst logic (in main goroutine to throttle launches)
        if *burst > 0 && *burstCooldown > 0 {
            if burstCount >= *burst {
                fmt.Fprintf(os.Stderr,
                    "🌩  burst of %d reached, cooling down for %ds...\n",
                    *burst, *burstCooldown,
                )
                time.Sleep(time.Duration(*burstCooldown) * time.Second)
                burstCount = 0
            }
            burstCount++
        }

        // Acquire concurrency slot
        sem <- struct{}{}
        wg.Add(1)

        go func(u string) {
            defer wg.Done()
            defer func() { <-sem }()

            // Parse URL to find host for rate-limit & cookies
            reqURL, err := url.Parse(u)
            if err != nil {
                fmt.Fprintf(os.Stderr, "❌ parse error %s: %v\n", u, err)
                return
            }
            host := reqURL.Host

            // Per-host rate-limit with jitter
            if *rl > 0 {
                mu.Lock()
                if prev, ok := lastRequest[host]; ok {
                    elapsed := time.Since(prev)
                    baseWait := time.Duration(*rl)*time.Second - elapsed
                    if baseWait < 0 {
                        baseWait = 0
                    }
                    // jitter ± rl/2
                    jitterMax := time.Duration(*rl)*time.Second / 2
                    jitter := time.Duration(rand.Int63n(int64(jitterMax)*2)) - jitterMax
                    wait := baseWait + jitter
                    if wait < 0 {
                        wait = 0
                    }
                    time.Sleep(wait)
                }
                lastRequest[host] = time.Now()
                mu.Unlock()
            }

            // Build request
            req, err := http.NewRequest(method, u, nil)
            if err != nil {
                fmt.Fprintf(os.Stderr, "❌ request build %s: %v\n", u, err)
                return
            }

            // Add custom headers
            for _, hdr := range headerFlags {
                parts := strings.SplitN(hdr, ":", 2)
                if len(parts) == 2 {
                    req.Header.Add(strings.TrimSpace(parts[0]), strings.TrimSpace(parts[1]))
                }
            }

            // Add initial cookies
            for _, ck := range cookieFlags {
                parts := strings.SplitN(ck, "=", 2)
                if len(parts) == 2 {
                    req.AddCookie(&http.Cookie{
                        Name:  strings.TrimSpace(parts[0]),
                        Value: strings.TrimSpace(parts[1]),
                    })
                }
            }

            // Do the request
            resp, err := client.Do(req)
            if err != nil {
                fmt.Fprintf(os.Stderr, "❌ %s %s: %v\n", method, u, err)
                return
            }
            defer resp.Body.Close()

            // Status-code filter
            if len(filterCodes) > 0 {
                if !filterCodes[resp.StatusCode] {
                    return
                }
            }

            // Body-match filter
            if bodyRe != nil {
                body, err := io.ReadAll(resp.Body)
                if err != nil {
                    fmt.Fprintf(os.Stderr, "❌ read body %s: %v\n", u, err)
                    return
                }
                if !bodyRe.Match(body) {
                    return
                }
            }

            // Output result
            fmt.Printf("%s %s -> %d %s\n", method, u, resp.StatusCode, resp.Status)
        }(rawURL)
    }
    if err := scanner.Err(); err != nil {
        log.Fatalf("Error reading input: %v", err)
    }

    wg.Wait()
}
