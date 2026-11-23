package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"trotamundos/filius/internal/store"
	"trotamundos/shared/event"
)

type fakePublisher struct{ events []event.Event }

func (f *fakePublisher) Publish(_ context.Context, e event.Event) error {
	f.events = append(f.events, e)
	return nil
}

func newServer() (*Server, *fakePublisher) {
	f := &fakePublisher{}
	return &Server{Store: store.New(), Publisher: f}, f
}

func do(t *testing.T, h http.Handler, method, path, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	return w
}

func TestCrearYListarViajes(t *testing.T) {
	s, _ := newServer()
	h := s.Handler()
	w := do(t, h, http.MethodPost, "/api/trips", `{"name":"Madrid 1930","start_date":"2026-12-01","end_date":"2026-12-10"}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST status = %d, body=%s", w.Code, w.Body.String())
	}
	w = do(t, h, http.MethodGet, "/api/trips", "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET status = %d", w.Code)
	}
	var out struct{ Trips []struct{ Name string } `json:"trips"` }
	if err := json.Unmarshal(w.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Trips) != 1 || out.Trips[0].Name != "Madrid 1930" {
		t.Fatalf("lista inesperada: %+v", out.Trips)
	}
}

func TestEventoDescubiertoEsIdempotente(t *testing.T) {
	s, pub := newServer()
	h := s.Handler()
	payload := `{"type":"booking.discovered","source":"spiritus","payload":{"provider":"Aerolíneas del Plata","provider_ref":"PNR-77","type":"transport"}}`
	for i := 0; i < 2; i++ {
		w := do(t, h, http.MethodPost, "/internal/events", payload)
		if w.Code != http.StatusAccepted {
			t.Fatalf("intento %d: status %d body=%s", i, w.Code, w.Body.String())
		}
	}
	var trips struct{ Trips []struct{ Bookings []struct{ ProviderRef string `json:"provider_ref"` } `json:"bookings"` } `json:"trips"` }
	w := do(t, h, http.MethodGet, "/api/trips", "")
	_ = json.Unmarshal(w.Body.Bytes(), &trips)
	count := 0
	for _, tr := range trips.Trips {
		for _, b := range tr.Bookings {
			if b.ProviderRef == "PNR-77" {
				count++
			}
		}
	}
	if count != 1 {
		t.Fatalf("reserva duplicada: %d copias", count)
	}
	if len(pub.events) != 2 {
		t.Fatalf("esperaba 2 booking.confirmed (uno por reentrega), got %d", len(pub.events))
	}
}

func TestValidacionCrearReserva(t *testing.T) {
	s, _ := newServer()
	w := do(t, s.Handler(), http.MethodGet, "/api/trips", "")
	var out struct{ Trips []struct{ ID string } `json:"trips"` }
	_ = json.Unmarshal(w.Body.Bytes(), &out)
	w = do(t, s.Handler(), http.MethodPost, "/api/bookings", `{"trip_id":"no-existe","type":"transport","provider":"X","provider_ref":"R1"}`)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", w.Code)
	}
}
