package store

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"trotamundos/filius/internal/domain"
)

// User es la entidad de identidad del bounded context de Itinerary (ADR-106).
type User struct {
	ID           string         `json:"id"`
	Email        string         `json:"email"`
	PasswordHash string         `json:"-"`
	DisplayName  string         `json:"display_name"`
	Preferences  map[string]any `json:"preferences"`
}

func token(n int) string {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}

func hash(password string) string {
	sum := sha256.Sum256([]byte(password))
	return hex.EncodeToString(sum[:])
}

// SeedUser crea el usuario de demostración (dev@trotamundos.app / dev).
func (s *Memory) SeedUser(email, password, name string) User {
	s.mu.Lock()
	defer s.mu.Unlock()
	u := User{
		ID: token(8), Email: email, PasswordHash: hash(password), DisplayName: name,
		Preferences: map[string]any{"locale": "es", "currency": "USD", "gps_radius_km": 50, "preferred_agency_id": nil},
	}
	s.users[u.ID] = &u
	return u
}

func (s *Memory) VerifyCredentials(email, password string) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, u := range s.users {
		if u.Email == email && u.PasswordHash == hash(password) {
			return *u, true
		}
	}
	return User{}, false
}

func (s *Memory) Preferences(userID string) map[string]any {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if u, ok := s.users[userID]; ok {
		out := map[string]any{}
		for k, v := range u.Preferences {
			out[k] = v
		}
		return out
	}
	return map[string]any{}
}

func (s *Memory) UpdatePreferences(userID string, patch map[string]any) map[string]any {
	s.mu.Lock()
	defer s.mu.Unlock()
	if u, ok := s.users[userID]; ok {
		for k, v := range patch {
			u.Preferences[k] = v
		}
		out := map[string]any{}
		for k, v := range u.Preferences {
			out[k] = v
		}
		return out
	}
	return nil
}

// Share crea un token opaco de solo lectura con vencimiento (FR-8, QA-10).
func (s *Memory) Share(tripID string, ttl time.Duration) domain.Share {
	sh := domain.Share{
		Token: token(12), TripID: tripID, Permission: "view",
		ExpiresAt: time.Now().Add(ttl).UTC().Format(time.RFC3339),
	}
	s.mu.Lock()
	s.shares[sh.Token] = sh
	s.mu.Unlock()
	return sh
}

func (s *Memory) ResolveShare(tokenValue string) (domain.Trip, bool) {
	s.mu.RLock()
	sh, ok := s.shares[tokenValue]
	s.mu.RUnlock()
	if !ok || sh.Permission != "view" {
		return domain.Trip{}, false
	}
	if when, err := time.Parse(time.RFC3339, sh.ExpiresAt); err == nil && time.Now().After(when) {
		return domain.Trip{}, false
	}
	return s.Get(sh.TripID)
}

func (s *Memory) AddTraveler(tripID, name, document, email string) domain.Traveler {
	t := domain.NewTraveler(tripID, name, document, email)
	s.mu.Lock()
	s.travelers[tripID] = append(s.travelers[tripID], t)
	s.mu.Unlock()
	return t
}

func (s *Memory) Travelers(tripID string) []domain.Traveler {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]domain.Traveler{}, s.travelers[tripID]...)
}

// SetBookingCoords asocia lat/lon a una reserva (QA-13).
func (s *Memory) SetBookingCoords(providerRef string, lat, lon float64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.trips {
		for i := range t.Bookings {
			if t.Bookings[i].ProviderRef == providerRef {
				t.Bookings[i].Details["lat"] = lat
				t.Bookings[i].Details["lon"] = lon
				return true
			}
		}
	}
	return false
}

// UpdateBookingStatus aplica transiciones válidas de la state machine.
func (s *Memory) UpdateBookingStatus(providerRef, status string) (domain.Booking, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, t := range s.trips {
		for i := range t.Bookings {
			if t.Bookings[i].ProviderRef == providerRef && domain.CanTransition(t.Bookings[i].Status, status) {
				t.Bookings[i].Status = status
				return t.Bookings[i], true
			}
		}
	}
	return domain.Booking{}, false
}

// Nearby devuelve viajes con reservas dentro del radio (fórmula de Haversine, QA-13).
func (s *Memory) Nearby(lat, lon, radiusKm float64) []domain.Trip {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []domain.Trip{}
	for _, id := range s.order {
		t := s.trips[id]
		if t == nil {
			continue
		}
		match := false
		for _, b := range t.Bookings {
			bLat, okLat := b.Details["lat"].(float64)
			bLon, okLon := b.Details["lon"].(float64)
			if okLat && okLon && distance(lat, lon, bLat, bLon) <= radiusKm {
				match = true
				break
			}
		}
		if match {
			out = append(out, clone(*t))
		}
	}
	sort.Slice(out, func(i, j int) bool { return strings.Compare(out[i].ID, out[j].ID) < 0 })
	return out
}

func distance(lat1, lon1, lat2, lon2 float64) float64 {
	const r = 6371.0
	rad := math.Pi / 180
	dLat := (lat2 - lat1) * rad
	dLon := (lon2 - lon1) * rad
	a := math.Sin(dLat/2)*math.Sin(dLat/2) + math.Cos(lat1*rad)*math.Cos(lat2*rad)*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * r * math.Asin(math.Sqrt(a))
}
