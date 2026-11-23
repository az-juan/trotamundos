package ingest

import "testing"

func TestParseConfirmacionDeVuelo(t *testing.T) {
	d, ok := Parse(
		"Confirmación de su vuelo",
		"Estimado viajero: su reserva PNR-1930 con Aerolíneas del Plata ha sido confirmada.",
	)
	if !ok {
		t.Fatal("esperaba extraer la reserva")
	}
	if d.ProviderRef != "PNR-1930" || d.Provider != "Aerolíneas del Plata" || d.Type != "transport" {
		t.Fatalf("borrador incorrecto: %+v", d)
	}
	if d.Confidence < 0.9 {
		t.Errorf("confidence = %v, esperaba alto", d.Confidence)
	}
}

func TestParseHotel(t *testing.T) {
	d, ok := Parse("Hotel Presidente", "Confirmación reserva HT-4471 para su estadía en Buenos Aires")
	if !ok || d.Type != "lodging" || d.ProviderRef != "HT-4471" {
		t.Fatalf("borrador incorrecto: %+v ok=%v", d, ok)
	}
}

func TestParseSinReferencia(t *testing.T) {
	if _, ok := Parse("Buenos días", "Gracias por elegirnos"); ok {
		t.Fatal("no debería extraer nada sin referencia")
	}
}
