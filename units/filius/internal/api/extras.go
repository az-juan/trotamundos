package api

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"trotamundos/shared/event"
)

// moreRoutes añade las rutas documentadas en OpenAPI y los contratos internos.
// Las rutas sin /api son alias del contrato público (Pater usa /api por el Gateway).
func (s *Server) moreRoutes(r chi.Router) {
	r.Post("/internal/auth/verify", s.verifyCredentials)
	r.Get("/internal/users/{id}/preferences", s.getPreferences)
	r.Patch("/internal/users/{id}/preferences", s.patchPreferences)
	r.Get("/share/{token}", s.resolveShare)
	r.Post("/api/trips/{id}/share", s.shareTrip)
	r.Get("/api/trips/nearby", s.nearby)
	r.Get("/api/trips/{id}/travelers", s.travelers)
	r.Get("/api/bookings", s.listBookings)
	r.Get("/api/bookings/{id}", s.getBooking)
	r.Post("/api/webhooks/{provider}", s.webhook)

	// Alias del contrato público documentado en fase_0/docs/api/openapi.yaml
	r.Get("/trips", s.listTrips)
	r.Post("/trips", s.createTrip)
	r.Get("/trips/nearby", s.nearby)
	r.Get("/trips/{id}", s.getTrip)
	r.Post("/trips/{id}/share", s.shareTrip)
	r.Get("/bookings", s.listBookings)
	r.Post("/bookings", s.createBooking)
	r.Post("/webhooks/{provider}", s.webhook)
}

func (s *Server) verifyCredentials(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "credenciales inválidas")
		return
	}
	u, ok := s.Store.VerifyCredentials(in.Email, in.Password)
	if !ok {
		fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "credenciales incorrectas")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"user_id": u.ID, "email": u.Email, "display_name": u.DisplayName})
}

func (s *Server) getPreferences(w http.ResponseWriter, r *http.Request) {
	prefs := s.Store.Preferences(chi.URLParam(r, "id"))
	if prefs == nil {
		fail(w, http.StatusNotFound, "NOT_FOUND", "usuario no encontrado")
		return
	}
	writeJSON(w, http.StatusOK, prefs)
}

func (s *Server) patchPreferences(w http.ResponseWriter, r *http.Request) {
	var patch map[string]any
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "preferencias inválidas")
		return
	}
	out := s.Store.UpdatePreferences(chi.URLParam(r, "id"), patch)
	if out == nil {
		fail(w, http.StatusNotFound, "NOT_FOUND", "usuario no encontrado")
		return
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) shareTrip(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, ok := s.Store.Get(id); !ok {
		fail(w, http.StatusNotFound, "NOT_FOUND", "viaje no encontrado")
		return
	}
	sh := s.Store.Share(id, 7*24*time.Hour)
	writeJSON(w, http.StatusCreated, map[string]any{
		"token": sh.Token, "url": "/shared/" + sh.Token,
		"expires_at": sh.ExpiresAt, "permission": sh.Permission,
	})
}

func (s *Server) resolveShare(w http.ResponseWriter, r *http.Request) {
	t, ok := s.Store.ResolveShare(chi.URLParam(r, "token"))
	if !ok {
		fail(w, http.StatusNotFound, "NOT_FOUND", "enlace vencido o inválido")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"trip": t, "travelers": s.Store.Travelers(t.ID), "shared": true,
	})
}

func (s *Server) nearby(w http.ResponseWriter, r *http.Request) {
	lat, e1 := strconv.ParseFloat(r.URL.Query().Get("lat"), 64)
	lon, e2 := strconv.ParseFloat(r.URL.Query().Get("lon"), 64)
	if e1 != nil || e2 != nil {
		fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "lat y lon son obligatorios")
		return
	}
	radius := 50.0
	if v, err := strconv.ParseFloat(r.URL.Query().Get("radius_km"), 64); err == nil && v > 0 {
		radius = v
	}
	writeJSON(w, http.StatusOK, map[string]any{"trips": s.Store.Nearby(lat, lon, radius)})
}

func (s *Server) listBookings(w http.ResponseWriter, r *http.Request) {
	tripID := r.URL.Query().Get("trip_id")
	all := []any{}
	for _, t := range s.Store.List("") {
		if tripID != "" && t.ID != tripID {
			continue
		}
		for _, b := range t.Bookings {
			all = append(all, b)
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": all})
}

func (s *Server) getBooking(w http.ResponseWriter, r *http.Request) {
	wanted := chi.URLParam(r, "id")
	for _, t := range s.Store.List("") {
		for _, b := range t.Bookings {
			if b.ID == wanted {
				writeJSON(w, http.StatusOK, b)
				return
			}
		}
	}
	fail(w, http.StatusNotFound, "NOT_FOUND", "reserva no encontrada")
}

// webhook verifica la firma HMAC-SHA256 antes de mutar dominio (ADR-007).
func (s *Server) webhook(w http.ResponseWriter, r *http.Request) {
	raw, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 1<<20))
	if err != nil {
		fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "body ilegible")
		return
	}
	secret := os.Getenv("WEBHOOK_SECRET")
	if secret == "" {
		secret = "dev-secret"
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(raw)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(strings.TrimPrefix(r.Header.Get("X-Signature"), "sha256=")), []byte(expected)) {
		fail(w, http.StatusUnauthorized, "UNAUTHORIZED", "firma HMAC inválida")
		return
	}
	var in struct {
		ProviderRef string  `json:"provider_ref"`
		Status      string  `json:"status"`
		Lat         float64 `json:"lat"`
		Lon         float64 `json:"lon"`
		Message     string  `json:"message"`
	}
	if err := json.Unmarshal(raw, &in); err != nil || in.ProviderRef == "" {
		fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "payload inválido")
		return
	}
	if in.Lat != 0 || in.Lon != 0 {
		s.Store.SetBookingCoords(in.ProviderRef, in.Lat, in.Lon)
	}
	b, ok := s.Store.UpdateBookingStatus(in.ProviderRef, in.Status)
	if !ok {
		fail(w, http.StatusConflict, "CONFLICT", "transición no permitida o reserva inexistente")
		return
	}
	kind := event.BookingConfirmed
	if b.Status == "disrupted" {
		kind = event.TripDisrupted
	} else if b.Status == "cancelled" {
		kind = event.BookingCancelled
	}
	s.publish(r.Context(), kind, map[string]any{
		"booking_id": b.ID, "trip_id": b.TripID, "user_id": "demo", "provider_ref": b.ProviderRef,
		"status": b.Status, "message": in.Message, "severity": "medium",
	})
	writeJSON(w, http.StatusAccepted, map[string]any{"status": "accepted", "booking": b})
}

func (s *Server) travelers(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if _, ok := s.Store.Get(id); !ok {
		fail(w, http.StatusNotFound, "NOT_FOUND", "viaje no encontrado")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": s.Store.Travelers(id)})
}
