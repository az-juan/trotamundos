// Package logic contiene el acceso directo a PostgreSQL mediante sqlc (ADR-111).
// Los handlers de Pater llaman a este paquete; no hay peticiones JSON entre unidades.
package logic

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"trotamundos/pater/views"
	"trotamundos/shared/db"
)

const demoUser = "00000000-0000-0000-0000-000000000001"

type SQLStore struct {
	q      *db.Queries
	userID string
}

func New(pool *pgxpool.Pool) *SQLStore {
	return &SQLStore{q: db.New(pool), userID: demoUser}
}

func uuid(value string) pgtype.UUID {
	var u pgtype.UUID
	if value == "" {
		return u
	}
	_ = u.Scan(value)
	return u
}

func str(value any) string {
	if value == nil {
		return ""
	}
	if s, ok := value.(string); ok {
		return s
	}
	return fmt.Sprint(value)
}

func (s *SQLStore) withBookings(ctx context.Context, t views.TripView) (views.TripView, error) {
	rows, err := s.q.ListBookingsForTrip(ctx, uuid(t.ID))
	if err != nil {
		return t, err
	}
	t.Bookings = []views.BookingView{}
	for _, b := range rows {
		t.Bookings = append(t.Bookings, views.BookingView{
			ID: b.ID, Type: b.Type, Status: b.Status, Provider: b.Provider, ProviderRef: b.ProviderRef,
		})
	}
	return t, nil
}

