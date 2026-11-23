package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBuzonesYWhitelist(t *testing.T) {
	s := NewInMemoryServer("")
	h := s.Handler()
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/mailboxes", nil))
	if !strings.Contains(w.Body.String(), "dev@trotamundos.app") {
		t.Fatalf("buzones: %s", w.Body.String())
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/whitelists", strings.NewReader(`{"sender":"@renter.com"}`)))
	if w.Code != http.StatusCreated {
		t.Fatalf("alta whitelist: %d %s", w.Code, w.Body.String())
	}
	w = httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/whitelists", nil))
	if !strings.Contains(w.Body.String(), "@renter.com") {
		t.Fatalf("whitelist no lista el remitente: %s", w.Body.String())
	}
}

func TestDedupYWhitelistEnIngesta(t *testing.T) {
	s := NewInMemoryServer("")
	h := s.Handler()
	body := `{"subject":"Confirmación","body":"reserva PNR-8899 con Aerolíneas del Plata","from":"noreply@aerolineasdelplata.com"}`
	for i, want := range []string{"accepted", "duplicate"} {
		w := httptest.NewRecorder()
		h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/ingest/email", strings.NewReader(body)))
		if !strings.Contains(w.Body.String(), want) {
			t.Fatalf("intento %d: esperaba %s, got %d %s", i, want, w.Code, w.Body.String())
		}
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/api/ingest/email", strings.NewReader(`{"subject":"Otra","body":"reserva PNR-9900","from":"spam@ejemplo.com"}`)))
	if w.Code != http.StatusForbidden {
		t.Fatalf("remitente no autorizado: status=%d", w.Code)
	}
}

func TestStreamSSE(t *testing.T) {
	s := NewInMemoryServer("")
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/events/stream", nil)
	go s.Handler().ServeHTTP(w, req)
	// El evento ready se escribe antes de entrar al bucle.
	for i := 0; i < 50 && !strings.Contains(w.Body.String(), "event: ready"); i++ {
		time.Sleep(2 * time.Millisecond)
	}
	if !strings.Contains(w.Body.String(), "event: ready") {
		t.Fatal("el stream no envió el evento ready")
	}
}
