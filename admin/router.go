package admin

import (
	"encoding/json"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prasenjit-net/mcp-gateway/config"
	"github.com/prasenjit-net/mcp-gateway/mcp"
	"github.com/prasenjit-net/mcp-gateway/registry"
	"github.com/prasenjit-net/mcp-gateway/store"
)

type Deps struct {
	Store    store.Store
	Registry *registry.Registry
	SSE      *mcp.SSEServer
	HTTP     *mcp.HTTPTransport
	Config   *config.Config
}

func RegisterRoutes(mux *http.ServeMux, deps *Deps) {
	cors := newCORSMiddleware(deps.Config)
	auth := newAuthMiddleware(deps.Config)

	// ── Public MCP protocol endpoints — not protected by admin auth ──────────
	if deps.SSE != nil {
		mux.HandleFunc("GET /mcp/sse", cors(deps.SSE.HandleSSE))
		mux.HandleFunc("POST /mcp/sse/message", cors(deps.SSE.HandleMessage))
	}
	if deps.HTTP != nil {
		mux.HandleFunc("POST /mcp/http", cors(deps.HTTP.Handle))
	}

	// ── Prometheus metrics — public ──────────────────────────────────────────
	mux.Handle("GET /metrics", promhttp.Handler())

	// ── Admin auth routes — public ───────────────────────────────────────────
	loginHdlr := &loginHandler{config: deps.Config}
	mux.HandleFunc("GET /_auth/login", loginHdlr.ServeLogin)
	mux.HandleFunc("POST /_auth/login", loginHdlr.ServeLogin)
	mux.HandleFunc("POST /_auth/logout", loginHdlr.ServeLogout)

	// ── Protected API routes ─────────────────────────────────────────────────
	specHandler := &specsHandler{store: deps.Store, registry: deps.Registry, config: deps.Config}
	mux.HandleFunc("POST /_api/specs", cors(auth(specHandler.Create)))
	mux.HandleFunc("GET /_api/specs", cors(auth(specHandler.List)))
	mux.HandleFunc("GET /_api/specs/{id}", cors(auth(specHandler.Get)))
	mux.HandleFunc("PATCH /_api/specs/{id}", cors(auth(specHandler.Update)))
	mux.HandleFunc("DELETE /_api/specs/{id}", cors(auth(specHandler.Delete)))
	mux.HandleFunc("GET /_api/specs/{id}/operations", cors(auth(specHandler.ListOperations)))

	opsHdlr := &opsHandler{store: deps.Store, registry: deps.Registry, config: deps.Config}
	mux.HandleFunc("PATCH /_api/specs/{id}/operations/{opId}", cors(auth(opsHdlr.UpdateOperation)))

	statsHdlr := &statsHandler{store: deps.Store, registry: deps.Registry, sse: deps.SSE}
	mux.HandleFunc("GET /_api/stats", cors(auth(statsHdlr.Stats)))
	mux.HandleFunc("GET /_api/stats/tools", cors(auth(statsHdlr.ToolStats)))
	mux.HandleFunc("GET /_api/health", cors(statsHdlr.Health)) // health is public

	chatHdlr := &chatHandler{config: deps.Config}
	mux.HandleFunc("GET /_api/chat/config", cors(auth(chatHdlr.GetConfig)))
	mux.HandleFunc("POST /_api/chat/completions", cors(auth(chatHdlr.Completions)))

	resourcesHdlr := &resourcesHandler{store: deps.Store, registry: deps.Registry, config: deps.Config}
	mux.HandleFunc("POST /_api/resources", cors(auth(resourcesHdlr.Create)))
	mux.HandleFunc("GET /_api/resources", cors(auth(resourcesHdlr.List)))
	mux.HandleFunc("GET /_api/resources/{id}", cors(auth(resourcesHdlr.Get)))
	mux.HandleFunc("PATCH /_api/resources/{id}", cors(auth(resourcesHdlr.Update)))
	mux.HandleFunc("DELETE /_api/resources/{id}", cors(auth(resourcesHdlr.Delete)))
	mux.HandleFunc("GET /_api/resources/{id}/content", cors(auth(resourcesHdlr.GetContent)))

	// CORS preflight for all routes.
	mux.HandleFunc("OPTIONS /", newCORSPreflightHandler(deps.Config))
}

func jsonError(w http.ResponseWriter, msg string, code int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	w.Write([]byte(`{"error":` + jsonString(msg) + `}`)) //nolint:errcheck
}

// jsonString safely encodes a string as a JSON string literal.
func jsonString(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

