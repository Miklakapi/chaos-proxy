# Chaos Proxy

A small HTTP reverse proxy written in Go.

Chaos Proxy randomly applies simple network problems to HTTP traffic based on a YAML config.
It is a hobby project made for testing how applications behave when the network is slow, unstable, or interrupted.

The goal is not to cover every edge case.
The goal is to have a small local tool that is easy to run, easy to configure, and useful during development.

## General info

Chaos Proxy sits between a client and a target server:

```text
client -> chaos-proxy -> target server
```

It forwards HTTP requests to the configured target and can randomly apply selected chaos effects.

## Features

* HTTP reverse proxy
* YAML configuration
* Request latency
* Response latency
* Request connection failure
* Response connection failure
* Request bandwidth limit
* Response bandwidth limit
* Basic request logs
* Graceful shutdown

## Configuration

Example `config.yaml`:

```yaml
server:
  listen: ":9090"
  read_timeout: 30s
  write_timeout: 30s
  idle_timeout: 60s

proxy:
  target: "http://localhost:8000"
  preserve_host: false
  add_headers:
    X-Chaos-Proxy: "true"

chaos:
  enable: true

  latency:
    enable: true

    request:
      enable: true
      probability: 0.25
      min: 200ms
      max: 2000ms

    response:
      enable: true
      probability: 0.25
      min: 200ms
      max: 3000ms

  connection_failure:
    enable: true

    request:
      enable: true
      probability: 0.03

    response:
      enable: true
      probability: 0.03
      after_bytes_min: 1024
      after_bytes_max: 65536

  bandwidth_limit:
    enable: true

    request:
      enable: false
      probability: 0.10
      bytes_per_second_min: 10240
      bytes_per_second_max: 20480

    response:
      enable: true
      probability: 0.15
      bytes_per_second_min: 10240
      bytes_per_second_max: 40960
```

## Usage

Run with the default config path:

```bash
go run ./cmd/chaos-proxy
```

Run with a custom config file:

```bash
go run ./cmd/chaos-proxy --config config.yaml
```

Then send traffic to the proxy instead of the original server:

```text
http://localhost:9090
```

The proxy will forward requests to the configured target.

## Chaos effects

### Latency

Adds random delay before forwarding a request or before returning a response.

Useful for testing slow networks, slow gateways, and frontend loading states.

### Connection failure

Closes the connection before the request reaches the target or while the response body is being sent.

Useful for testing retry logic and interrupted network connections.

### Bandwidth limit

Slows down request or response body transfer.

Useful for testing slow uploads, slow downloads, and large responses.

## Logs

The application writes one log line per request.

Example:

```text
request method=GET status=proxied duration=284ms path=/api/users chaos=latency.response
```

Requests affected by chaos are highlighted in the terminal.

## Status

Small hobby project.

Implemented for local development and manual testing, not as a production-grade proxy.

