# gety
Commandline tool for sending HTTP requests (GET, POST, HEAD, PUT, etc.) through a configurable proxy like Burp Suite.

## Installation
Requires `Go 1.18` or newer.

```
go install github.com/<github‑user>/<repo>/cmd/gety@latest
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
| Flag         | Description                                  |
|--------------|----------------------------------------------|
| `-GET`  | Use HTTP GET method.              |
| `-POST`     | Use HTTP POST method.      |
| `-HEAD`   | Use HTTP HEAD method.   |
| `-PUT`    | Use HTTP PUT method.         |
| `-proxy`    | (Required) Proxy URL to route requests through (e.g. Burp Suite)              |
