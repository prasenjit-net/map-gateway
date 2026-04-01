# MCP Gateway — Code Improvement Analysis

> Comprehensive static analysis of the codebase identifying security vulnerabilities, concurrency issues, performance gaps, and missing features. Issues are rated **HIGH**, **MEDIUM**, or **LOW** severity.
>
> **Last updated:** March 2026 — items addressed in prior work have been removed.

---

## Recently Addressed ✅

The following issues from the original analysis have been resolved:

| Severity | Area | Resolution |
|----------|------|-----------|
| **HIGH** | Admin API unauthenticated | Form-based auth with HMAC-SHA256 signed cookies (`admin/session.go`, `admin/auth_middleware.go`, `admin/login.go`). `/_api/*` → 401 JSON; `/_ui/*` → redirect to login. |
| **HIGH** | CORS wildcard on admin routes | Replaced hardcoded `*` with configurable `CORSConfig` allowlist (`admin/cors.go`). Secure default: same-origin only. |
| **HIGH** | Path traversal in file upload (`admin/resources.go`) | `store.SafeJoin` validates resolved path stays within `DataDir`. File permissions tightened to `0o600`/`0o750`; `O_EXCL` on create prevents race. |
| **HIGH** | No HTTP client timeout (`admin/chat.go`) | Configurable `ChatTimeoutSeconds` (default 60 s); `io.LimitReader` caps streaming response at 32 MiB; write errors no longer ignored. |
| **HIGH** | Goroutine leak — registry subscription (`cmd/serve.go`) | Goroutine now uses `shutdownCtx` + `Unsubscribe`; exits when context is cancelled. |
| **HIGH** | Incomplete shutdown — TLS multiplexer (`cmd/serve.go`) | `cmuxInst.Close()` called after `server.Shutdown()`; signal channel capacity raised to 2. |
| **HIGH** | Gateway secret read from environment (`config/config.go`) | `GatewaySecretFile` config option reads secret from a permissions-restricted file; `GATEWAY_SECRET` env still overrides as fallback. |
| **HIGH** | File path traversal in resource read (`mcp/handlers.go`) | `safeJoinDataDir` wrapper around `store.SafeJoin` validates path before `os.ReadFile`. |
| **HIGH** | SSE session channel not drained on cancel (`mcp/sse.go`) | `HandleSSE` defers close + drain of `outCh`; `HandleMessage` selects on `session.ctx.Done()` (returns 410 Gone). |
| **HIGH** | Path traversal in `store.DeleteResource` | `store.SafeJoin` guards `os.Remove` in `jsonstore.go`. |
| **MEDIUM** | Files written with world-readable permissions (`admin/resources.go`) | Changed from `0644` → `0o600`; directories `0755` → `0o750`. |
| **MEDIUM** | No streaming response size limit (`admin/chat.go`) | `io.LimitReader` caps response body at 32 MiB. |
| **MEDIUM** | Write errors ignored (`admin/chat.go`) | Write errors now checked; loop breaks on disconnect. |
| **MEDIUM** | No timeout on OAuth2 token fetch (`auth/oauth2.go`) | Configurable `OAuthTimeoutSeconds` (default 10 s) per-request `http.Client`. |
| **MEDIUM** | Overly generous default OAuth2 expiry (`auth/oauth2.go`) | Changed from 3600 s → 300 s (`OAuthDefaultExpirySeconds`). |
| **MEDIUM** | No JSON body size limit (`mcp/http.go`) | Body wrapped with `io.LimitReader(MaxRequestBytes)` before decode. |
| **MEDIUM** | Request body size unlimited (`mcp/sse.go`) | `HandleMessage` body wrapped with `io.LimitReader`. |
| **MEDIUM** | URL path argument injection (`proxy/builder.go`) | `url.PathEscape` used for all path placeholder values. |
| **MEDIUM** | HTTP method not allowlisted (`proxy/builder.go`) | `allowedMethods` map rejects TRACE, CONNECT, and other unsafe methods. |
| **MEDIUM** | URL assembled before validation (`proxy/builder.go`) | Upstream URL validated before path concatenation. |
| **MEDIUM** | JSON injection via string concatenation | `jsonString()` helper in `admin/router.go` uses `json.Marshal` instead of string concatenation. |
| **LOW** | Zero test coverage | Test coverage raised to ≥60% (currently 60.2%) and enforced in `make test`, CI, and release workflows. |
| **BUG** | TOML config scoping bug | `openai_api_key` was silently ignored when placed after `[cors]`. Moved before all section headers in `DefaultConfigContent`; regression test added. |

