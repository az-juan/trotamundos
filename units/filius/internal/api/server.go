// Package api expone la API REST de Filius (Núcleo) y el puerto interno de eventos.
package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"trotamundos/filius/internal/domain"
	"trotamundos/filius/internal/store"
	"trotamundos/shared/event"
)

type ctxKey int

const correlationKey ctxKey = 1

// Server es el adaptador de entrada HTTP del Núcleo.
type Server struct {
	Store     *store.Memory
	Publisher event.Publisher
}

func (s *Server) publish(ctx context.Context, kind string, payload map[string]any) {
	if s.Publisher == nil {
		return
	}
	_ = s.Publisher.Publish(ctx, event.New(kind, "filius", payload))
}

func correlation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Correlation-ID")
		if id == "" {
			b := make([]byte, 6)
			_, _ = rand.Read(b)
			id = hex.EncodeToString(b)
		}
		ctx := context.WithValue(r.Context(), correlationKey, id)
		w.Header().Set("X-Correlation-ID", id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func fail(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{
		"code":           code,
		"message":        message,
		"correlation_id": w.Header().Get("X-Correlation-ID"),
	})
}

// Handler arma el router con las dos aristas documentadas: REST con Pater y eventos con Spiritus.
func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(correlation)
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "unit": "filius"})
	})
	r.Route("/api", func(r chi.Router) {
		r.Get("/trips", s.listTrips)
		r.Post("/trips", s.createTrip)
		r.Get("/trips/{id}", s.getTrip)
		r.Post("/trips/{id}/archive", s.archiveTrip)
		r.Post("/bookings", s.createBooking)
	})
	r.Post("/internal/events", s.ingestEvent)
	s.moreRoutes(r)
	return r
}

func (s *Server) listTrips(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	writeJSON(w, http.StatusOK, map[string]any{"trips": s.Store.List(status)})
}

type tripRequest struct {
	Name      string `json:"name"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

func (s *Server) createTrip(w http.ResponseWriter, r *http.Request) {
	var in tripRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&in); err != nil {
		fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "cuerpo JSON inválido")
		return
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "el nombre del viaje es obligatorio")
		return
	}
	writeJSON(w, http.StatusCreated, s.Store.Create(in.Name, in.StartDate, in.EndDate))
}

func (s *Server) getTrip(w http.ResponseWriter, r *http.Request) {
	t, ok := s.Store.Get(chi.URLParam(r, "id"))
	if !ok {
		fail(w, http.StatusNotFound, "NOT_FOUND", "viaje no encontrado")
		return
	}
	writeJSON(w, http.StatusOK, t)
}

func (s *Server) archiveTrip(w http.ResponseWriter, r *http.Request) {
	t, ok := s.Store.Archive(chi.URLParam(r, "id"))
	if !ok {
		fail(w, http.StatusNotFound, "NOT_FOUND", "viaje no encontrado")
		return
	}
	s.publish(r.Context(), event.TripArchived, map[string]any{"trip_id": t.ID, "archived_at": t.ArchivedAt})
	writeJSON(w, http.StatusOK, t)
}

type bookingRequest struct {
	TripID      string `json:"trip_id"`
	Type        string `json:"type"`
	Provider    string `json:"provider"`
	ProviderRef string `json:"provider_ref"`
}

func (s *Server) createBooking(w http.ResponseWriter, r *http.Request) {
	var in bookingRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&in); err != nil {
		fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "cuerpo JSON inválido")
		return
	}
	if in.Type != "transport" && in.Type != "lodging" && in.Type != "car_rental" {
		fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "tipo de reserva inválido")
		return
	}
	b, err := s.Store.AddBooking(in.TripID, in.Type, in.Provider, in.ProviderRef)
	if err != nil {
		fail(w, http.StatusConflict, "CONFLICT", err.Error())
		return
	}
	s.publish(r.Context(), event.BookingConfirmed, map[string]any{
		"booking_id": b.ID, "trip_id": b.TripID, "user_id": "demo",
		"type": b.Type, "provider": b.Provider, "provider_ref": b.ProviderRef,
		"status": domain.Confirmed, "confirmed_at": time.Now().UTC().Format(time.RFC3339),
	})
	writeJSON(w, http.StatusCreated, b)
}

// ingestEvent recibe booking.discovered desde Spiritus (arista async del prototipo).
func (s *Server) ingestEvent(w http.ResponseWriter, r *http.Request) {
	var e event.Event
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&e); err != nil {
		fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "evento inválido")
		return
	}
	if e.Type != event.BookingDiscovered {
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "ignored", "type": e.Type})
		return
	}
	str := func(key string) string {
		v, _ := e.Payload[key].(string)
		return v
	}
	b, trip := s.Store.IngestDiscovered(str("provider"), str("provider_ref"), str("type"))
	s.publish(r.Context(), event.BookingConfirmed, map[string]any{
		"booking_id": b.ID, "trip_id": trip.ID, "user_id": "demo",
		"type": b.Type, "provider": b.Provider, "provider_ref": b.ProviderRef,
		"status": domain.Confirmed, "confirmed_at": time.Now().UTC().Format(time.RFC3339),
	})
	writeJSON(w, http.StatusAccepted, map[string]any{"status": "accepted", "booking_id": b.ID, "trip_id": trip.ID})
}
