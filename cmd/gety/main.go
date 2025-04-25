// cmd/gety/main.go
package main

import (
    "bufio"
    "flag"
    "fmt"
    "log"
    "net/http"
    "net/url"
    "os"
    "strconv"
    "strings"
    "sync"
    "time"
    "crypto/tls"
)

func main() {
    // HTTP method flags
    fGET := flag.Bool("GET", false, "Use GET method")
    fPOST := flag.Bool("POST", false, "Use POST method")
    fHEAD := flag.Bool("HEAD", false, "Use HEAD method")
    fPUT := flag.Bool("PUT", false, "Use PUT method")
    // Proxy flag
    proxyAddr := flag.String("proxy", "", "Proxy URL (e.g. http://127.0.0.1:8080)")
    // Rate-limit (seconds between requests to the same host)
    rl := flag.Int("rl", 0, "Rate limit in seconds between requests to the same host")
    // Filter codes (comma-separated list)
    fc := flag.String("fc", "", "Comma-separated HTTP status codes to include (e.g., 200,403)")
    // Insecure TLS (skip certificate verification)
    insecure := flag.Bool("insecure", false, "Disable TLS certificate verification")

    flag.Parse()

    // Determine HTTP method
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

    // Parse filter codes into a set
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

    // HTTP transport with proxy & optional TLS config
    transport := &http.Transport{}
    if *insecure {
        transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
    }
    if *proxyAddr != "" {
        pu, err := url.Parse(*proxyAddr)
        if err != nil {
            log.Fatalf("Invalid proxy URL: %v", err)
        }
        transport.Proxy = http.ProxyURL(pu)
    }
    client := &http.Client{Transport: transport}

    // Track last request time per host
    var mu sync.Mutex
    lastRequest := make(map[string]time.Time)

    scanner := bufio.NewScanner(os.Stdin)
    var wg sync.WaitGroup

    for scanner.Scan() {
        u := strings.TrimSpace(scanner.Text())
        if u == "" {
            continue
        }

        wg.Add(1)
        go func(rawURL string) {
            defer wg.Done()

            // Parse URL to get host
            reqURL, err := url.Parse(rawURL)
            if err != nil {
                fmt.Fprintf(os.Stderr, "Error parsing URL %s: %v\n", rawURL, err)
                return
            }
            host := reqURL.Host

            // Rate-limit per host
            if *rl > 0 {
                mu.Lock()
                if t, ok := lastRequest[host]; ok {
                    elapsed := time.Since(t)
                    wait := time.Duration(*rl)*time.Second - elapsed
                    if wait > 0 {
                        time.Sleep(wait)
                    }
                }
                lastRequest[host] = time.Now()
                mu.Unlock()
            }

            // Build request
            req, err := http.NewRequest(method, rawURL, nil)
            if err != nil {
                fmt.Fprintf(os.Stderr, "Error creating request for %s: %v\n", rawURL, err)
                return
            }

            // Send it
            resp, err := client.Do(req)
            if err != nil {
                fmt.Fprintf(os.Stderr, "Request error %s %s: %v\n", method, rawURL, err)
                return
            }
            defer resp.Body.Close()

            // Filter by status code if requested
            if len(filterCodes) > 0 {
                if !filterCodes[resp.StatusCode] {
                    return
                }
            }

            // Print method, URL, status code and text
            fmt.Printf("%s %s -> %d %s\n", method, rawURL, resp.StatusCode, resp.Status)
        }(u)
    }

    if err := scanner.Err(); err != nil {
        log.Fatalf("Error reading input: %v\n", err)
    }

    wg.Wait()
}
