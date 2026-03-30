package admin_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/prasenjit-net/mcp-gateway/admin"
	"github.com/prasenjit-net/mcp-gateway/registry"
	"github.com/prasenjit-net/mcp-gateway/store"
)

// petStoreSpecJSON is a minimal OpenAPI 3.0 spec used across handler tests.
const petStoreSpecJSON = `{"openapi":"3.0.0","info":{"title":"Pet Store","version":"1.0.0"},"paths":{"/pets":{"get":{"operationId":"listPets","summary":"List pets","responses":{"200":{"description":"ok"}}}}}}`

// newMux creates a mux with all admin routes registered and auth disabled.
func newMux(t *testing.T) *http.ServeMux {
	t.Helper()
	cfg := adminCfg("", "secret") // empty password → no auth
	cfg.DataDir = t.TempDir()     // isolate test data
	s, err := store.NewJSONStore(t.TempDir())
	if err != nil {
		t.Fatalf("store: %v", err)
	}
	t.Cleanup(func() { s.Close() }) //nolint:errcheck
	mux := http.NewServeMux()
	admin.RegisterRoutes(mux, &admin.Deps{
		Store:    s,
		Registry: registry.NewRegistry(),
		Config:   cfg,
	})
	return mux
}

// ── Specs handler tests ───────────────────────────────────────────────────────

func TestSpecsListEmpty(t *testing.T) {
	mux := newMux(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/_api/specs", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET /_api/specs = %d, want 200", rec.Code)
	}
	var result []interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("response not valid JSON array: %v", err)
	}
	if len(result) != 0 {
		t.Errorf("expected empty list, got %d items", len(result))
	}
}

func TestSpecsCreateAndGet(t *testing.T) {
	mux := newMux(t)

	body := map[string]string{
		"name":         "Pet Store",
		"upstream_url": "https://api.example.com",
		"spec_raw":     petStoreSpecJSON,
	}
	bodyBytes, _ := json.Marshal(body)

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/_api/specs", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /_api/specs = %d, body: %s", rec.Code, rec.Body.String())
	}

	var created map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("response not valid JSON: %v", err)
	}
	id, _ := created["id"].(string)
	if id == "" {
		t.Fatal("created spec has no id")
	}

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/_api/specs/"+id, nil)
	mux.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("GET /_api/specs/%s = %d", id, rec2.Code)
	}
}

func TestSpecsGetNotFound(t *testing.T) {
	mux := newMux(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/_api/specs/nonexistent-id", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /_api/specs/nonexistent = %d, want 404", rec.Code)
	}
}

func TestSpecsDelete(t *testing.T) {
	mux := newMux(t)

	body := map[string]string{
		"name":         "Delete Me",
		"upstream_url": "https://api.example.com",
		"spec_raw":     petStoreSpecJSON,
	}
	bodyBytes, _ := json.Marshal(body)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/_api/specs", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	var created map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &created) //nolint:errcheck
	id := created["id"].(string)

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("DELETE", "/_api/specs/"+id, nil)
	mux.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusNoContent {
		t.Errorf("DELETE /_api/specs/%s = %d, want 204", id, rec2.Code)
	}

	rec3 := httptest.NewRecorder()
	req3 := httptest.NewRequest("GET", "/_api/specs/"+id, nil)
	mux.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusNotFound {
		t.Errorf("after delete, GET = %d, want 404", rec3.Code)
	}
}

func TestSpecsList(t *testing.T) {
	mux := newMux(t)

	for _, name := range []string{"Alpha", "Beta"} {
		body := map[string]string{
			"name": name, "upstream_url": "https://api.example.com", "spec_raw": petStoreSpecJSON,
		}
		b, _ := json.Marshal(body)
		rec := httptest.NewRecorder()
		req := httptest.NewRequest("POST", "/_api/specs", bytes.NewReader(b))
		req.Header.Set("Content-Type", "application/json")
		mux.ServeHTTP(rec, req)
	}

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/_api/specs", nil)
	mux.ServeHTTP(rec, req)

	var result []interface{}
	json.Unmarshal(rec.Body.Bytes(), &result) //nolint:errcheck
	if len(result) != 2 {
		t.Errorf("ListSpecs = %d items, want 2", len(result))
	}
}

