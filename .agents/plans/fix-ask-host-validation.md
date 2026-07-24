# fix/ask-host-validation

## Problem

`la-famille ask` binds to `127.0.0.1` and performs no `Host` or `Origin`
validation. Verified against the running server before the fix:

```
Host: evil.com  GET /            -> HTTP 200
Host: evil.com  GET /api/status  -> HTTP 200
```

A cross-origin *simple* POST to `/api/ask` (`Content-Type: text/plain`,
`Origin: https://evil.com`) was also accepted and processed — it reached the LLM
provider and returned 503 only because Ollama was not configured locally.

Binding to loopback keeps other **machines** out; it does not keep other
**origins** out. A page on any site can point a hostname it controls at
`127.0.0.1` (DNS rebinding). The browser then treats the assistant as
same-origin, so the page can both post questions and read the answers —
including the site content the corpus was built from.

That contradicts the feature's stated guarantee in `content/docs/ask.md`:
"never sends content off your machine."

The absence of CORS response headers is currently the only thing preventing
cross-origin *reads*, and rebinding removes it, because after rebinding the
request is no longer cross-origin.

## Decision

Validate the `Host` header. It is the signal that survives rebinding: the
browser keeps sending the attacker's hostname even once it resolves to
127.0.0.1, so a hostname allowlist rejects the rebound request while leaving
genuine local access untouched.

Rejected: adding CORS headers. That controls who may *read* a response, not who
may *reach* the handler, and it does nothing against rebinding (same-origin
requests are not subject to CORS).

Rejected: a CSRF token. It would work, but it needs state and a UI handshake for
a problem the `Host` check solves in a few lines, and it would not stop a
rebound page from simply loading the UI and reading the token.

## Design

`guardHost` middleware wraps the mux. When `Config.LoopbackOnly` is true (the
default; false only when the operator passes `--expose-host`), a request whose
`Host` does not resolve to this server's loopback address is refused with
`403`.

`hostAllowed` accepts the configured bind host verbatim plus anything the
existing `IsLoopbackHost` helper recognises (`localhost`, `127.0.0.0/8`, `::1`,
bracketed IPv6, optional port). An empty `Host` is refused — HTTP/1.1 requires
one.

The guard is skipped entirely under `--expose-host`, because a deliberately
exposed deployment is legitimately reached under arbitrary hostnames and behind
proxies, and that flag already carries a startup warning.

## Files

- `internal/ask/server.go` — `hostAllowed`, `guardHost`, mux wrapped in `Start`
- `internal/ask/host_guard_test.go` (new)
- `content/docs/ask.md` — document the check under the privacy guarantees

## Breaking changes to static asset generation

None. No change to the site build.

## Validation

- `go test ./...`
- `go vet ./...`
- Manual, against the running server:
  `Host: evil.com` and `Host: 127.0.0.1.evil.com` -> 403;
  `Host: 127.0.0.1:PORT`, `Host: localhost:PORT`, and an ordinary request -> 200.