---

## Table of Contents

1. [Admin Package](#1-admin-package)
2. [Auth Package](#2-auth-package)
3. [Build Info Package](#3-buildinfo-package)
4. [CMD Package](#4-cmd-package)
5. [Config Package](#5-config-package)
6. [MCP Package](#6-mcp-package)
7. [Proxy Package](#7-proxy-package)
8. [Registry Package](#8-registry-package)
9. [Store Package](#9-store-package)
10. [Telemetry Package](#10-telemetry-package)
11. [TLS Util Package](#11-tlsutil-package)
12. [Dockerfile & Build System](#12-dockerfile--build-system)
13. [Cross-Cutting Issues](#13-cross-cutting-issues)
14. [Prioritised Recommendations](#14-prioritised-recommendations)

---

## 1. Admin Package

### `admin/specs.go`

| Severity | Issue | Details |
|----------|-------|---------|
| **HIGH** | Potential path traversal | `filepath.Join("resources", id)` (line 55) is currently safe because IDs are UUIDs, but provides no defence-in-depth. Add an explicit check that the resolved path starts with `filepath.Abs(h.config.DataDir)`. |
| **MEDIUM** | Silent JSON unmarshal error | `json.Unmarshal([]byte(ph), &passthroughHeaders)` (line 47) discards its error return, allowing malformed passthrough-header config to be silently accepted. |
| **MEDIUM** | Multipart form size too large / no quotas | 32 MB upload limit per request with no per-user quotas exposes the service to disk-exhaustion DoS. |
| **MEDIUM** | Race condition on registry rebuild | `RebuildRegistryFromStore` is called synchronously in request handlers (lines 123, 206, 223) without debouncing; concurrent requests can trigger simultaneous rebuilds. |

### `admin/resources.go`

| Severity | Issue | Details |
|----------|-------|---------|
| **LOW** | No MIME-type content validation | Uploaded files are not verified to match their declared MIME type. Consider using magic-byte inspection (e.g. `github.com/gabriel-vasile/mimetype`). |
| **LOW** | Incomplete hardcoded MIME detection | Lines 332–361 do not cover many common formats (WebP, MP4, etc.). Use a library instead. |

### `admin/router.go`

| Severity | Issue | Details |
|----------|-------|---------|
| **MEDIUM** | No input validation middleware | All routes lack validation of path parameters, query strings, and body size limits. Add middleware to enforce `Content-Length` caps and parameter format checks. |

### `admin/stats.go`

| Severity | Issue | Details |
|----------|-------|---------|
| **MEDIUM** | Store errors silently ignored | `h.store.ListSpecs()` and `h.store.GetAllStats()` errors are discarded (lines 23, 25). Log at warning level. |
| **LOW** | Package-level `startTime` variable | `var startTime = time.Now()` (line 14) tracks uptime only from package init, not from actual server start. Pass start time as a dependency. |

### `admin/operations.go`

| Severity | Issue | Details |
|----------|-------|---------|
| **LOW** | No audit logging | Enable/disable operations have no audit trail. Add structured log entries (who, when, what). |

---

## 2. Auth Package

### `auth/auth.go`

| Severity | Issue | Details |
|----------|-------|---------|
| **MEDIUM** | Nil authenticator allows passthrough | If the configured authenticator is `nil` and inbound auth exists, the inbound token is forwarded without validation (lines 43–55). Enforce that at least one auth method is active. |

### `auth/encrypt.go`

| Severity | Issue | Details |
|----------|-------|---------|
| **MEDIUM** | Weak key derivation | `sha256.Sum256([]byte(secret))` (line 14) is a single fast hash with no salt or iteration count — inadequate for password-grade secrets. Use `scrypt` or `argon2id`. |
| **MEDIUM** | No authenticated additional data | GCM encryption omits AD (additional authenticated data); header tampering cannot be detected. Include a spec/config identifier as AD. |
| **LOW** | Nonce reuse not explicitly guarded | If `rand.Read` fails silently, a short nonce could be reused. Explicitly verify `len(nonce) == gcm.NonceSize()`. |

### `auth/apikey.go`

| Severity | Issue | Details |
|----------|-------|---------|
| **MEDIUM** | Header name case sensitivity | String comparison for header names is case-sensitive (line 33) but HTTP headers are canonically case-insensitive. Use `http.CanonicalHeaderKey()`. |

### `auth/oauth2.go`

| Severity | Issue | Details |
|----------|-------|---------|
| **HIGH** | Token cache expiry race | The 30-second expiry buffer check happens while the lock is held, but the fetch may still serve a near-expiry token to concurrent callers that slip in between the `RUnlock` and `Lock` (line 60). Use a single exclusive lock path or `sync.Once`-style lazy init. |
| **MEDIUM** | No retry on token fetch failure | `http.PostForm` is called once (line 77); transient network errors propagate directly to the caller. Add exponential-backoff retry. |

### `auth/basic.go` / `auth/bearer.go`

| Severity | Issue | Details |
|----------|-------|---------|
| **LOW** | Credential/token leakage in logs | Error paths may include partial credentials or tokens in messages. Return generic error strings and never log raw credentials. |

---

## 3. Buildinfo Package

| Severity | Issue | Details |
|----------|-------|---------|
| **LOW** | No semantic-version validation | Version string is set at build time but never verified to follow semver. Validate in `init()`. |

---

## 4. CMD Package

### `cmd/serve.go`

| Severity | Issue | Details |
|----------|-------|---------|
| **MEDIUM** | Config values not validated at startup | Listen address format, data-directory writeability, and certificate paths are not checked before the server starts (lines 46–59). |
| **MEDIUM** | Asymmetric error handling | TLS mode logs but continues on server error; non-TLS mode calls `os.Exit(1)`. Behaviour should be consistent (lines 166–185). |

### `cmd/init.go`

| Severity | Issue | Details |
|----------|-------|---------|
| **MEDIUM** | TOCTOU race on config file creation | `os.Stat` followed by `os.WriteFile` (lines 27–29) has a race if another process creates the file in between. Use `os.OpenFile` with `O_CREATE|O_EXCL`. |

---

## 5. Config Package

### `config/config.go`

| Severity | Issue | Details |
|----------|-------|---------|
| **MEDIUM** | No configuration validation | No validation function exists for listen address format, data-directory writeability, or cert/key path existence. Add `func (c *Config) Validate() error`. |
| **MEDIUM** | TLS private key stored in data directory | Default key path is inside `DataDir` (lines 104–118), which may be accessible to uploaded file paths. Store TLS material in a dedicated, permission-restricted directory. |
| **MEDIUM** | `MaxResponseBytes` not bounds-checked | The value is read but not validated; a misconfiguration of 0 or a negative value would silently disable the limit. |

---

## 6. MCP Package

### `mcp/handlers.go`

| Severity | Issue | Details |
|----------|-------|---------|
| **HIGH** | Double-checked locking race | `getAuthenticator` releases the read lock, then acquires a write lock without re-checking under the read lock, allowing two goroutines to both build a new authenticator (lines 167–194). Simplify to a single write-lock path. |
| **MEDIUM** | Upstream response not read-timeout guarded | If `ctx` has no deadline (line 34), upstream requests can hang indefinitely. Callers should always set a deadline; document this contract or enforce it internally. |
| **MEDIUM** | Unbounded tool input size | `json.Unmarshal(req.Params, &params)` (line 94) has no size limit; large payloads can cause memory exhaustion. Enforce limits at the HTTP handler layer. |
| **MEDIUM** | Silent stat write failures | `_ = h.Store.IncrementStats(...)` (lines 121, 132, 142) discards all errors. Log at warning level. |
| **LOW** | Unstructured error messages | Tool errors are embedded in text content, making it impossible for clients to distinguish network errors from upstream application errors. Return structured error objects. |

---

## 6. MCP Package

### `mcp/handlers.go`

| Severity | Issue | Details |
|----------|-------|---------|
| **HIGH** | Double-checked locking race | `getAuthenticator` releases the read lock, then acquires a write lock without re-checking under the read lock, allowing two goroutines to both build a new authenticator (lines 167–194). Simplify to a single write-lock path. |
| **MEDIUM** | Upstream response not read-timeout guarded | If `ctx` has no deadline (line 34), upstream requests can hang indefinitely. Callers should always set a deadline; document this contract or enforce it internally. |
| **MEDIUM** | Unbounded tool input size | `json.Unmarshal(req.Params, &params)` (line 94) has no size limit; large payloads can cause memory exhaustion. Enforce limits at the HTTP handler layer. |
| **MEDIUM** | Silent stat write failures | `_ = h.Store.IncrementStats(...)` (lines 121, 132, 142) discards all errors. Log at warning level. |
| **LOW** | Unstructured error messages | Tool errors are embedded in text content, making it impossible for clients to distinguish network errors from upstream application errors. Return structured error objects. |

### `mcp/http.go`

| Severity | Issue | Details |
|----------|-------|---------|
| **MEDIUM** | Empty batch encodes as `null` | If the batch response slice is nil, `json.Encoder.Encode` writes `null` instead of `[]`. Initialise with `make([]*Response, 0)`. |

### `mcp/sse.go`

| Severity | Issue | Details |
|----------|-------|---------|
| **MEDIUM** | Fixed-size channel causes message drops | `outCh: make(chan []byte, 64)` (line 47): slow clients silently drop messages. Increase buffer size or implement backpressure. |
| **MEDIUM** | `sessionId` not format-validated | The session ID is accepted as any non-empty string (line 100). Validate UUID format to prevent injection. |
| **MEDIUM** | Race condition on `session.inbound` | `mergeInboundAuth` mutates `session.inbound` without a lock (line 119). Use a `sync.RWMutex` or `atomic.Value`. |

---

## 7. Proxy Package

### `proxy/proxy.go`

| Severity | Issue | Details |
|----------|-------|---------|
| **MEDIUM** | Unsafe type assertion | `v.(*http.Client)` (lines 39, 53) will panic if the stored value is not an `*http.Client`. Add a comma-ok check. |
| **MEDIUM** | No connection pool cleanup | HTTP clients accumulate per-spec without any idle-connection or shutdown cleanup. Call `transport.CloseIdleConnections()` periodically or on spec removal. |
| **LOW** | `MaxIdleConnsPerHost` too high | 20 connections per host can exhaust system file descriptors. 5–10 is adequate for most use cases. |

### `proxy/mapper.go`

| Severity | Issue | Details |
|----------|-------|---------|
| **MEDIUM** | Pretty-printed JSON may exceed `MaxResponseBytes` | `json.MarshalIndent` (line 42) can produce output significantly larger than the raw JSON. Check the pretty-printed size before accepting it. |

---

## 8. Registry Package

### `registry/registry.go`

| Severity | Issue | Details |
|----------|-------|---------|
| **MEDIUM** | No pre-allocation in list methods | `ListStaticResources`, `ListResourceTemplates`, and `ListPrompts` (lines 89–122) grow slices inside a read lock without pre-allocating. Use `make([]*T, 0, len(r.resources))`. |
| **MEDIUM** | O(n) unsubscribe | Linear search and slice reallocation on `Unsubscribe` (lines 61–70) degrades under high subscriber counts. Track subscribers with a `map[chan struct{}]struct{}`. |
| **LOW** | Send on closed channel is unsafe | The non-blocking send at line 75 is safe, but nothing documents that subscribers must not close the channel. Add a comment or enforce this with a wrapper type. |

---

## 9. Store Package

### `store/jsonstore.go`

| Severity | Issue | Details |
|----------|-------|---------|
| **HIGH** | Non-atomic stat counter updates | `st.CallCount++`, `st.ErrorCount++`, and `st.TotalLatencyMs += latencyMs` (line 216) are not protected by any lock or atomic operation; concurrent callers produce lost updates. Use a single mutex held across all three increments, or switch to `sync/atomic`. |
| **MEDIUM** | Stats lost on startup | Writes that arrive between loading stats from disk and the first flush are dropped (lines 68–84). Merge loaded stats with in-memory state on startup. |
| **MEDIUM** | File deletion errors silently ignored | `_ = os.Remove(...)` (lines 293, 297) discards errors. Log removals that fail as warnings. |
| **MEDIUM** | Path traversal in `GetResource` | `filepath.Join(h.config.DataDir, rec.FilePath)` (line 257) is unsafe if `FilePath` contains `..`. Validate at record load time. |
| **LOW** | `os.Rename` may fail cross-device | The atomic-write helper uses `os.Rename` (line 101) which fails if `tmp` and destination are on different filesystems. Implement a fallback copy-then-delete. |

### `store/store.go` / `store/models.go`

| Severity | Issue | Details |
|----------|-------|---------|
| **LOW** | Store interface lacks error classification | Callers cannot distinguish "not found" from "permission denied". Define sentinel errors or an error type enum. |
| **LOW** | No field-level validation in models | `SpecRecord`, `ResourceRecord`, etc. accept empty names and arbitrary URL strings. Validate at the store boundary. |

---

## 10. Telemetry Package

| Severity | Issue | Details |
|----------|-------|---------|
| **LOW** | No log rotation | Logs are written to stdout; if redirected to a file, they grow unbounded. Document log management expectations. |
| **LOW** | Inconsistent log level aliases | Only `debug`, `warn`, and `error` are recognised; `warning` is silently rejected. Accept common aliases. |
| **LOW** | Missing metric documentation | Prometheus metric definitions lack `Help` text explaining their purpose and labels. |

---

## 11. TLSUtil Package

### `tlsutil/certgen.go`

| Severity | Issue | Details |
|----------|-------|---------|
| **MEDIUM** | No expiry warning mechanism | The self-signed cert is valid for 10 years (line 49) but there is no code to warn operators before it expires. Log a warning on startup if expiry is within 30 days. |
| **MEDIUM** | Self-signed cert marked as CA | `IsCA: true` (line 52) is unnecessary for a leaf server certificate and could be misused. Only set `IsCA` for actual CA certificates. |
| **LOW** | Hardcoded SAN entries | `DNSNames` and `IPAddresses` are hard-coded to `localhost`/`127.0.0.1` (lines 53–54). Make these configurable so the cert is usable for custom domain deployments. |

---

## 12. Dockerfile & Build System

### `Dockerfile`

| Severity | Issue | Details |
|----------|-------|---------|
| **MEDIUM** | No `HEALTHCHECK` directive | Container orchestrators cannot detect a hung process. Add a `HEALTHCHECK` that probes the `/healthz` or `/metrics` endpoint. |
| **MEDIUM** | Container runs as root | No non-root user is created in the final stage. Add `RUN adduser -u 1000 mcp` and `USER 1000`. |
| **MEDIUM** | Distroless image has no shell | Debugging is impossible in production. Document this limitation and optionally provide a dev image with a shell. |

### `Makefile`

| Severity | Issue | Details |
|----------|-------|---------|
| **LOW** | `-s -w` strips all debug symbols | Makes crash analysis impossible. Only strip symbols in a dedicated `release` target, not the default `build`. |
| **LOW** | Build date not embedded | Builds are not reproducible and lack a timestamp. Embed `buildTime` and commit hash via `-ldflags`. |

---

## 13. Cross-Cutting Issues

### Security

| Severity | Area | Detail |
|----------|------|--------|
| **HIGH** | Path traversal in `admin/specs.go` | `filepath.Join("resources", id)` relies on UUIDs for safety but lacks defence-in-depth. Add explicit `SafeJoin` validation. |
| **MEDIUM** | Weak key derivation | Single SHA-256 with no salt used to derive encryption keys. |

### Concurrency

| Severity | Area | Detail |
|----------|------|--------|
| **HIGH** | Double-checked locking race | `mcp/handlers.go` `getAuthenticator` is unsafe. |
| **HIGH** | Non-atomic stat counters | `store/jsonstore.go` counter increments are unprotected. |
| **MEDIUM** | Session `inbound` auth race | `mcp/sse.go` session struct mutated without synchronisation. |
| **MEDIUM** | Registry rebuild on concurrent requests | `admin/specs.go` triggers simultaneous rebuilds without debouncing. |

### Error Handling

| Severity | Area | Detail |
|----------|------|--------|
| **MEDIUM** | Store errors silently discarded | `admin/stats.go`, `mcp/handlers.go`, `store/jsonstore.go` all discard error returns with `_`. |

### Observability & Missing Features

| Severity | Area | Detail |
|----------|------|--------|
| **MEDIUM** | No request/response audit logging | Sensitive operations (spec creation, operation enable/disable) leave no audit trail. |
| **MEDIUM** | No rate limiting | Neither admin API nor MCP endpoint have rate limiting. |
| **MEDIUM** | No request correlation IDs | Logs cannot be correlated across components. Add `X-Request-ID` middleware. |
| **MEDIUM** | No circuit breaker for upstream calls | A slow upstream blocks proxy goroutines with no fallback. |
| **LOW** | No CI linting | `golangci-lint` or `staticcheck` is not configured. |

---

## 14. Prioritised Recommendations

### Critical — Fix Before Production

1. **Fix race conditions** — stat counters (`store/jsonstore.go`), `getAuthenticator` double-check lock, SSE `inbound` auth.
2. **Add path traversal guard** in `admin/specs.go` — add `SafeJoin` to the spec file path (currently relies on UUID format).

### High Priority

3. Add HTTP client timeouts to `proxy/proxy.go` (currently only `admin/chat.go` and `auth/oauth2.go` are fixed).
4. Fix double-checked locking in `getAuthenticator` (`mcp/handlers.go`).
5. Add structured logging and request correlation IDs.
6. Implement rate limiting on both admin and MCP endpoints.
7. Add audit logging for all mutating admin operations.
8. Document and enforce thread-safety contracts on all exported types.

### Medium Priority

9. Return structured error codes from MCP tool handlers.
10. Add a circuit breaker for upstream proxy calls.
11. Validate configuration at startup (`config/config.go`).
12. Improve OAuth2 token caching — retry on transient failure; address double-checked locking race.
13. Add `HEALTHCHECK` to `Dockerfile` and run container as non-root.

### Low Priority

14. Add `golangci-lint` / `staticcheck` to the CI pipeline.
15. Use a MIME detection library instead of hardcoded extension mapping.
16. Make TLS SAN entries configurable.
17. Embed build timestamp and commit hash in binary.
18. Pre-allocate slices in registry list methods.

---

## Summary Statistics

| Severity | Original | Remaining |
|----------|----------|-----------|
| HIGH | 16 | 4 |
| MEDIUM | 42 | 28 |
| LOW | 27 | 26 |
| **Total** | **85+** | **58** |

**Highest-risk areas remaining**: concurrency safety (double-checked locking, non-atomic stat counters), SSE `inbound` race, and `admin/specs.go` path traversal hardening.
