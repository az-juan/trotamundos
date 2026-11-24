// Package store implementa persistencia en memoria para el prototipo ejecutable.
// Sustituible por un adaptador PostgreSQL sin tocar dominio ni casos de uso (ADR-002/104).
package store

import (
	"fmt"
	"sync"
	"time"

	"trotamundos/filius/internal/domain"
)

type Memory struct {
	mu        sync.RWMutex
	trips     map[string]*domain.Trip
	order     []string
	users     map[string]*User
	shares    map[string]domain.Share
	travelers map[string][]domain.Traveler
	cases     map[string]*AssistanceCase
	caseOrder []string
}

func New() *Memory {
	return &Memory{
		trips:     map[string]*domain.Trip{},
		order:     []string{},
		users:     map[string]*User{},
		shares:    map[string]domain.Share{},
		travelers: map[string][]domain.Traveler{},
		cases:     map[string]*AssistanceCase{},
		caseOrder: []string{},
	}
}

func clone(t domain.Trip) domain.Trip {
	copyTrips := make([]domain.Booking, len(t.Bookings))
	copy(copyTrips, t.Bookings)
	t.Bookings = copyTrips
	return t
}

func (s *Memory) List(status string) []domain.Trip {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]domain.Trip, 0, len(s.order))
	for _, id := range s.order {
		t, ok := s.trips[id]
		if !ok {
			continue
		}
		if status != "" && t.Status != status {
			continue
		}
		out = append(out, clone(*t))
	}
	return out
}

func (s *Memory) Get(id string) (domain.Trip, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	t, ok := s.trips[id]
	if !ok {
		return domain.Trip{}, false
	}
	return clone(*t), true
}

func (s *Memory) Create(name, start, end string) domain.Trip {
	s.mu.Lock()
	defer s.mu.Unlock()
	t := domain.NewTrip(name, start, end)
	s.trips[t.ID] = &t
	s.order = append(s.order, t.ID)
	return clone(t)
}

func (s *Memory) Archive(id string) (domain.Trip, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.trips[id]
	if !ok {
		return domain.Trip{}, false
	}
	t.Archive()
	return clone(*t), true
}

// ArchiveCompleted aplica el auto-archivado (FR-5) y devuelve lo archivado.
func (s *Memory) ArchiveCompleted(today time.Time) []domain.Trip {
	s.mu.Lock()
	defer s.mu.Unlock()
	trips := make([]domain.Trip, 0, len(s.order))
	for _, id := range s.order {
		t, ok := s.trips[id]
		if ok {
			trips = append(trips, clone(*t))
		}
	}
	archived := domain.ArchiveCompleted(trips, today)
	for _, t := range archived {
		if original, ok := s.trips[t.ID]; ok {
			original.Archive()
		}
	}
	return archived
}

func (s *Memory) AddBooking(tripID, kind, provider, ref string) (domain.Booking, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	t, ok := s.trips[tripID]
	if !ok {
		return domain.Booking{}, fmt.Errorf("viaje %s no encontrado", tripID)
	}
	if ref == "" {
		return domain.Booking{}, fmt.Errorf("provider_ref obligatorio")
	}
	if _, found := s.findByRefLocked(ref); found {
		return domain.Booking{}, fmt.Errorf("reserva %s ya existe (idempotencia)", ref)
	}
	b := domain.NewBooking(tripID, kind, provider, ref)
	b.Status = domain.Pending
	t.AddBooking(b)
	return b, nil
}

func (s *Memory) findByRefLocked(ref string) (domain.Booking, bool) {
	for _, t := range s.trips {
		for _, b := range t.Bookings {
			if b.ProviderRef == ref {
				return b, true
			}
		}
	}
	return domain.Booking{}, false
}

// IngestDiscovered crea o actualiza una reserva desde un correo (FR-2).
// Devuelve la reserva y el viaje al que pertenece.
func (s *Memory) IngestDiscovered(provider, ref, kind string) (domain.Booking, domain.Trip) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if existing, found := s.findByRefLocked(ref); found {
		return existing, cloneTripValue(s.trips[existing.TripID])
	}
	// Busca un viaje abierto para agrupar (FR-5); si no hay, crea uno.
	var target *domain.Trip
	for _, id := range s.order {
		t := s.trips[id]
		if t.Status == "upcoming" || t.Status == "active" {
			target = t
			break
		}
	}
	if target == nil {
		t := domain.NewTrip("Viaje descubierto · "+provider, "", "")
		s.trips[t.ID] = &t
		s.order = append(s.order, t.ID)
		target = &t
	}
	b := domain.NewBooking(target.ID, kind, provider, ref)
	b.Status = domain.Confirmed
	target.AddBooking(b)
	return b, clone(*target)
}

func cloneTripValue(t *domain.Trip) domain.Trip {
	copyBookings := make([]domain.Booking, len(t.Bookings))
	copy(copyBookings, t.Bookings)
	ct := *t
	ct.Bookings = copyBookings
	return ct
}

// Seed carga datos de demostración para el primer prototipo (CON-4).
func (s *Memory) Seed(today time.Time) {
	start := today.AddDate(0, 0, 12).Format("2006-01-02")
	end := today.AddDate(0, 0, 19).Format("2006-01-02")
	a := s.Create("Buenos Aires → Madrid", start, end)
	_, _ = s.AddBooking(a.ID, "transport", "Aerolíneas del Plata", "PNR-1930")
	_, _ = s.AddBooking(a.ID, "lodging", "Hotel Presidente", "HT-4471")
	_, _ = s.UpdateBookingStatus("PNR-1930", domain.Confirmed)
	_, _ = s.UpdateBookingStatus("HT-4471", domain.Confirmed)
	b := s.Create("Escapada a Montevideo", "", "")
	_, _ = s.AddBooking(b.ID, "transport", "Ferrocarril del Este", "TR-2210")
}
