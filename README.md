# gety
Commandline tool for sending HTTP requests (GET, POST, HEAD, PUT, etc.) through a configurable proxy like Burp Suite.

## Installation
Requires `Go 1.21` or newer.

```
go install github.com/G0urmetD/gety/cmd/gety@latest
```

## Usage
- Send GET requests through Burp Suite
  ```
  cat urls.txt | gety -GET -proxy http://127.0.0.1:8080
  ```
- Send POST requests through Burp Suite
  ```
  cat urls.txt | gety -POST -proxy http://127.0.0.1:8080
  ```
- Send HEAD requests through Burp Suite
  ```
  cat urls.txt | gety -HEAD -proxy http://127.0.0.1:8080
  ```
  
## Parameters
| Flag                          | Description                                                                                         |
|-------------------------------|-----------------------------------------------------------------------------------------------------|
| `-version`                    | Print version information and exit.                                                                 |
| `-GET`                        | Use GET method.                                                                                     |
| `-POST`                       | Use POST method.                                                                                    |
| `-HEAD`                       | Use HEAD method.                                                                                    |
| `-PUT`                        | Use PUT method.                                                                                     |
| `-proxy <url>`                | Proxy URL (e.g. `http://127.0.0.1:8080`).                                                           |
| `-timeout <duration>`         | Request timeout (e.g. `10s`, `1m`).                                                                 |
| `-no-follow`                  | Disable redirect following.                                                                         |
| `-insecure`                   | Disable TLS certificate verification.                                                               |
| `-H "Header: Value"`          | Custom HTTP header (can be repeated).                                                               |
| `-cookie "name=value"`        | Send cookie on every request (can be repeated).                                                     |
| `-c <int>`                    | Maximum number of concurrent requests.                                                              |
| `-rl <seconds>`               | Rate-limit between requests to the same host, in seconds, with jitter.                              |
| `-burst <count>`              | Number of requests in a burst before a cooldown.                                                    |
| `-burst-cooldown <seconds>`   | Cooldown duration after each burst, in seconds.                                                     |
| `-fc <codes>`                 | Comma-separated HTTP status codes to include (e.g., `200,403`).                                      |
| `-match <regex>`              | Only print responses whose body matches the provided regex.                                         |