// ── Health ────────────────────────────────────────────────────────────────────

func TestHealthEndpoint(t *testing.T) {
	mux := newMux(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/_api/health", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET /_api/health = %d, want 200", rec.Code)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("health response not valid JSON: %v", err)
	}
	if result["status"] != "ok" {
		t.Errorf("health status = %v, want ok", result["status"])
	}
}

// ── Resources ─────────────────────────────────────────────────────────────────

func TestResourcesListEmpty(t *testing.T) {
	mux := newMux(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/_api/resources", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("GET /_api/resources = %d, want 200", rec.Code)
	}
	var result []interface{}
	json.Unmarshal(rec.Body.Bytes(), &result) //nolint:errcheck
	if len(result) != 0 {
		t.Errorf("expected empty resources, got %d", len(result))
	}
}

func TestResourcesCreateText(t *testing.T) {
	mux := newMux(t)

	body := map[string]interface{}{
		"name":      "My Text Resource",
		"type":      "text",
		"mime_type": "text/plain",
		"content":   "Hello, world!",
	}
	bodyBytes, _ := json.Marshal(body)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/_api/resources", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /_api/resources = %d, body: %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &created) //nolint:errcheck
	if created["id"] == nil {
		t.Error("created resource has no id")
	}
}

func TestResourcesGetNotFound(t *testing.T) {
	mux := newMux(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/_api/resources/no-such-id", nil)
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("GET /_api/resources/no-such-id = %d, want 404", rec.Code)
	}
}

// ── Operations ────────────────────────────────────────────────────────────────

func TestOperationsListAndUpdate(t *testing.T) {
	mux := newMux(t)

	// Create spec first.
	body := map[string]string{
		"name": "Ops Spec", "upstream_url": "https://api.example.com", "spec_raw": petStoreSpecJSON,
	}
	bodyBytes, _ := json.Marshal(body)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/_api/specs", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)

	var created map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &created) //nolint:errcheck
	specID := created["id"].(string)

	// List operations.
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/_api/specs/"+specID+"/operations", nil)
	mux.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Fatalf("GET operations = %d", rec2.Code)
	}
	var ops []map[string]interface{}
	json.Unmarshal(rec2.Body.Bytes(), &ops) //nolint:errcheck
	if len(ops) == 0 {
		t.Skip("no operations extracted")
	}

	opID := ops[0]["id"].(string)
	currentEnabled := ops[0]["enabled"].(bool)

	patchBody, _ := json.Marshal(map[string]interface{}{"enabled": !currentEnabled})
	rec3 := httptest.NewRecorder()
	req3 := httptest.NewRequest("PATCH", "/_api/specs/"+specID+"/operations/"+opID, bytes.NewReader(patchBody))
	req3.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec3, req3)

	if rec3.Code != http.StatusOK {
		t.Errorf("PATCH operation = %d, body: %s", rec3.Code, rec3.Body.String())
	}
}

// ── CORS preflight ────────────────────────────────────────────────────────────

func TestCORSPreflightReturns204(t *testing.T) {
	cfg := adminCfg("", "secret")
	cfg.CORS.AllowedOrigins = []string{"https://example.com"}
	s, _ := store.NewJSONStore(t.TempDir())
	t.Cleanup(func() { s.Close() }) //nolint:errcheck
	mux := http.NewServeMux()
	admin.RegisterRoutes(mux, &admin.Deps{Store: s, Registry: registry.NewRegistry(), Config: cfg})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("OPTIONS", "/_api/specs", nil)
	req.Header.Set("Origin", "https://example.com")
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Errorf("OPTIONS = %d, want 204", rec.Code)
	}
}

