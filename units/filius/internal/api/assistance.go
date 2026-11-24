// Package api implementa el agregado AssistanceCase y los eventos de derivación (Fase 1, ADR-110/113).
package api

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"trotamundos/filius/internal/store"
	"trotamundos/shared/cache"
	"trotamundos/shared/event"
)

var (
	cacheOnce   sync.Once
	sharedCache *cache.Store
)

func getCache() *cache.Store {
	cacheOnce.Do(func() {
		sharedCache = cache.New(os.Getenv("REDIS_ADDR"))
	})
	return sharedCache
}

func randomToken(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("tok-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

type openCaseInput struct {
	UserID    string `json:"user_id"`
	TripID    string `json:"trip_id"`
	BookingID string `json:"booking_id"`
	Reason    string `json:"reason"`
	Severity  string `json:"severity"`
	AgencyID  string `json:"agency_id"`
}

type assistanceHandoff struct {
	Case                  store.AssistanceCase `json:"case"`
	DeepLinkURL           string               `json:"deep_link_url"`
	ContextToken          string               `json:"context_token"`
	ContextTokenExpiresAt string               `json:"context_token_expires_at"`
}

func validReason(r string) string {
	switch r {
	case "complex_request", "reschedule_failed", "complaint", "other":
		return r
	default:
		return "complex_request"
	}
}

func validSeverity(sev string) string {
	switch sev {
	case "low", "medium", "high":
		return sev
	default:
		return "high"
	}
}

func (s *Server) openAssistanceCase(w http.ResponseWriter, r *http.Request) {
	cStore := getCache()
	idemKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idemKey != "" {
		if cached, ok := cStore.Get("idem:assistance:" + idemKey); ok {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			w.WriteHeader(http.StatusCreated)
			_, _ = w.Write([]byte(cached))
			return
		}
	}

	var in openCaseInput
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 64<<10)).Decode(&in); err != nil {
		fail(w, http.StatusBadRequest, "VALIDATION_ERROR", "solicitud de caso inválida")
		return
	}
	if strings.TrimSpace(in.UserID) == "" {
		in.UserID = "00000000-0000-0000-0000-000000000001"
	}
	in.Reason = validReason(in.Reason)
	in.Severity = validSeverity(in.Severity)

	c := s.Store.OpenCase(in.UserID, in.TripID, in.BookingID, in.Reason, in.Severity, in.AgencyID)
	ctxToken := "ctx_" + randomToken(10)
	expiresAt := time.Now().Add(15 * time.Minute).UTC().Format(time.RFC3339)

	// El token efímero en Redis guarda solo el contexto mínimo del caso, nunca la transcripción (ADR-110, QA-9).
	minCtx, _ := json.Marshal(map[string]string{
		"case_id":   c.ID,
		"reason":    c.Reason,
		"severity":  c.Severity,
		"agency_id": c.AgencyID,
	})
	cStore.Set("assistance:token:"+ctxToken, string(minCtx), 15*time.Minute)

	deepLink := fmt.Sprintf("https://agencias.trotamundos.app/%s/handoff?case_id=%s&context_token=%s", c.AgencyID, c.ID, ctxToken)
	out := assistanceHandoff{
		Case:                  c,
		DeepLinkURL:           deepLink,
		ContextToken:          ctxToken,
		ContextTokenExpiresAt: expiresAt,
	}

	s.publish(r.Context(), event.AssistanceHandoffRequested, map[string]any{
		"case_id":                  c.ID,
		"user_id":                  c.UserID,
		"trip_id":                  c.TripID,
		"booking_id":               c.BookingID,
		"agency_id":                c.AgencyID,
		"reason":                   c.Reason,
		"severity":                 c.Severity,
		"context_token_expires_at": expiresAt,
	})

	raw, _ := json.Marshal(out)
	if idemKey != "" {
		cStore.Set("idem:assistance:"+idemKey, string(raw), 24*time.Hour)
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	_, _ = w.Write(raw)
}

func (s *Server) listAssistanceCases(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"items": s.Store.ListCases()})
}

func (s *Server) getAssistanceCase(w http.ResponseWriter, r *http.Request) {
	c, ok := s.Store.GetCase(chi.URLParam(r, "id"))
	if !ok {
		fail(w, http.StatusNotFound, "NOT_FOUND", "caso de asistencia no encontrado")
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) handoffAssistanceCase(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	token := strings.TrimSpace(r.URL.Query().Get("context_token"))
	if token != "" {
		if _, ok := getCache().GetDel("assistance:token:" + token); !ok {
			fail(w, http.StatusConflict, "TOKEN_EXPIRED", "el context_token ya fue utilizado o expiró")
			return
		}
	}
	c, ok := s.Store.UpdateCaseStatus(id, "handed_off")
	if !ok {
		fail(w, http.StatusNotFound, "NOT_FOUND", "caso de asistencia no encontrado")
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) resolveAssistanceCase(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	c, ok := s.Store.UpdateCaseStatus(id, "resolved")
	if !ok {
		fail(w, http.StatusNotFound, "NOT_FOUND", "caso de asistencia no encontrado")
		return
	}
	s.publish(r.Context(), event.AssistanceResolved, map[string]any{
		"case_id":        c.ID,
		"user_id":        c.UserID,
		"outcome":        "resolved_by_agency",
		"tier1_resolved": false,
		"resolved_at":    c.ResolvedAt,
	})
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) recordTier1Resolution(w http.ResponseWriter, r *http.Request) {
	var in struct {
		UserID string `json:"user_id"`
		Intent string `json:"intent"`
	}
	_ = json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&in)
	if in.UserID == "" {
		in.UserID = "00000000-0000-0000-0000-000000000001"
	}
	s.publish(r.Context(), event.AssistanceResolved, map[string]any{
		"case_id":        "tier1-" + randomToken(6),
		"user_id":        in.UserID,
		"intent":         in.Intent,
		"outcome":        "resolved_by_assistant",
		"tier1_resolved": true,
		"resolved_at":    time.Now().UTC().Format(time.RFC3339),
	})
	writeJSON(w, http.StatusAccepted, map[string]any{"status": "recorded", "tier1_resolved": true})
}
