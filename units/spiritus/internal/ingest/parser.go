// Package ingest captura reservas desde correos (FR-2) con reglas simples y auditables.
package ingest

import (
	"regexp"
	"strings"
)

// Draft es el borrador extraído de un correo de confirmación.
type Draft struct {
	Provider    string  `json:"provider"`
	ProviderRef string  `json:"provider_ref"`
	Type        string  `json:"type"`
	Confidence  float64 `json:"confidence"`
}

var refPattern = regexp.MustCompile(`(?i:pnr|localizador|confirmaci[oó]n|reserva|booking)[\s:·-]*([A-Z]{2,8}-[A-Z0-9]{2,8}|[A-Z0-9]{5,12})`)

var providers = []struct {
	name     string
	keywords []string
}{
	{"Aerolíneas del Plata", []string{"plata", "aerolineas", "aerolíneas"}},
	{"Hotel Presidente", []string{"presidente", "hotel"}},
	{"Ferrocarril del Este", []string{"ferrocarril", "tren"}},
	{"Rentadora del Sur", []string{"rentadora", "auto", "coche"}},
}

// Parse extrae Proveedor, referencia y tipo de un correo. No usa servicios externos.
func Parse(subject, body string) (Draft, bool) {
	text := subject + " " + body
	lower := strings.ToLower(text)
	m := refPattern.FindStringSubmatch(text)
	if m == nil || len(m) < 2 {
		return Draft{}, false
	}
	kind := "transport"
	switch {
	case strings.Contains(lower, "hotel") || strings.Contains(lower, "hospedaje") || strings.Contains(lower, "alojamiento"):
		kind = "lodging"
	case strings.Contains(lower, "rentadora") || strings.Contains(lower, "auto") || strings.Contains(lower, "coche"):
		kind = "car_rental"
	}
	provider := "Proveedor no identificado"
	for _, p := range providers {
		for _, kw := range p.keywords {
			if strings.Contains(lower, kw) {
				provider = p.name
				break
			}
		}
		if provider != "Proveedor no identificado" {
			break
		}
	}
	confidence := 0.72
	if provider != "Proveedor no identificado" {
		confidence = 0.94
	}
	return Draft{Provider: provider, ProviderRef: strings.ToUpper(m[1]), Type: kind, Confidence: confidence}, true
}
