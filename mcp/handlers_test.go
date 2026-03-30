package mcp_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/prasenjit-net/mcp-gateway/auth"
	"github.com/prasenjit-net/mcp-gateway/config"
	"github.com/prasenjit-net/mcp-gateway/mcp"
	"github.com/prasenjit-net/mcp-gateway/registry"
	"github.com/prasenjit-net/mcp-gateway/store"
)

func makeReq(method string, id interface{}, params interface{}) *mcp.Request {
	var raw json.RawMessage
	if params != nil {
		b, _ := json.Marshal(params)
		raw = b
	}
	return &mcp.Request{JSONRPC: "2.0", ID: id, Method: method, Params: raw}
}

func buildHandlerDeps(t *testing.T) *mcp.HandlerDeps {
	t.Helper()
	s, err := store.NewJSONStore(t.TempDir())
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { s.Close() }) //nolint:errcheck
	cfg := config.DefaultConfig()
	cfg.DataDir = t.TempDir()
	return &mcp.HandlerDeps{
		Registry: registry.NewRegistry(),
		Store:    s,
		Config:   cfg,
	}
}

// ── initialize ────────────────────────────────────────────────────────────────

func TestHandleInitialize(t *testing.T) {
	h := buildHandlerDeps(t)
	resp := h.Handle(context.Background(), makeReq("initialize", 1, map[string]interface{}{
		"protocolVersion": mcp.MCPVersion,
		"capabilities":    map[string]interface{}{},
		"clientInfo":      map[string]interface{}{"name": "test", "version": "1.0"},
	}), &auth.InboundAuth{})

	if resp == nil {
		t.Fatal("got nil response")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error.Message)
	}
	result, ok := resp.Result.(mcp.InitializeResult)
	if !ok {
		t.Fatalf("expected InitializeResult, got %T", resp.Result)
	}
	if result.ProtocolVersion != mcp.MCPVersion {
		t.Errorf("protocolVersion = %q, want %q", result.ProtocolVersion, mcp.MCPVersion)
	}
	if result.ServerInfo.Name == "" {
		t.Error("ServerInfo.Name should not be empty")
	}
}

// ── initialized (notification) ────────────────────────────────────────────────

func TestHandleInitialized(t *testing.T) {
	h := buildHandlerDeps(t)
	resp := h.Handle(context.Background(), makeReq("initialized", nil, nil), &auth.InboundAuth{})
	if resp != nil {
		t.Errorf("expected nil response for initialized notification, got: %+v", resp)
	}
}

// ── ping ─────────────────────────────────────────────────────────────────────

func TestHandlePing(t *testing.T) {
	h := buildHandlerDeps(t)
	resp := h.Handle(context.Background(), makeReq("ping", 2, nil), &auth.InboundAuth{})
	if resp == nil {
		t.Fatal("got nil response")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error.Message)
	}
}

// ── tools/list ────────────────────────────────────────────────────────────────

func TestHandleToolsListEmpty(t *testing.T) {
	h := buildHandlerDeps(t)
	resp := h.Handle(context.Background(), makeReq("tools/list", 3, nil), &auth.InboundAuth{})
	if resp == nil || resp.Error != nil {
		t.Fatalf("unexpected error or nil: %v", resp)
	}
}

// ── resources/list ────────────────────────────────────────────────────────────

func TestHandleResourcesListEmpty(t *testing.T) {
	h := buildHandlerDeps(t)
	resp := h.Handle(context.Background(), makeReq("resources/list", 4, nil), &auth.InboundAuth{})
	if resp == nil || resp.Error != nil {
		t.Fatalf("unexpected error or nil: %v", resp)
	}
}

// ── resources/templates/list ──────────────────────────────────────────────────

func TestHandleResourceTemplatesListEmpty(t *testing.T) {
	h := buildHandlerDeps(t)
	resp := h.Handle(context.Background(), makeReq("resources/templates/list", 5, nil), &auth.InboundAuth{})
	if resp == nil || resp.Error != nil {
		t.Fatalf("unexpected error or nil: %v", resp)
	}
}

// ── unknown method ────────────────────────────────────────────────────────────

func TestHandleUnknownMethod(t *testing.T) {
	h := buildHandlerDeps(t)
	resp := h.Handle(context.Background(), makeReq("no/such/method", 9, nil), &auth.InboundAuth{})
	if resp == nil {
		t.Fatal("got nil response for unknown method")
	}
	if resp.Error == nil {
		t.Error("expected RPC error for unknown method")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("error code = %d, want -32601", resp.Error.Code)
	}
}

