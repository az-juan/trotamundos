// Package api expone la API de Spiritus (Mente): ingesta, eventos e insights.
package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	"trotamundos/shared/event"
	"trotamundos/spiritus/internal/analytics"
	"trotamundos/spiritus/internal/ingest"
	"trotamundos/spiritus/internal/mail"
)

type Server struct {
	Store     *analytics.Memory
	Mail      *mail.Memory
	Publisher event.Publisher
}

type inboxRequest struct {
	Subject string `json:"subject"`
	Body    string `json:"body"`
	From    string `json:"from,omitempty"`
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func fail(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]any{"code": code, "message": message, "correlation_id": ""})
}

func (s *Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Get("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "unit": "spiritus"})
	})
	r.Post("/api/events", s.receiveEvent)
	r.Get("/api/events/list", s.listEvents)
	r.Get("/api/insights", s.insights)
	r.Post("/api/ingest/email", s.ingestEmail)
	s.moreRoutes(r)
	return r
}

func (s *Server) receiveEvent(w http.ResponseWriter, r *http.Request) {
	var e event.Event
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&e); err != nil {
		fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "evento inválido")
		return
	}
	if e.ID == "" {
		e.ID = "evt-" + e.CorrelationID
	}
	s.Store.Add(e)
	writeJSON(w, http.StatusAccepted, map[string]string{"status": "stored", "type": e.Type})
}

func (s *Server) listEvents(w http.ResponseWriter, r *http.Request) {
	limit := 12
	if n := r.URL.Query().Get("limit"); n != "" {
		if v, err := strconv.Atoi(n); err == nil && v > 0 {
			limit = v
		}
	}
	writeJSON(w, http.StatusOK, map[string]any{"events": s.Store.List(limit)})
}

func (s *Server) insights(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, s.Store.Summary())
}

func (s *Server) ingestEmail(w http.ResponseWriter, r *http.Request) {
	var in inboxRequest
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&in); err != nil {
		fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "correo inválido")
		return
	}
	if !s.Mail.Allowed(in.From) {
		fail(w, http.StatusForbidden, "FORBIDDEN", "remitente fuera de la whitelist")
		return
	}
	if s.Mail.Seen(in.Subject, in.Body) {
		writeJSON(w, http.StatusAccepted, map[string]any{"status": "duplicate", "reason": "sha256 ya procesado"})
		return
	}
	draft, ok := ingest.Parse(in.Subject, in.Body)
	if !ok {
		fail(w, http.StatusUnprocessableEntity, "NO_MATCH", "no se pudo extraer una reserva del correo")
		return
	}
	e := event.New(event.BookingDiscovered, "spiritus", map[string]any{
		"provider": draft.Provider, "provider_ref": draft.ProviderRef,
		"type": draft.Type, "confidence": draft.Confidence,
	})
	if s.Publisher != nil {
		if err := s.Publisher.Publish(r.Context(), e); err != nil {
			fail(w, http.StatusBadGateway, "UPSTREAM_UNAVAILABLE", "Filius no aceptó el evento: "+err.Error())
			return
		}
	}
	s.Store.Add(e)
	writeJSON(w, http.StatusAccepted, map[string]any{
		"status": "accepted", "draft": draft, "event": e.Type, "correlation_id": e.CorrelationID,
	})
}

// NewInMemoryServer arma un servidor listo para tests (sin infra externa).
func NewInMemoryServer(filiusURL string) *Server {
	return &Server{
		Store: analytics.New(),
		Mail:  mail.New(),
		Publisher: event.HTTPPublisher{
			BaseURL: strings.TrimRight(filiusURL, "/"),
			Path:    "/internal/events",
		},
	}
}