func (s *SQLStore) Trips(ctx context.Context, status string) ([]views.TripView, error) {
	rows, err := s.q.ListTrips(ctx, status)
	if err != nil {
		return nil, err
	}
	out := []views.TripView{}
	for _, r := range rows {
		t := views.TripView{ID: r.ID, Name: r.Name, Status: r.Status, StartDate: r.StartDate, EndDate: r.EndDate}
		t, err = s.withBookings(ctx, t)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

func (s *SQLStore) Trip(ctx context.Context, id string) (views.TripView, error) {
	r, err := s.q.GetTrip(ctx, uuid(id))
	if err != nil {
		return views.TripView{}, err
	}
	return s.withBookings(ctx, views.TripView{
		ID: r.ID, Name: r.Name, Status: r.Status, StartDate: r.StartDate, EndDate: r.EndDate,
	})
}

func (s *SQLStore) CreateTrip(ctx context.Context, name, start, end string) error {
	_, err := s.q.CreateTrip(ctx, db.CreateTripParams{
		UserID: uuid(s.userID), Name: name, StartDate: start, EndDate: end,
	})
	return err
}

func (s *SQLStore) AddBooking(ctx context.Context, tripID, kind, provider, ref string) error {
	_, err := s.q.AddBooking(ctx, db.AddBookingParams{
		TripID: uuid(tripID), Kind: kind, Provider: provider, ProviderRef: ref,
	})
	return err
}

func (s *SQLStore) ArchiveTrip(ctx context.Context, id string) error {
	return s.q.ArchiveTrip(ctx, uuid(id))
}

func (s *SQLStore) Travelers(ctx context.Context, tripID string) ([]views.TravelerView, error) {
	rows, err := s.q.ListTravelers(ctx, uuid(tripID))
	if err != nil {
		return nil, err
	}
	out := []views.TravelerView{}
	for _, r := range rows {
		out = append(out, views.TravelerView{ID: r.ID, Name: r.Name, Document: r.Document, Email: r.Email})
	}
	return out, nil
}

func (s *SQLStore) ShareTrip(ctx context.Context, tripID string) (views.ShareResult, error) {
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	token := hex.EncodeToString(b)
	r, err := s.q.CreateShare(ctx, db.CreateShareParams{TripID: uuid(tripID), Token: token})
	if err != nil {
		return views.ShareResult{}, err
	}
	return views.ShareResult{Token: r.Token, URL: "/shared/" + r.Token, ExpiresAt: r.ExpiresAt}, nil
}

func (s *SQLStore) SharedTrip(ctx context.Context, token string) (views.SharedData, error) {
	share, err := s.q.GetShare(ctx, token)
	if err != nil {
		return views.SharedData{}, err
	}
	t, err := s.Trip(ctx, share.TripID)
	if err != nil {
		return views.SharedData{}, err
	}
	travelers, _ := s.Travelers(ctx, share.TripID)
	return views.SharedData{Trip: t, Travelers: travelers}, nil
}

func (s *SQLStore) Verify(ctx context.Context, email, password string) (views.UserIdentity, error) {
	u, err := s.q.VerifyUser(ctx, db.VerifyUserParams{Email: email, Password: password})
	if err != nil {
		return views.UserIdentity{}, err
	}
	return views.UserIdentity{ID: u.ID, Email: u.Email, DisplayName: u.DisplayName}, nil
}

func (s *SQLStore) Preferences(ctx context.Context, userID string) (views.PreferencesData, error) {
	raw, err := s.q.GetPreferences(ctx, uuid(userID))
	if err != nil {
		return views.PreferencesData{}, err
	}
	var prefs map[string]any
	if err := json.Unmarshal(raw, &prefs); err != nil {
		return views.PreferencesData{}, err
	}
	radius := ""
	if v, ok := prefs["gps_radius_km"]; ok {
		radius = fmt.Sprintf("%v", v)
	}
	return views.PreferencesData{
		Locale: str(prefs["locale"]), Currency: str(prefs["currency"]),
		GpsRadius:       strings.TrimSuffix(strings.TrimSuffix(radius, ".0"), ".0"),
		PreferredAgency: str(prefs["preferred_agency_id"]),
	}, nil
}

func (s *SQLStore) UpdatePreferences(ctx context.Context, userID string, p views.PreferencesData) error {
	raw, err := s.q.GetPreferences(ctx, uuid(userID))
	if err != nil {
		return err
	}
	prefs := map[string]any{}
	_ = json.Unmarshal(raw, &prefs)
	prefs["locale"] = p.Locale
	prefs["currency"] = p.Currency
	prefs["preferred_agency_id"] = p.PreferredAgency
	if p.GpsRadius != "" {
		var radius float64
		if _, err := fmt.Sscanf(p.GpsRadius, "%f", &radius); err == nil {
			prefs["gps_radius_km"] = radius
		}
	}
	encoded, err := json.Marshal(prefs)
	if err != nil {
		return err
	}
	return s.q.UpdatePreferences(ctx, db.UpdatePreferencesParams{Preferences: encoded, ID: uuid(userID)})
}

func (s *SQLStore) Nearby(ctx context.Context, lat, lon, radius string) ([]views.TripView, error) {
	var la, lo, rad float64
	if _, err := fmt.Sscanf(lat, "%f", &la); err != nil {
		return nil, err
	}
	if _, err := fmt.Sscanf(lon, "%f", &lo); err != nil {
		return nil, err
	}
	if _, err := fmt.Sscanf(radius, "%f", &rad); err != nil || rad <= 0 {
		rad = 50
	}
	rows, err := s.q.NearbyTrips(ctx, db.NearbyTripsParams{Lat: la, Lon: lo, RadiusKm: rad})
	if err != nil {
		return nil, err
	}
	out := []views.TripView{}
	for _, r := range rows {
		t := views.TripView{ID: r.ID, Name: r.Name, Status: r.Status, StartDate: r.StartDate, EndDate: r.EndDate}
		t, err = s.withBookings(ctx, t)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, nil
}

func (s *SQLStore) Insights(ctx context.Context) (views.InsightView, error) {
	row, err := s.q.AnalyticsSummary(ctx)
	if err != nil {
		return views.InsightView{}, err
	}
	return views.InsightView{
		Total: int(row.Total), Confirmations: int(row.Confirmations),
		Disruptions: int(row.Disruptions), ByType: map[string]int{}, LastEventAt: row.LastEventAt,
	}, nil
}

func (s *SQLStore) Activity(ctx context.Context) ([]views.ActivityView, error) {
	rows, err := s.q.ListAnalyticsEvents(ctx, 6)
	if err != nil {
		return nil, err
	}
	out := []views.ActivityView{}
	for _, r := range rows {
		var payload map[string]any
		_ = json.Unmarshal(r.Payload, &payload)
		summary := r.EventType
		if provider := str(payload["provider"]); provider != "" {
			summary = provider
			if ref := str(payload["provider_ref"]); ref != "" {
				summary += " · " + ref
			}
		}
		out = append(out, views.ActivityView{Type: r.EventType, Source: r.Source, OccurredAt: r.OccurredAt, Summary: summary})
	}
	return out, nil
}

var refPattern = regexp.MustCompile(`(?i:pnr|reserva|confirmaci[oó]n)[\s:·-]*([A-Z]{2,8}-[A-Z0-9]{2,8}|[A-Z0-9]{5,12})`)

// proveedoresDemo alimenta "Simular correo": cada captura elige una entidad
// ficticia distinta para que el panel no repita siempre la misma reserva.
var proveedoresDemo = []struct{ name, prefix, kind string }{
	{"Aerolíneas del Plata", "AR", "transport"},
	{"Líneas Aéreas Cóndor", "CO", "transport"},
	{"Vapores del Sur", "VS", "transport"},
	{"Expresos Andinos", "EX", "transport"},
	{"Hotel Presidente", "HT", "lodging"},
	{"Posada del Ceibo", "PC", "lodging"},
}

// IngestDemo simula la captura de un correo y escribe directamente con sqlc (FR-2).
func (s *SQLStore) IngestDemo(ctx context.Context) error {
	demo := proveedoresDemo[time.Now().UnixNano()%int64(len(proveedoresDemo))]
	ref := fmt.Sprintf("%s-%04X", demo.prefix, time.Now().UnixNano()%0x10000)
	body := fmt.Sprintf("Estimado viajero: su reserva %s con %s ha sido confirmada.", ref, demo.name)
	match := refPattern.FindStringSubmatch(body)
	if match == nil {
		return fmt.Errorf("no se pudo extraer la reserva")
	}
	ref = strings.ToUpper(match[1])
	if _, err := s.q.FindBookingByRef(ctx, ref); err == nil {
		return nil // dedup por referencia
	}
	trips, err := s.Trips(ctx, "")
	if err != nil || len(trips) == 0 {
		return fmt.Errorf("no hay viaje destino: %v", err)
	}
	if _, err := s.q.AddBooking(ctx, db.AddBookingParams{
		TripID: uuid(trips[0].ID), Kind: demo.kind, Provider: demo.name, ProviderRef: ref,
	}); err != nil {
		return err
	}
	payload, _ := json.Marshal(map[string]any{"provider": demo.name, "provider_ref": ref, "type": demo.kind})
	return s.q.InsertAnalyticsEvent(ctx, db.InsertAnalyticsEventParams{
		EventID: ref, EventType: "booking.discovered", Payload: payload,
	})
}

func (s *SQLStore) Mailboxes(ctx context.Context) ([]views.MailboxView, error) {
	rows, err := s.q.ListMailboxes(ctx)
	if err != nil {
		return nil, err
	}
	out := []views.MailboxView{}
	for _, r := range rows {
		out = append(out, views.MailboxView{ID: r.ID, Provider: r.Provider, Email: r.Email, LastScanAt: str(r.LastScanAt)})
	}
	return out, nil
}

func (s *SQLStore) AddMailbox(ctx context.Context, provider, email string) error {
	_, err := s.q.AddMailbox(ctx, db.AddMailboxParams{UserID: uuid(s.userID), Provider: provider, Email: email})
	return err
}

func (s *SQLStore) Senders(ctx context.Context) ([]views.SenderView, error) {
	rows, err := s.q.ListSenders(ctx)
	if err != nil {
		return nil, err
	}
	out := []views.SenderView{}
	for _, r := range rows {
		out = append(out, views.SenderView{ID: r.ID, Sender: r.Sender})
	}
	return out, nil
}

func (s *SQLStore) AddSender(ctx context.Context, sender string) error {
	_, err := s.q.AddSender(ctx, db.AddSenderParams{UserID: uuid(s.userID), Sender: sender})
	return err
}

func (s *SQLStore) RemoveSender(ctx context.Context, id string) error {
	return s.q.RemoveSender(ctx, uuid(id))
}

func (s *SQLStore) GeoPoints(ctx context.Context) ([]views.GeoPoint, error) {
	rows, err := s.q.ListGeoPoints(ctx)
	if err != nil {
		return nil, err
	}
	out := []views.GeoPoint{}
	for _, r := range rows {
		out = append(out, views.GeoPoint{
			TripID: r.TripID, TripName: r.TripName, Provider: r.Provider,
			ProviderRef: r.ProviderRef, Type: r.Type, Lat: r.Lat, Lon: r.Lon,
		})
	}
	return out, nil
}
