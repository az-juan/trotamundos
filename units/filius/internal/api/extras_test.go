package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestVerificarCredencialesYPreferencias(t *testing.T) {
	s, _ := newServer()
	s.Store.SeedUser("dev@trotamundos.app", "dev", "Viajero Dev")
	h := s.Handler()
	w := do(t, h, http.MethodPost, "/internal/auth/verify", `{"email":"dev@trotamundos.app","password":"dev"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("verify status=%d body=%s", w.Code, w.Body.String())
	}
	var auth struct {
		UserID string `json:"user_id"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &auth)
	w = do(t, h, http.MethodPatch, "/internal/users/"+auth.UserID+"/preferences", `{"currency":"ARS"}`)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "ARS") {
		t.Fatalf("patch preferencias: %d %s", w.Code, w.Body.String())
	}
}

func TestCompartirItinerarioSoloLectura(t *testing.T) {
	s, _ := newServer()
	s.Store.Seed(time.Now())
	h := s.Handler()
	trips := s.Store.List("")
	id := trips[0].ID
	w := do(t, h, http.MethodPost, "/api/trips/"+id+"/share", `{}`)
	if w.Code != http.StatusCreated {
		t.Fatalf("share status=%d body=%s", w.Code, w.Body.String())
	}
	var sh struct {
		Token string `json:"token"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &sh)
	w = do(t, h, http.MethodGet, "/share/"+sh.Token, "")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"shared":true`) {
		t.Fatalf("resolve: %d %s", w.Code, w.Body.String())
	}
}

func TestWebhookRequiereFirmaHMAC(t *testing.T) {
	s, _ := newServer()
	s.Store.Seed(time.Now())
	h := s.Handler()
	payload := `{"provider_ref":"PNR-1930","status":"disrupted","message":"Vuelo demorado"}`
	req := httptest.NewRequest(http.MethodPost, "/api/webhooks/aerolinea", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("sin firma: status=%d", w.Code)
	}
	_, _ = s.Store.UpdateBookingStatus("PNR-1930", "confirmed")
	mac := hmac.New(sha256.New, []byte("dev-secret"))
	_, _ = mac.Write([]byte(payload))
	req = httptest.NewRequest(http.MethodPost, "/api/webhooks/aerolinea", strings.NewReader(payload))
	req.Header.Set("X-Signature", hex.EncodeToString(mac.Sum(nil)))
	w = httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusAccepted {
		t.Fatalf("con firma: status=%d body=%s", w.Code, w.Body.String())
	}
	if !strings.Contains(w.Body.String(), "disrupted") {
		t.Error("el webhook no actualizó el estado")
	}
}

func TestNearby(t *testing.T) {
	s, _ := newServer()
	s.Store.Seed(time.Now())
	for _, tr := range s.Store.List("") {
		for _, b := range tr.Bookings {
			s.Store.SetBookingCoords(b.ProviderRef, -34.6037, -58.3816)
		}
	}
	w := do(t, s.Handler(), http.MethodGet, "/api/trips/nearby?lat=-34.60&lon=-58.38&radius_km=5", "")
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"trips"`) {
		t.Fatalf("nearby: %d %s", w.Code, w.Body.String())
	}
}
