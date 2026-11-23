package domain

// Traveler es un viajero asociado a un viaje (QA-10, modelo de datos §3).
type Traveler struct {
	ID       string `json:"id"`
	TripID   string `json:"trip_id"`
	Name     string `json:"name"`
	Document string `json:"document,omitempty"`
	Email    string `json:"email,omitempty"`
}

func NewTraveler(tripID, name, document, email string) Traveler {
	return Traveler{ID: newID(), TripID: tripID, Name: name, Document: document, Email: email}
}

// Share Token de compartición con permiso de solo lectura (FR-8, QA-10).
type Share struct {
	Token      string `json:"token"`
	TripID     string `json:"trip_id"`
	Permission string `json:"permission"`
	ExpiresAt  string `json:"expires_at"`
}
