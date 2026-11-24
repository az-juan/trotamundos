// Package assistant implementa el módulo de Asistente Virtual en Pater (Inicio parcial Fase 1, ADR-109/110/113).
// Opera con un clasificador de intenciones determinista detrás del puerto IntentClassifier,
// redacta PII (QA-9), guarda la sesión efímera en Redis con TTL (sin persistir transcripción en SQL)
// y delega la apertura/seguimiento de casos complejos en Filius por REST (ADR-110).
package assistant

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"trotamundos/pater/views"
	"trotamundos/shared/cache"
)

type Intent string

const (
	IntentStatusQuery        Intent = "status_query"
	IntentRescheduleGuidance Intent = "reschedule_guidance"
	IntentMinorIssue         Intent = "minor_issue"
	IntentComplexHandoff     Intent = "complex_handoff"
	IntentOutOfScope         Intent = "out_of_scope"
)

type Classification struct {
	Intent   Intent
	Reason   string
	Severity string
}

// IntentClassifier es el puerto de salida para clasificar la intención del viajero (ADR-109).
type IntentClassifier interface {
	Classify(ctx context.Context, redactedMessage string) Classification
}

var (
	emailRegex = regexp.MustCompile(`[a-zA-Z0-9._%+\-]+@[a-zA-Z0-9.\-]+\.[a-zA-Z]{2,}`)
	docRegex   = regexp.MustCompile(`\b\d{7,12}[A-Za-z]?\b`)
)

// RedactPII suprime correos y documentos antes de evaluar el mensaje (QA-9 · Datensparsamkeit).
func RedactPII(input string) string {
	out := emailRegex.ReplaceAllString(input, "[EMAIL_REDACTADO]")
	return docRegex.ReplaceAllString(out, "[DOC_REDACTADO]")
}

// RuleBasedClassifier es el adaptador determinista del MVP de Fase 1 (ADR-109 Opción 4 / ADR-113).
type RuleBasedClassifier struct{}

func (RuleBasedClassifier) Classify(_ context.Context, msg string) Classification {
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "agencia") || strings.Contains(lower, "reembolso") ||
		strings.Contains(lower, "reclamo") || strings.Contains(lower, "complej") ||
		strings.Contains(lower, "cancelaron") || strings.Contains(lower, "indemnizaci") ||
		strings.Contains(lower, "derivar"):
		reason := "complex_request"
		if strings.Contains(lower, "reclamo") || strings.Contains(lower, "indemnizaci") {
			reason = "complaint"
		}
		return Classification{Intent: IntentComplexHandoff, Reason: reason, Severity: "high"}
	case strings.Contains(lower, "reprogram") || strings.Contains(lower, "cambiar fecha") ||
		strings.Contains(lower, "cambio de vuelo") || strings.Contains(lower, "modificar"):
		return Classification{Intent: IntentRescheduleGuidance, Reason: "reschedule_failed", Severity: "medium"}
	case strings.Contains(lower, "estado") || strings.Contains(lower, "pnr") ||
		strings.Contains(lower, "confirmad") || strings.Contains(lower, "vuelo") ||
		strings.Contains(lower, "reserva") || strings.Contains(lower, "viaje") ||
		strings.Contains(lower, "itinerario"):
		return Classification{Intent: IntentStatusQuery, Severity: "low"}
	case strings.Contains(lower, "equipaje") || strings.Contains(lower, "valija") ||
		strings.Contains(lower, "check-in") || strings.Contains(lower, "checkin") ||
		strings.Contains(lower, "puerta") || strings.Contains(lower, "embarque") ||
		strings.Contains(lower, "asiento") || strings.Contains(lower, "documento"):
		return Classification{Intent: IntentMinorIssue, Severity: "low"}
	default:
		return Classification{Intent: IntentOutOfScope, Severity: "low"}
	}
}

var (
	sessOnce  sync.Once
	sessCache *cache.Store
)

func sessionStore() *cache.Store {
	sessOnce.Do(func() {
		sessCache = cache.New(os.Getenv("REDIS_ADDR"))
	})
	return sessCache
}

