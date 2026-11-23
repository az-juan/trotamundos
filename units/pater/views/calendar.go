package views

import (
	"fmt"
	"strings"
	"time"
)

// Calendario SSR: una cuadrícula mensual con los viajes en cada fecha.
// Sin JavaScript: los botones de mes usan HTMX sobre #main-content.

var mesesDelAnio = []string{
	"Enero", "Febrero", "Marzo", "Abril", "Mayo", "Junio",
	"Julio", "Agosto", "Septiembre", "Octubre", "Noviembre", "Diciembre",
}

type CalendarDay struct {
	Number  int
	InMonth bool
	IsToday bool
	Trips   []TripView
}

// CalendarWeek es una fila de la agenda: número de semana ISO y sus 7 días.
type CalendarWeek struct {
	Number int
	Days   []CalendarDay
}

type CalendarMonth struct {
	Key     string
	Label   string
	Prev    string
	Next    string
	Weeks   []CalendarWeek
	Undated []TripView
}

// ParseMonth acepta YYYY-MM y usa el mes actual si no es válido.
func ParseMonth(value string) time.Time {
	if t, err := time.Parse("2006-01", value); err == nil {
		return time.Date(t.Year(), t.Month(), 1, 0, 0, 0, 0, time.UTC)
	}
	now := time.Now().UTC()
	return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
}

// BuildCalendar arma semanas completas (lunes a domingo) y ubica cada viaje
// en todas las fechas de su rango, inclusive.
func BuildCalendar(trips []TripView, month time.Time) CalendarMonth {
	first := time.Date(month.Year(), month.Month(), 1, 0, 0, 0, 0, time.UTC)
	offset := (int(first.Weekday()) + 6) % 7 // lunes = 0
	days := first.AddDate(0, 1, -1).Day()
	cells := ((offset + days + 6) / 7) * 7
	start := first.AddDate(0, 0, -offset)
	today := time.Now().UTC().Format("2006-01-02")

	cal := CalendarMonth{
		Key:   first.Format("2006-01"),
		Label: fmt.Sprintf("%s %d", mesesDelAnio[int(first.Month())-1], first.Year()),
		Prev:  first.AddDate(0, -1, 0).Format("2006-01"),
		Next:  first.AddDate(0, 1, 0).Format("2006-01"),
	}
	for _, trip := range trips {
		if trip.StartDate == "" {
			cal.Undated = append(cal.Undated, trip)
		}
	}

	week := []CalendarDay{}
	for i := 0; i < cells; i++ {
		date := start.AddDate(0, 0, i)
		key := date.Format("2006-01-02")
		day := CalendarDay{
			Number:  date.Day(),
			InMonth: date.Month() == first.Month(),
			IsToday: key == today,
		}
		for _, trip := range trips {
			if trip.StartDate == "" {
				continue
			}
			end := trip.EndDate
			if end == "" {
				end = trip.StartDate
			}
			if key >= trip.StartDate && key <= end {
				day.Trips = append(day.Trips, trip)
			}
		}
		week = append(week, day)
		if len(week) == 7 {
			_, weekNumber := date.ISOWeek()
			cal.Weeks = append(cal.Weeks, CalendarWeek{Number: weekNumber, Days: week})
			week = []CalendarDay{}
		}
	}
	return cal
}

// StatusClass mapea el estado del viaje a la clase CSS del calendario y leyenda.
func StatusClass(status string) string {
	switch status {
	case "active":
		return "trip-active"
	case "archived":
		return "trip-archived"
	default:
		return "trip-upcoming"
	}
}

// CalendarDayClass distingue días de otros meses y el día de hoy.
func CalendarDayClass(day CalendarDay) string {
	classes := []string{"calendar-day"}
	if !day.InMonth {
		classes = append(classes, "outside")
	}
	if day.IsToday {
		classes = append(classes, "is-today")
	}
	return strings.Join(classes, " ")
}