// ── resources/read (text resource) ────────────────────────────────────────────

func TestHandleResourcesReadText(t *testing.T) {
	h := buildHandlerDeps(t)

	// Write the content to a file in DataDir (like the admin createJSON handler does).
	relPath := "resources/test-res-1/content.txt"
	absPath := h.Config.DataDir + "/" + relPath
	if err := os.MkdirAll(h.Config.DataDir+"/resources/test-res-1", 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	if err := os.WriteFile(absPath, []byte("hello world"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	rec := &store.ResourceRecord{
		ID:       "test-res-1",
		Name:     "Test",
		Type:     "text",
		MimeType: "text/plain",
		FilePath: relPath,
	}
	if err := h.Store.SaveResource(rec); err != nil {
		t.Fatalf("SaveResource: %v", err)
	}
	h.Registry.RebuildResources([]*store.ResourceRecord{rec})

	resp := h.Handle(context.Background(), makeReq("resources/read", 6, map[string]string{
		"uri": "gateway://resources/test-res-1",
	}), &auth.InboundAuth{})

	if resp == nil {
		t.Fatal("got nil response")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error.Message)
	}
}

// ── tools/call (unknown tool) ─────────────────────────────────────────────────

func TestHandleToolsCallUnknownTool(t *testing.T) {
	h := buildHandlerDeps(t)
	resp := h.Handle(context.Background(), makeReq("tools/call", 7, map[string]interface{}{
		"name":      "nonexistent_tool",
		"arguments": map[string]interface{}{},
	}), &auth.InboundAuth{})

	if resp == nil {
		t.Fatal("got nil response")
	}
	// Should return an error or isError content for unknown tool.
	if resp.Error == nil && resp.Result == nil {
		t.Error("expected either error or result for unknown tool")
	}
}

// ── resources/read error paths ────────────────────────────────────────────────

func TestHandleResourcesReadUnsupportedURI(t *testing.T) {
	h := buildHandlerDeps(t)
	resp := h.Handle(context.Background(), makeReq("resources/read", 8, map[string]string{
		"uri": "http://example.com/not-gateway",
	}), &auth.InboundAuth{})
	if resp == nil || resp.Error == nil {
		t.Error("expected RPC error for unsupported URI")
	}
}

func TestHandleResourcesReadNotFound(t *testing.T) {
	h := buildHandlerDeps(t)
	resp := h.Handle(context.Background(), makeReq("resources/read", 9, map[string]string{
		"uri": "gateway://resources/no-such-id",
	}), &auth.InboundAuth{})
	if resp == nil || resp.Error == nil {
		t.Error("expected RPC error for not-found resource")
	}
}

// ── HTTPTransport ─────────────────────────────────────────────────────────────

func TestHTTPTransportSingleRequest(t *testing.T) {
	h := buildHandlerDeps(t)
	transport := mcp.NewHTTPTransport(h)

	body, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "ping",
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/mcp/http", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	transport.Handle(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("transport = %d, want 200", rec.Code)
	}
}

func TestHTTPTransportBatchRequest(t *testing.T) {
	h := buildHandlerDeps(t)
	transport := mcp.NewHTTPTransport(h)

	batch := []map[string]interface{}{
		{"jsonrpc": "2.0", "id": 1, "method": "ping"},
		{"jsonrpc": "2.0", "id": 2, "method": "tools/list"},
	}
	body, _ := json.Marshal(batch)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/mcp/http", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	transport.Handle(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("batch transport = %d, want 200", rec.Code)
	}
	var responses []interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &responses); err != nil {
		t.Fatalf("batch response not valid JSON array: %v", err)
	}
	if len(responses) < 2 {
		t.Errorf("expected 2 batch responses, got %d", len(responses))
	}
}

func TestHTTPTransportInvalidJSON(t *testing.T) {
	h := buildHandlerDeps(t)
	transport := mcp.NewHTTPTransport(h)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/mcp/http", bytes.NewReader([]byte("not json")))
	transport.Handle(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("invalid JSON = %d, want 400", rec.Code)
	}
}

func TestHTTPTransportNotification(t *testing.T) {
	h := buildHandlerDeps(t)
	transport := mcp.NewHTTPTransport(h)

	// "initialized" has no ID and returns nil (notification) → 204
	body, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "initialized",
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/mcp/http", bytes.NewReader(body))
	transport.Handle(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("notification = %d, want 204", rec.Code)
	}
}