// ── JSON error is valid JSON ──────────────────────────────────────────────────

func TestJSONErrorBodyIsValidJSON(t *testing.T) {
	cfg := adminCfg("pass", "secret")
	s, _ := store.NewJSONStore(t.TempDir())
	t.Cleanup(func() { s.Close() }) //nolint:errcheck
	mux := http.NewServeMux()
	admin.RegisterRoutes(mux, &admin.Deps{Store: s, Registry: registry.NewRegistry(), Config: cfg})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/_api/specs", nil) // no auth cookie
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("want 401, got %d", rec.Code)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(rec.Body.Bytes(), &result); err != nil {
		t.Fatalf("401 body not valid JSON: %v\nbody: %s", err, rec.Body.String())
	}
	if result["error"] == nil {
		t.Error("JSON error response must have 'error' field")
	}
}

// ── Resource Update / Delete / GetContent ─────────────────────────────────────

func TestResourcesUpdateAndDelete(t *testing.T) {
	mux := newMux(t)

	// Create a text resource.
	body := map[string]interface{}{
		"name": "UpdateMe", "type": "text", "mime_type": "text/plain", "content": "original",
	}
	b, _ := json.Marshal(body)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/_api/resources", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	var created map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &created) //nolint:errcheck
	id := created["id"].(string)

	// Update name.
	patch, _ := json.Marshal(map[string]string{"name": "Updated"})
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("PATCH", "/_api/resources/"+id, bytes.NewReader(patch))
	req2.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Errorf("PATCH resource = %d, body: %s", rec2.Code, rec2.Body.String())
	}

	// Delete it.
	rec3 := httptest.NewRecorder()
	req3 := httptest.NewRequest("DELETE", "/_api/resources/"+id, nil)
	mux.ServeHTTP(rec3, req3)
	if rec3.Code != http.StatusNoContent {
		t.Errorf("DELETE resource = %d, want 204", rec3.Code)
	}
}

func TestResourceUpdateNotFound(t *testing.T) {
	mux := newMux(t)
	patch, _ := json.Marshal(map[string]string{"name": "x"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/_api/resources/no-such", bytes.NewReader(patch))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("PATCH nonexistent = %d, want 404", rec.Code)
	}
}

func TestResourceDeleteNotFound(t *testing.T) {
	mux := newMux(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("DELETE", "/_api/resources/no-such", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("DELETE nonexistent = %d, want 404", rec.Code)
	}
}

// ── Chat config ───────────────────────────────────────────────────────────────

func TestChatGetConfigNoKey(t *testing.T) {
	mux := newMux(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/_api/chat/config", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /_api/chat/config = %d", rec.Code)
	}
	var result map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &result) //nolint:errcheck
	if result["hasKey"] != false {
		t.Errorf("hasKey = %v, want false", result["hasKey"])
	}
}

func TestChatCompletionsNoKey(t *testing.T) {
	mux := newMux(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/_api/chat/completions", strings.NewReader(`{"messages":[]}`))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	// No OpenAI key configured → 503 Service Unavailable
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("POST /_api/chat/completions (no key) = %d, want 503", rec.Code)
	}
}

// ── Stats handler ─────────────────────────────────────────────────────────────

func TestStatsEndpoint(t *testing.T) {
	mux := newMux(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/_api/stats", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("GET /_api/stats = %d, want 200", rec.Code)
	}
	var result map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &result) //nolint:errcheck
	if _, ok := result["totalSpecs"]; !ok {
		t.Error("stats response missing 'totalSpecs'")
	}
}

func TestToolStatsEndpoint(t *testing.T) {
	mux := newMux(t)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/_api/stats/tools", nil)
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("GET /_api/stats/tools = %d, want 200", rec.Code)
	}
}

// ── Spec Update ───────────────────────────────────────────────────────────────

