package assistant

import (
	"context"
	"strings"
	"testing"

	"trotamundos/pater/views"
)

func TestRuleBasedClassifierAndPIIRedaction(t *testing.T) {
	raw := "Hola, mi correo es viajero@trotamundos.app y DNI 12345678Z, necesito reembolso con mi agencia"
	redacted := RedactPII(raw)
	if strings.Contains(redacted, "viajero@trotamundos.app") || strings.Contains(redacted, "12345678Z") {
		t.Fatalf("PII no fue redactada: %s", redacted)
	}

	clf := RuleBasedClassifier{}
	cases := []struct {
		msg    string
		expect Intent
	}{
		{"¿Cuál es el estado de mi reserva PNR-1930?", IntentStatusQuery},
		{"Necesito reprogramar mi vuelo a Madrid", IntentRescheduleGuidance},
		{"¿Cuánto equipaje de mano puedo llevar en cabina?", IntentMinorIssue},
		{redacted, IntentComplexHandoff},
		{"¿Cuál es la receta del tango?", IntentOutOfScope},
	}
	for _, tc := range cases {
		got := clf.Classify(context.Background(), tc.msg)
		if got.Intent != tc.expect {
			t.Errorf("mensaje %q: esperado %s, obtuvo %s", tc.msg, tc.expect, got.Intent)
		}
	}
}

func TestSessionStoreEphemeralTurns(t *testing.T) {
	sid := "test-sess-1930"
	turns := AppendTurn(sid, views.AssistantTurnView{
		UserMessage:   "Estado PNR-1930",
		Intent:        string(IntentStatusQuery),
		Answer:        "Confirmada",
		Tier1Resolved: true,
	})
	if len(turns) != 1 || turns[0].Intent != "status_query" {
		t.Fatalf("turno efímero no almacenado: %+v", turns)
	}
}
