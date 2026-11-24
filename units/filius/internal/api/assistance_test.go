package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"trotamundos/filius/internal/store"
	"trotamundos/shared/event"
)

type spyPublisher struct {
	events []event.Event
}

func (s *spyPublisher) Publish(_ any, e event.Event) error { return nil }

type memPublisher struct {
	events []event.Event
}

func (m *memPublisher) Publish(_ any, e event.Event) error { return nil }

func TestAssistanceCaseLifecycleAndOneTimeToken(t *testing.T) {
	st := store.New()
	st.Seed(time.Now())
	u := st.SeedUser("dev@trotamundos.app", "dev", "Viajero Dev")
	st.UpdatePreferences(u.ID, map[string]any{"preferred_agency_id": "agencia-condor"})

	srv := &Server{Store: st, Publisher: event.NoopPublisher{}}
	h := srv.Handler()

	// 1. Abrir caso con Idempotency-Key
	body := `{"user_id":"` + u.ID + `","reason":"complex_request","severity":"high"}`
	req := httptest.NewRequest(http.MethodPost, "/internal/assistance/cases", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Idempotency-Key", "idem-1930")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("esperaba 201, obtuvo %d: %s", w.Code, w.Body.String())
	}

	var handoff assistanceHandoff
	if err := json.Unmarshal(w.Body.Bytes(), &handoff); err != nil {
		t.Fatalf("JSON inválido: %v", err)
	}
	if handoff.Case.AgencyID != "agencia-condor" || handoff.ContextToken == "" {
		t.Fatalf("handoff incompleto: %+v", handoff)
	}

	// 2. Consumir el context_token de un solo uso (open -> handed_off)
	reqHandoff := httptest.NewRequest(http.MethodPost, "/internal/assistance/cases/"+handoff.Case.ID+"/handoff?context_token="+handoff.ContextToken, nil)
	wHandoff := httptest.NewRecorder()
	h.ServeHTTP(wHandoff, reqHandoff)
	if wHandoff.Code != http.StatusOK {
		t.Fatalf("handoff con token válido falló: %d", wHandoff.Code)
	}

	// 3. Reutilizar el mismo context_token debe fallar con 409 TOKEN_EXPIRED
	reqReplay := httptest.NewRequest(http.MethodPost, "/internal/assistance/cases/"+handoff.Case.ID+"/handoff?context_token="+handoff.ContextToken, nil)
	wReplay := httptest.NewRecorder()
	h.ServeHTTP(wReplay, reqReplay)
	if wReplay.Code != http.StatusConflict {
		t.Fatalf("un token de un solo uso no debe aceptarse dos veces: %d", wReplay.Code)
	}

	// 4. Resolver caso (handed_off -> resolved)
	reqResolve := httptest.NewRequest(http.MethodPost, "/internal/assistance/cases/"+handoff.Case.ID+"/resolve", nil)
	wResolve := httptest.NewRecorder()
	h.ServeHTTP(wResolve, reqResolve)
	if wResolve.Code != http.StatusOK {
		t.Fatalf("resolve falló: %d", wResolve.Code)
	}
}
