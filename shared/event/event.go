// Package event define el contrato mínimo de eventos de la arista Filius↔Spiritus.
// En el prototipo Fase 0 el transporte es HTTP; la topología documentada sigue siendo NATS JetStream.
package event

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	BookingDiscovered = "booking.discovered"
	BookingConfirmed  = "booking.confirmed"
	BookingCancelled  = "booking.cancelled"
	TripArchived      = "trip.archived"
	TripDisrupted     = "trip.disrupted"
)

// Event es el envelope CloudEvents simplificado usado por las 3 unidades.
type Event struct {
	ID            string         `json:"id"`
	Type          string         `json:"type"`
	Source        string         `json:"source"`
	CorrelationID string         `json:"correlation-id"`
	OccurredAt    string         `json:"occurred-at"`
	Payload       map[string]any `json:"payload"`
}

func token(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

// New construye un evento con id, correlation-id y timestamp propios.
func New(kind, source string, payload map[string]any) Event {
	return Event{
		ID:            token(12),
		Type:          kind,
		Source:        source,
		CorrelationID: token(8),
		OccurredAt:    time.Now().UTC().Format(time.RFC3339),
		Payload:       payload,
	}
}

// Publisher es el puerto de salida (ADR-002) para publicar hechos de negocio.
type Publisher interface {
	Publish(ctx context.Context, e Event) error
}

// NoopPublisher se usa en tests y en unidades que no emiten eventos.
type NoopPublisher struct{}

func (NoopPublisher) Publish(context.Context, Event) error { return nil }

// HTTPPublisher es el adaptador del prototipo: envía el evento por HTTP.
// Path permite apuntar a /api/events (Spiritus) o /internal/events (Filius).
type HTTPPublisher struct {
	BaseURL string
	Path    string
	Client  *http.Client
}

func (p HTTPPublisher) Publish(ctx context.Context, e Event) error {
	if p.BaseURL == "" {
		return nil
	}
	body, err := json.Marshal(e)
	if err != nil {
		return err
	}
	path := p.Path
	if path == "" {
		path = "/api/events"
	}
	url := strings.TrimRight(p.BaseURL, "/") + path
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Correlation-ID", e.CorrelationID)
	client := p.Client
	if client == nil {
		client = http.DefaultClient
	}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("publicar %s: status %d", e.Type, resp.StatusCode)
	}
	return nil
}
