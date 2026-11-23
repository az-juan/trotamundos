package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestIngestaPublicaEventoHaciaFilius(t *testing.T) {
	var recibido map[string]any
	filius := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/internal/events" {
			t.Errorf("ruta inesperada: %s", r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&recibido)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer filius.Close()

	s := NewInMemoryServer(filius.URL)
	body := `{"subject":"Confirmación de vuelo","body":"reserva PNR-77 con Aerolíneas del Plata"}`
	req := httptest.NewRequest(http.MethodPost, "/api/ingest/email", strings.NewReader(body))
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if recibido["type"] != "booking.discovered" {
		t.Fatalf("evento enviado incorrecto: %+v", recibido)
	}
	sum := s.Store.Summary()
	if sum.ByType["booking.discovered"] != 1 {
		t.Fatalf("resumen incorrecto: %+v", sum)
	}
}

func TestInsightsVacio(t *testing.T) {
	s := NewInMemoryServer("")
	w := httptest.NewRecorder()
	s.Handler().ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/api/insights", nil))
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"total":0`) {
		t.Fatalf("respuesta inesperada: %d %s", w.Code, w.Body.String())
	}
}
