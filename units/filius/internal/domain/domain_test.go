package domain

import (
	"testing"
	"time"
)

func TestStateMachineBooking(t *testing.T) {
	cases := []struct{ from, to string; want bool }{
		{Discovered, Pending, true},
		{Pending, Confirmed, true},
		{Confirmed, Disrupted, true},
		{Disrupted, Confirmed, true},
		{Cancelled, Confirmed, false},
		{Archived, Pending, false},
		{Pending, Archived, false},
	}
	for _, c := range cases {
		if got := CanTransition(c.from, c.to); got != c.want {
			t.Errorf("CanTransition(%s,%s) = %v, want %v", c.from, c.to, got, c.want)
		}
	}
}

func TestArchiveCompletedViajePasado(t *testing.T) {
	today := time.Date(2026, 11, 22, 0, 0, 0, 0, time.UTC)
	trips := []Trip{
		NewTrip("Finalizado", "2026-10-01", "2026-10-10"),
		NewTrip("Próximo", "2026-12-01", "2026-12-10"),
	}
	archived := ArchiveCompleted(trips, today)
	if len(archived) != 1 || archived[0].Name != "Finalizado" {
		t.Fatalf("esperaba archivar solo el viaje finalizado, got %+v", archived)
	}
	if archived[0].Status != Archived {
		t.Errorf("estado = %s, want %s", archived[0].Status, Archived)
	}
}
