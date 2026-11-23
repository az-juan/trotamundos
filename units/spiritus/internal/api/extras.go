package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// moreRoutes expone buzones, whitelist y el stream SSE del read-model.
func (s *Server) moreRoutes(r chi.Router) {
	r.Get("/api/mailboxes", s.listMailboxes)
	r.Post("/api/mailboxes", s.addMailbox)
	r.Get("/api/whitelists", s.listSenders)
	r.Post("/api/whitelists", s.addSender)
	r.Delete("/api/whitelists/{id}", s.removeSender)
	r.Get("/api/events/stream", s.eventStream)
	r.Get("/mailboxes", s.listMailboxes)
	r.Get("/whitelists", s.listSenders)
}

func (s *Server) listMailboxes(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": s.Mail.Mailboxes()})
}

func (s *Server) addMailbox(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Provider string `json:"provider"`
		Email    string `json:"email"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Email == "" {
		fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "email obligatorio")
		return
	}
	if in.Provider != "gmail" && in.Provider != "imap" {
		fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "provider debe ser gmail o imap")
		return
	}
	writeJSON(w, http.StatusCreated, s.Mail.AddMailbox(in.Provider, in.Email))
}

func (s *Server) listSenders(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": s.Mail.Senders()})
}

func (s *Server) addSender(w http.ResponseWriter, r *http.Request) {
	var in struct {
		Sender string `json:"sender"`
	}
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil || in.Sender == "" {
		fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "remitente obligatorio")
		return
	}
	writeJSON(w, http.StatusCreated, s.Mail.AddSender(in.Sender))
}

func (s *Server) removeSender(w http.ResponseWriter, r *http.Request) {
	if !s.Mail.RemoveSender(chi.URLParam(r, "id")) {
		fail(w, http.StatusNotFound, "NOT_FOUND", "remitente no encontrado")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// eventStream mantiene el read-model abierto para clientes SSE (arista Spiritus→Pater).
// El navegador usa polling HTMX; este endpoint sirve a apps nativas y a Nginx.
func (s *Server) eventStream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming no soportado", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	fmt.Fprint(w, "event: ready\ndata: {}\n\n")
	flusher.Flush()
	last := len(s.Store.List(1000))
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			current := s.Store.List(1000)
			if len(current) > last {
				e := current[0]
				payload, _ := json.Marshal(e)
				fmt.Fprintf(w, "event: %s\ndata: %s\nid: %s\n\n", e.Type, payload, e.ID)
				flusher.Flush()
				last = len(current)
			}
		}
	}
}
