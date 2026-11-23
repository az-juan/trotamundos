// Package views contiene los tipos y componentes templ del Rostro (Pater).
package views

type BookingView struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Provider    string `json:"provider"`
	ProviderRef string `json:"provider_ref"`
}

type TripView struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Status    string        `json:"status"`
	StartDate string        `json:"start_date,omitempty"`
	EndDate   string        `json:"end_date,omitempty"`
	Bookings  []BookingView `json:"bookings"`
}

type InsightView struct {
	Total          int            `json:"total"`
	Confirmations  int            `json:"confirmations"`
	Disruptions    int            `json:"disruptions"`
	ByType         map[string]int `json:"by_type"`
	LastEventAt    string         `json:"last_event_at,omitempty"`
	ResolutionRate float64        `json:"resolution_rate,omitempty"`
}

type ActivityView struct {
	Type       string
	Source     string
	OccurredAt string
	Summary    string
}

type DashboardData struct {
	Trips    []TripView
	Active   string
	Insights InsightView
	Activity []ActivityView
}

// FilterClass evita condicionales inline dentro de los atributos templ.
func FilterClass(active, target string) string {
	if active == target {
		return "filter-chip is-active"
	}
	return "filter-chip"
}

func StatusLabel(status string) string {
	switch status {
	case "upcoming":
		return "Próximo"
	case "active":
		return "En curso"
	case "archived":
		return "Archivado"
	case "confirmed":
		return "Confirmada"
	case "pending":
		return "Pendiente"
	case "discovered":
		return "Descubierta"
	default:
		return status
	}
}

func TypeLabel(kind string) string {
	switch kind {
	case "transport":
		return "Transporte"
	case "lodging":
		return "Alojamiento"
	case "car_rental":
		return "Auto"
	default:
		return kind
	}
}

// KeyValue filas genéricas para listas de datos (componente DataList).
type KeyValue struct{ Key, Value string }

type TravelerView struct {
	ID       string
	Name     string
	Document string
	Email    string
}

type TripDetailData struct {
	Trip      TripView
	Travelers []TravelerView
	Notice    string
}

type MailboxView struct {
	ID         string `json:"id"`
	Provider   string `json:"provider"`
	Email      string `json:"email"`
	LastScanAt string `json:"last_scan_at,omitempty"`
}

type SenderView struct {
	ID     string `json:"id"`
	Sender string `json:"sender"`
}

type PreferencesData struct {
	Locale          string `json:"locale"`
	Currency        string `json:"currency"`
	GpsRadius       string `json:"gps_radius_km,omitempty"`
	PreferredAgency string `json:"preferred_agency_id,omitempty"`
}

type InsightPageData struct {
	Insights InsightView
	Activity []ActivityView
}

type SharedData struct {
	Trip      TripView
	Travelers []TravelerView
}

type NearbyData struct {
	Lat    string
	Lon    string
	Radius string
	Trips  []TripView
}

func NavClass(active, target string) string {
	if active == target {
		return "nav-link is-current"
	}
	return "nav-link"
}

func ButtonClass(variant string) string {
	if variant == "" {
		return "btn"
	}
	return "btn " + variant
}

type ShareResult struct {
	Token     string `json:"token"`
	URL       string `json:"url"`
	ExpiresAt string `json:"expires_at"`
}

type UserIdentity struct {
	ID          string `json:"user_id"`
	Email       string `json:"email"`
	DisplayName string `json:"display_name"`
}

// GeoPoint es una reserva geolocalizada para el mapa SSR (QA-13).
type GeoPoint struct {
	TripID      string
	TripName    string
	Status      string
	Provider    string
	ProviderRef string
	Type        string
	Lat         float64
	Lon         float64
}

// MapPageData alimenta el mapa: filtro activo y conteos por estado.
type MapPageData struct {
	Status string
	Counts map[string]int
	Total  int
}
