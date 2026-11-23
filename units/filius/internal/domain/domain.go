// Package domain contiene el modelo puro del Núcleo (sin infraestructura, ADR-002).
package domain

import "time"

const (
	Discovered = "discovered"
	Pending    = "pending"
	Confirmed  = "confirmed"
	Disrupted  = "disrupted"
	Cancelled  = "cancelled"
	Archived   = "archived"
)

// Booking es una reserva de transporte o alojamiento.
type Booking struct {
	ID          string         `json:"id"`
	TripID      string         `json:"trip_id"`
	Type        string         `json:"type"`
	Status      string         `json:"status"`
	Provider    string         `json:"provider"`
	ProviderRef string         `json:"provider_ref"`
	Details     map[string]any `json:"details"`
	CreatedAt   string         `json:"created_at"`
}

// Trip agrupa reservas y viajeros (FR-5).
type Trip struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Status     string    `json:"status"`
	StartDate  string    `json:"start_date,omitempty"`
	EndDate    string    `json:"end_date,omitempty"`
	CreatedAt  string    `json:"created_at"`
	ArchivedAt string    `json:"archived_at,omitempty"`
	Bookings   []Booking `json:"bookings"`
}

func now() string { return time.Now().UTC().Format(time.RFC3339) }

func NewTrip(name, start, end string) Trip {
	return Trip{
		ID:        newID(),
		Name:      name,
		Status:    "upcoming",
		StartDate: start,
		EndDate:   end,
		CreatedAt: now(),
		Bookings:  []Booking{},
	}
}

func NewBooking(tripID, kind, provider, ref string) Booking {
	return Booking{
		ID:          newID(),
		TripID:      tripID,
		Type:        kind,
		Status:      Discovered,
		Provider:    provider,
		ProviderRef: ref,
		Details:     map[string]any{},
		CreatedAt:   now(),
	}
}

func (t *Trip) AddBooking(b Booking) {
	b.TripID = t.ID
	t.Bookings = append(t.Bookings, b)
}

func (t *Trip) Archive() {
	t.Status = Archived
	t.ArchivedAt = now()
	for i := range t.Bookings {
		t.Bookings[i].Status = Archived
	}
}

// ArchiveCompleted aplica FR-5: archiva viajes cuya fecha de fin ya pasó.
func ArchiveCompleted(trips []Trip, today time.Time) []Trip {
	var archived []Trip
	for i := range trips {
		if trips[i].Status == Archived || trips[i].EndDate == "" {
			continue
		}
		end, err := time.Parse("2006-01-02", trips[i].EndDate)
		if err == nil && end.Before(today) {
			trips[i].Archive()
			archived = append(archived, trips[i])
		}
	}
	return archived
}

// CanTransition implementa la state machine del Booking (modelo-datos §6).
func CanTransition(from, to string) bool {
	allowed := map[string][]string{
		Discovered: {Pending, Confirmed, Cancelled, Archived},
		Pending:    {Confirmed, Cancelled},
		Confirmed:  {Disrupted, Cancelled, Archived},
		Disrupted:  {Confirmed, Cancelled},
		Cancelled:  {},
		Archived:   {},
	}
	for _, candidate := range allowed[from] {
		if candidate == to {
			return true
		}
	}
	return false
}
