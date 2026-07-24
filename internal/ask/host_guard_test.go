package ask

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHostAllowed(t *testing.T) {
	s := &Server{cfg: Config{Host: "127.0.0.1", Port: 8090, LoopbackOnly: true}}

	allowed := []string{
		"127.0.0.1:8090",
		"127.0.0.1",
		"localhost:8090",
		"localhost",
		"LOCALHOST:8090",
		"[::1]:8090",
		"127.0.0.5:8090", // all of 127.0.0.0/8 is loopback
	}
	for _, h := range allowed {
		if !s.hostAllowed(h) {
			t.Errorf("hostAllowed(%q) = false, want true", h)
		}
	}

	// The rebinding cases: the browser keeps sending the attacker's hostname
	// even after it resolves to 127.0.0.1.
	denied := []string{
		"",
		"   ",
		"evil.com",
		"evil.com:8090",
		"attacker.localhost.example.com",
		"127.0.0.1.evil.com",
		"192.168.1.10:8090",
		"[::]:8090",
	}
	for _, h := range denied {
		if s.hostAllowed(h) {
			t.Errorf("hostAllowed(%q) = true, want false", h)
		}
	}
}

func TestHostAllowedHonoursConfiguredHost(t *testing.T) {
	// An operator who bound to a specific loopback alias must still be served.
	s := &Server{cfg: Config{Host: "127.0.0.2", Port: 8090, LoopbackOnly: true}}
	if !s.hostAllowed("127.0.0.2:8090") {
		t.Error("configured host was rejected")
	}
}

func TestGuardHostBlocksForgedHostHeader(t *testing.T) {
	s := &Server{cfg: Config{Host: "127.0.0.1", Port: 8090, LoopbackOnly: true}}
	reached := false
	h := s.guardHost(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8090/api/status", nil)
	req.Host = "evil.com"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if reached {
		t.Fatal("handler ran despite a forged Host header")
	}
	if rec.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want %d", rec.Code, http.StatusForbidden)
	}
}

func TestGuardHostAllowsLoopbackRequest(t *testing.T) {
	s := &Server{cfg: Config{Host: "127.0.0.1", Port: 8090, LoopbackOnly: true}}
	reached := false
	h := s.guardHost(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://127.0.0.1:8090/api/status", nil)
	req.Host = "127.0.0.1:8090"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if !reached || rec.Code != http.StatusOK {
		t.Fatalf("legitimate loopback request was blocked (reached=%v, code=%d)", reached, rec.Code)
	}
}

// An operator who passed --expose-host has opted into arbitrary hostnames and
// proxies, so the guard must not break that path.
func TestGuardHostSkippedWhenExposed(t *testing.T) {
	s := &Server{cfg: Config{Host: "0.0.0.0", Port: 8090, LoopbackOnly: false}}
	reached := false
	h := s.guardHost(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		reached = true
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "http://example.com/api/status", nil)
	req.Host = "some.public.hostname"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if !reached || rec.Code != http.StatusOK {
		t.Fatalf("guard fired despite --expose-host (reached=%v, code=%d)", reached, rec.Code)
	}
}
