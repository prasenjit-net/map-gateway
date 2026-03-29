# MCP Gateway

A self-hosted HTTP server that converts any OpenAPI 3 REST API into MCP (Model Context Protocol) tools — no code required. Upload a spec, configure the upstream, and AI agents can immediately call your API through a standard MCP interface.

## Features

- **Auto-tool generation** — every OpenAPI 3 operation becomes an MCP tool
- **SSE + HTTP transports** — `/mcp/sse` for stateful agents, `/mcp/http` for stateless
- **Auth passthrough** — `Authorization` headers from agents forwarded to the backend
- **Configurable auth** — per-spec api-key / bearer / basic / oauth2-client-credentials
- **Multiple specs** — upload as many APIs as you like; all tools merged into one endpoint
- **Admin UI** — React dashboard at `/_ui/` for spec management, enable/disable operations
- **Built-in chat client** — test your tools interactively with any OpenAI-compatible model
- **Prometheus metrics** — `/metrics` endpoint for observability
- **Single binary** — UI embedded; only dependency is a writable data directory

---

## Quick Start

### Local (Go)

```bash
# Requires Go 1.22+
git clone <repo>
cd mcp-gateway

# Build UI first
cd ui && npm ci && npm run build && cd ..

# Run
GATEWAY_SECRET=my-secret go run .
# Server starts at http://localhost:8080
# Admin UI: http://localhost:8080/_ui/
```

### Docker

```bash
docker compose up
# Admin UI: http://localhost:8080/_ui/
# Mock upstream at http://localhost:8081
```

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `LISTEN_ADDR` | `:8080` | HTTP listen address |
| `DATA_DIR` | `./data` | Directory for JSON state files |
| `LOG_LEVEL` | `info` | `debug` / `info` / `warn` / `error` |
| `MAX_RESPONSE_BYTES` | `1048576` | Max upstream response size (bytes) |
| `GATEWAY_SECRET` | *(required)* | Key for encrypting stored credentials |
| `UI_DEV_PROXY` | *(unset)* | Proxy `/_ui/` to Vite dev server (e.g. `http://localhost:5173`) |

---

## Usage

### 1. Upload a Spec

Open `http://localhost:8080/_ui/specs`, click **Upload New Spec**, and provide:
- **Name** — display name for this API
- **Upstream Base URL** — where the real REST API is running
- **Spec file** — OpenAPI 3 YAML or JSON
- **Auth** — none / api-key / bearer / basic / oauth2
- **Passthrough** — whether to forward the agent's `Authorization` header to the backend

### 2. Connect an Agent

**SSE transport** (for stateful agents / Claude Desktop):
```
GET http://localhost:8080/mcp/sse
```

**HTTP transport** (stateless JSON-RPC):
```
POST http://localhost:8080/mcp/http
Content-Type: application/json

{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}
```

#### Claude Desktop config example
```json
{
  "mcpServers": {
    "my-api": {
      "url": "http://localhost:8080/mcp/sse",
      "transport": "sse"
    }
  }
}
```

### 3. Test in the Browser

Go to `http://localhost:8080/_ui/chat`, enter your OpenAI API key, and start chatting. The assistant will automatically use your registered MCP tools.

---

## Admin API

All endpoints under `/_api/`:

```
POST   /_api/specs                          Upload new spec
GET    /_api/specs                          List all specs
GET    /_api/specs/:id                      Get spec detail
PATCH  /_api/specs/:id                      Update spec config
DELETE /_api/specs/:id                      Delete spec
GET    /_api/specs/:id/operations           List operations
PATCH  /_api/specs/:id/operations/:opId     Enable/disable operation
GET    /_api/stats                          Global stats
GET    /_api/stats/tools                    Per-tool call stats
GET    /_api/health                         Health check
GET    /metrics                             Prometheus metrics
```

---

## Auth Passthrough

When an AI agent connects with an `Authorization` header, the gateway forwards it to every upstream call in that session:

```
Agent → GET /mcp/sse
        Authorization: Bearer <user-token>
            ↓
Gateway → GET https://api.example.com/resource
          Authorization: Bearer <user-token>   ← forwarded automatically
```

This lets agents act on behalf of authenticated users without the gateway managing those credentials.

---

## Development

```bash
# Terminal 1: Run Go backend
GATEWAY_SECRET=dev go run .

# Terminal 2: Run Vite dev server (UI hot-reload)
cd ui && UI_DEV_PROXY=http://localhost:5173 npm run dev
```

Or set `UI_DEV_PROXY=http://localhost:5173` on the Go server and Vite will proxy automatically.

---

## Data Storage

All state is stored as JSON files under `DATA_DIR`:

```
data/
├── specs/          {id}.json — spec metadata + raw spec
├── operations/     {spec-id}.json — array of operations
├── auth/           {spec-id}.json — AES-GCM encrypted auth config
└── stats/          tool_stats.json — call counts and latencies
```

Files are written atomically (temp file + rename). Easy to back up, inspect, or migrate.