// LoadTurns recupera los turnos efímeros de la sesión desde Redis (TTL 30 min, sin persistir en PostgreSQL).
func LoadTurns(sessionID string) []views.AssistantTurnView {
	if sessionID == "" {
		return nil
	}
	raw, ok := sessionStore().Get("assistant:session:" + sessionID)
	if !ok || raw == "" {
		return nil
	}
	var turns []views.AssistantTurnView
	_ = json.Unmarshal([]byte(raw), &turns)
	return turns
}

// AppendTurn agrega un turno a la sesión efímera en Redis con TTL de 30 minutos (ADR-109).
func AppendTurn(sessionID string, turn views.AssistantTurnView) []views.AssistantTurnView {
	turns := LoadTurns(sessionID)
	turns = append(turns, turn)
	if len(turns) > 12 {
		turns = turns[len(turns)-12:]
	}
	if sessionID != "" {
		if encoded, err := json.Marshal(turns); err == nil {
			sessionStore().Set("assistant:session:"+sessionID, string(encoded), 30*time.Minute)
		}
	}
	return turns
}

type HandoffResponse struct {
	Case                  views.AssistanceCaseView `json:"case"`
	DeepLinkURL           string                   `json:"deep_link_url"`
	ContextToken          string                   `json:"context_token"`
	ContextTokenExpiresAt string                   `json:"context_token_expires_at"`
}

func filiusBaseURL() string {
	if u := strings.TrimSpace(os.Getenv("FILIUS_URL")); u != "" {
		return strings.TrimRight(u, "/")
	}
	return "http://127.0.0.1:8081"
}

func newIdempotencyKey() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)
}

// NotifyFiliusHandoff invoca POST /internal/assistance/cases en Filius (arista REST Pater→Filius, ADR-110).
func NotifyFiliusHandoff(ctx context.Context, userID, tripID, bookingID, reason, severity, agencyID, correlationID string) (HandoffResponse, error) {
	payload, _ := json.Marshal(map[string]string{
		"user_id":    userID,
		"trip_id":    tripID,
		"booking_id": bookingID,
		"reason":     reason,
		"severity":   severity,
		"agency_id":  agencyID,
	})
	reqCtx, cancel := context.WithTimeout(ctx, 1500*time.Millisecond)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, filiusBaseURL()+"/internal/assistance/cases", bytes.NewReader(payload))
	if err != nil {
		return HandoffResponse{}, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", newIdempotencyKey())
	if correlationID != "" {
		req.Header.Set("X-Correlation-ID", correlationID)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return HandoffResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return HandoffResponse{}, fmt.Errorf("filius status %d", resp.StatusCode)
	}
	var out HandoffResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return HandoffResponse{}, err
	}
	return out, nil
}

// NotifyFiliusTier1 notifica a Filius una resolución autónoma de Nivel 1 para emitir assistance.resolved en NATS (QA-17).
func NotifyFiliusTier1(ctx context.Context, userID string, intent Intent, correlationID string) {
	payload, _ := json.Marshal(map[string]string{"user_id": userID, "intent": string(intent)})
	reqCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, filiusBaseURL()+"/internal/assistance/tier1", bytes.NewReader(payload))
	if err != nil {
		return
	}
	req.Header.Set("Content-Type", "application/json")
	if correlationID != "" {
		req.Header.Set("X-Correlation-ID", correlationID)
	}
	if resp, err := http.DefaultClient.Do(req); err == nil {
		_ = resp.Body.Close()
	}
}

// NotifyFiliusCaseAction propaga handoff o resolve a Filius por REST para publicar eventos NATS hacia Spiritus.
func NotifyFiliusCaseAction(ctx context.Context, caseID, action, correlationID string) {
	reqCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, filiusBaseURL()+"/internal/assistance/cases/"+caseID+"/"+action, nil)
	if err != nil {
		return
	}
	if correlationID != "" {
		req.Header.Set("X-Correlation-ID", correlationID)
	}
	if resp, err := http.DefaultClient.Do(req); err == nil {
		_ = resp.Body.Close()
	}
}