func TestSpecsUpdateName(t *testing.T) {
	mux := newMux(t)

	// Create spec.
	b, _ := json.Marshal(map[string]string{
		"name": "OriginalName", "upstream_url": "https://api.example.com", "spec_raw": petStoreSpecJSON,
	})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/_api/specs", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	var created map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &created) //nolint:errcheck
	id := created["id"].(string)

	// Patch it.
	patch, _ := json.Marshal(map[string]string{"name": "UpdatedName"})
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("PATCH", "/_api/specs/"+id, bytes.NewReader(patch))
	req2.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusOK {
		t.Errorf("PATCH spec = %d, body: %s", rec2.Code, rec2.Body.String())
	}
	var updated map[string]interface{}
	json.Unmarshal(rec2.Body.Bytes(), &updated) //nolint:errcheck
	if updated["name"] != "UpdatedName" {
		t.Errorf("updated name = %v, want UpdatedName", updated["name"])
	}
}

func TestSpecsUpdateNotFound(t *testing.T) {
	mux := newMux(t)
	patch, _ := json.Marshal(map[string]string{"name": "x"})
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("PATCH", "/_api/specs/no-such-id", bytes.NewReader(patch))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Errorf("PATCH nonexistent spec = %d, want 404", rec.Code)
	}
}

// ── Resource GetContent ───────────────────────────────────────────────────────

func TestResourceGetContent(t *testing.T) {
	cfg := adminCfg("", "secret")
	cfg.DataDir = t.TempDir()
	s, _ := store.NewJSONStore(t.TempDir())
	t.Cleanup(func() { s.Close() }) //nolint:errcheck
	mux := http.NewServeMux()
	admin.RegisterRoutes(mux, &admin.Deps{Store: s, Registry: registry.NewRegistry(), Config: cfg})

	// Create resource via API — admin createJSON writes the file to DataDir.
	body := map[string]interface{}{
		"name": "text-res", "type": "text", "mime_type": "text/plain", "content": "hello content",
	}
	b, _ := json.Marshal(body)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/_api/resources", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	mux.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST resource = %d, body: %s", rec.Code, rec.Body.String())
	}
	var created map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &created) //nolint:errcheck
	id := created["id"].(string)

	// Get content.
	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/_api/resources/"+id+"/content", nil)
	mux.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusOK {
		t.Errorf("GET content = %d, body: %s", rec2.Code, rec2.Body.String())
	}
	if rec2.Body.String() != "hello content" {
		t.Errorf("content = %q, want %q", rec2.Body.String(), "hello content")
	}
}

func TestResourceGetContentUpstreamError(t *testing.T) {
	// Create an upstream resource directly in the store.
	s2, _ := store.NewJSONStore(t.TempDir())
	t.Cleanup(func() { s2.Close() }) //nolint:errcheck
	cfg2 := adminCfg("", "secret")
	mux2 := http.NewServeMux()
	admin.RegisterRoutes(mux2, &admin.Deps{Store: s2, Registry: registry.NewRegistry(), Config: cfg2})

	// Upstream type resources can't be fetched via content endpoint.
	upstreamBody := map[string]interface{}{
		"name": "upstream-res", "type": "upstream", "upstream_url": "https://api.example.com/data",
	}
	b, _ := json.Marshal(upstreamBody)
	rec := httptest.NewRecorder()
	req := httptest.NewRequest("POST", "/_api/resources", bytes.NewReader(b))
	req.Header.Set("Content-Type", "application/json")
	mux2.ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST upstream resource = %d", rec.Code)
	}
	var created map[string]interface{}
	json.Unmarshal(rec.Body.Bytes(), &created) //nolint:errcheck
	id := created["id"].(string)

	rec2 := httptest.NewRecorder()
	req2 := httptest.NewRequest("GET", "/_api/resources/"+id+"/content", nil)
	mux2.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusBadRequest {
		t.Errorf("GET upstream content = %d, want 400", rec2.Code)
	}
}
