//cmd/gety/main.go
package main

import (
    "bufio"
    "flag"
    "fmt"
    "log"
    "net/http"
    "net/url"
    "os"
    "sync"
)

func main() {
    // Flags for HTTP methods
    fGET := flag.Bool("GET", false, "Use GET method")
    fPOST := flag.Bool("POST", false, "Use POST method")
    fHEAD := flag.Bool("HEAD", false, "Use HEAD method")
    fPUT := flag.Bool("PUT", false, "Use PUT method")

    // Proxy flag
    proxyAddr := flag.String("proxy", "", "Proxy URL (e.g. http://127.0.0.1:8080)")

    flag.Parse()

    // Determine which method to use
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

    // Create HTTP client, optionally using the provided proxy
    transport := &http.Transport{}
    if *proxyAddr != "" {
        proxyURL, err := url.Parse(*proxyAddr)
        if err != nil {
            log.Fatalf("Invalid proxy URL: %v\n", err)
        }
        transport.Proxy = http.ProxyURL(proxyURL)
    }
    client := &http.Client{Transport: transport}

    // Read URLs from stdin, one per line
    scanner := bufio.NewScanner(os.Stdin)
    var wg sync.WaitGroup

    for scanner.Scan() {
        rawURL := scanner.Text()
        if rawURL == "" {
            continue
        }

        wg.Add(1)
        go func(u string) {
            defer wg.Done()

            // Build the request
            req, err := http.NewRequest(method, u, nil)
            if err != nil {
                fmt.Fprintf(os.Stderr, "Error creating request for %s: %v\n", u, err)
                return
            }

            // Send the request
            resp, err := client.Do(req)
            if err != nil {
                fmt.Fprintf(os.Stderr, "Request error %s %s: %v\n", method, u, err)
                return
            }
            defer resp.Body.Close()

            // Print method, URL, status code and status text
            fmt.Printf("%s %s -> %d %s\n", method, u, resp.StatusCode, resp.Status)
        }(rawURL)
    }

    if err := scanner.Err(); err != nil {
        log.Fatalf("Error reading input: %v\n", err)
    }

    wg.Wait()
}
