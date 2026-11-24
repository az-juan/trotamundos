package web

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"trotamundos/pater/views"
)

type fakeStore struct {
	trips    []views.TripView
	created  []string
	added    []string
	prefs    views.PreferencesData
	senders  []views.SenderView
	ingested int
	cases    []views.AssistanceCaseView
	tier1    int
}

func (f *fakeStore) Trips(context.Context, string) ([]views.TripView, error) { return f.trips, nil }
func (f *fakeStore) CreateTrip(_ context.Context, name, _, _ string) error {
	f.created = append(f.created, name)
	f.trips = append(f.trips, views.TripView{ID: "new", Name: name, Status: "upcoming"})
	return nil
}
func (f *fakeStore) AddBooking(_ context.Context, _, _, provider, ref string) error {
	f.added = append(f.added, provider+"/"+ref)
	return nil
}
func (f *fakeStore) Trip(_ context.Context, id string) (views.TripView, error) {
	for _, t := range f.trips {
		if t.ID == id {
			return t, nil
		}
	}
	return views.TripView{}, errors.New("no encontrado")
}
func (f *fakeStore) Travelers(context.Context, string) ([]views.TravelerView, error) {
	return []views.TravelerView{{ID: "v1", Name: "Ana Ruiz"}}, nil
}
func (f *fakeStore) ShareTrip(context.Context, string) (views.ShareResult, error) {
	return views.ShareResult{Token: "tok", URL: "/shared/tok", ExpiresAt: "2026-12-01T00:00:00Z"}, nil
}
func (f *fakeStore) ArchiveTrip(context.Context, string) error { return nil }
func (f *fakeStore) Verify(_ context.Context, email, password string) (views.UserIdentity, error) {
	if email != "dev@trotamundos.app" || password != "dev" {
		return views.UserIdentity{}, errors.New("credenciales incorrectas")
	}
	return views.UserIdentity{ID: "user-1", Email: email}, nil
}
func (f *fakeStore) Preferences(context.Context, string) (views.PreferencesData, error) {
	return f.prefs, nil
}
func (f *fakeStore) UpdatePreferences(_ context.Context, _ string, p views.PreferencesData) error {
	f.prefs = p
	return nil
}
func (f *fakeStore) Nearby(context.Context, string, string, string) ([]views.TripView, error) {
	return f.trips, nil
}
func (f *fakeStore) SharedTrip(context.Context, string) (views.SharedData, error) {
	return views.SharedData{Trip: f.trips[0]}, nil
}
func (f *fakeStore) Insights(context.Context) (views.InsightView, error) {
	return views.InsightView{Total: 3, Confirmations: 2}, nil
}
func (f *fakeStore) Activity(context.Context) ([]views.ActivityView, error) {
	return []views.ActivityView{{Type: "booking.discovered", Summary: "Aerolíneas del Plata"}}, nil
}
func (f *fakeStore) IngestDemo(context.Context) error { f.ingested++; return nil }
func (f *fakeStore) GeoPoints(context.Context) ([]views.GeoPoint, error) {
	return []views.GeoPoint{
		{TripName: "Buenos Aires → Madrid", Status: "upcoming", Provider: "Aerolíneas del Plata", ProviderRef: "PNR-1930", Lat: -34.6037, Lon: -58.3816},
		{TripName: "Buenos Aires → Madrid", Status: "archived", Provider: "Hotel Presidente", ProviderRef: "HT-4471", Lat: -34.6200, Lon: -58.4200},
	}, nil
}
func (f *fakeStore) Mailboxes(context.Context) ([]views.MailboxView, error) {
	return []views.MailboxView{{ID: "m1", Provider: "gmail", Email: "dev@trotamundos.app"}}, nil
}
func (f *fakeStore) AddMailbox(context.Context, string, string) error    { return nil }
func (f *fakeStore) Senders(context.Context) ([]views.SenderView, error) { return f.senders, nil }
func (f *fakeStore) AddSender(_ context.Context, sender string) error {
	f.senders = append(f.senders, views.SenderView{ID: "s2", Sender: sender})
	return nil
}
func (f *fakeStore) RemoveSender(context.Context, string) error { return nil }
func (f *fakeStore) AssistanceCases(context.Context) ([]views.AssistanceCaseView, error) {
	return f.cases, nil
}
func (f *fakeStore) GetAssistanceCase(_ context.Context, id string) (views.AssistanceCaseView, error) {
	for _, c := range f.cases {
		if c.ID == id {
			return c, nil
		}
	}
	return views.AssistanceCaseView{}, errors.New("no encontrado")
}
func (f *fakeStore) CreateAssistanceCase(_ context.Context, tripID, bookingID, reason, severity, agencyID string) (views.AssistanceCaseView, error) {
	if agencyID == "" {
		agencyID = "agencia-austral"
	}
	c := views.AssistanceCaseView{
		ID: "case-1930", UserID: "user-1", TripID: tripID, BookingID: bookingID,
		Status: "open", Reason: reason, Severity: severity, AgencyID: agencyID,
	}
	f.cases = append(f.cases, c)
	return c, nil
}
func (f *fakeStore) UpdateAssistanceCaseStatus(_ context.Context, id, status string) (views.AssistanceCaseView, error) {
	for i := range f.cases {
		if f.cases[i].ID == id {
			f.cases[i].Status = status
			return f.cases[i], nil
		}
	}
	return views.AssistanceCaseView{}, errors.New("no encontrado")
}
func (f *fakeStore) RecordTier1Resolution(context.Context, string) error {
	f.tier1++
	return nil
}

func newHandlers() (*Handlers, *fakeStore) {
	f := &fakeStore{
		trips: []views.TripView{{ID: "abc123", Name: "Buenos Aires → Madrid", Status: "upcoming", Bookings: []views.BookingView{
			{Type: "transport", Provider: "Aerolíneas del Plata", ProviderRef: "PNR-1930", Status: "confirmed"},
		}}},
		senders: []views.SenderView{{ID: "s1", Sender: "@aerolinea.com"}},
	}
	return &Handlers{Store: f}, f
}

func serve(h *Handlers, method, path, body string, cookie *http.Cookie) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if cookie != nil {
		req.AddCookie(cookie)
	}
	w := httptest.NewRecorder()
	h.Handler().ServeHTTP(w, req)
	return w
}

func TestPortadaSinJSON(t *testing.T) {
	h, _ := newHandlers()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("Accept", "application/json")
	w := httptest.NewRecorder()
	h.Handler().ServeHTTP(w, req)
	body := w.Body.String()
	for _, needle := range []string{"TROTAMUNDOS", "MCMXXX", "Buenos Aires → Madrid", `id="main-content"`, `/static/assets/htmx.min.js`} {
		if !strings.Contains(body, needle) {
			t.Errorf("falta %q", needle)
		}
	}
	if strings.Count(body, `<script`) != 1 {
		t.Error("solo debe cargarse la librería HTMX")
	}
	if w := serve(h, http.MethodGet, "/users/me/preferences", "", nil); w.Code != http.StatusNotFound {
		t.Errorf("la ruta JSON de preferencias no debe existir: %d", w.Code)
	}
}

func TestFragmentosYFormularios(t *testing.T) {
	h, f := newHandlers()
	w := serve(h, http.MethodGet, "/trips?status=upcoming", "", nil)
	if strings.Contains(w.Body.String(), "<html") || !strings.Contains(w.Body.String(), `id="trip-list"`) {
		t.Error("el fragmento debe traer solo #trip-list")
	}
	w = serve(h, http.MethodPost, "/trips", url.Values{"name": {"Nuevo viaje"}}.Encode(), nil)
	if len(f.created) != 1 || !strings.Contains(w.Body.String(), "Nuevo viaje") {
		t.Error("alta de viaje HTMX falló")
	}
	serve(h, http.MethodPost, "/bookings", url.Values{
		"trip_id": {"abc123"}, "type": {"lodging"}, "provider": {"Hotel"}, "provider_ref": {"H-1"},
	}.Encode(), nil)
	if len(f.added) != 1 {
		t.Error("alta de reserva HTMX falló")
	}
}

func TestPaginasComponenHeaderFooter(t *testing.T) {
	h, _ := newHandlers()
	for _, path := range []string{"/trips/view", "/insights/view", "/map", "/calendar", "/mailboxes", "/whitelists", "/trips/abc123", "/nearby", "/shared/tok", "/login"} {
		w := serve(h, http.MethodGet, path, "", nil)
		if w.Code != http.StatusOK {
			t.Fatalf("%s: %d", path, w.Code)
		}
		if !strings.Contains(w.Body.String(), `<header class="site-header"`) || !strings.Contains(w.Body.String(), `class="site-footer"`) {
			t.Errorf("%s no compone Header/Footer", path)
		}
	}
}

func TestSesionPorCookie(t *testing.T) {
	h, _ := newHandlers()
	w := serve(h, http.MethodPost, "/auth/login", url.Values{
		"email": {"dev@trotamundos.app"}, "password": {"dev"},
	}.Encode(), nil)
	var cookie *http.Cookie
	for _, c := range w.Result().Cookies() {
		if c.Name == sessionCookie {
			cookie = c
		}
	}
	if cookie == nil || !cookie.HttpOnly || strings.Count(cookie.Value, ".") != 2 {
		t.Fatalf("cookie de sesión incorrecta: %+v", cookie)
	}
	if !strings.Contains(serve(h, http.MethodGet, "/preferences", "", cookie).Body.String(), "Preferencias personales") {
		t.Error("la cookie no autoriza preferencias")
	}
	if !strings.Contains(serve(h, http.MethodGet, "/preferences", "", nil).Body.String(), "Identifíquese") {
		t.Error("sin cookie debe pedir login")
	}
	if strings.Count(cookie.Value, ".") != 2 {
		t.Error("formato de cookie inválido")
	}
}

func TestOperacionesHTMX(t *testing.T) {
	h, f := newHandlers()
	w := serve(h, http.MethodPost, "/whitelists", "sender=@nuevo.com", nil)
	if len(f.senders) != 2 || !strings.Contains(w.Body.String(), "@nuevo.com") {
		t.Error("whitelist HTMX falló")
	}
	if serve(h, http.MethodDelete, "/whitelists/s1", "", nil).Code != http.StatusOK {
		t.Error("DELETE whitelist falló")
	}
	if !strings.Contains(serve(h, http.MethodPost, "/trips/abc123/share", "", nil).Body.String(), "/shared/tok") {
		t.Error("compartir falló")
	}
	w = serve(h, http.MethodPost, "/demo/ingest", "", nil)
	if f.ingested != 1 || !strings.Contains(w.Body.String(), `id="trip-list"`) {
		t.Error("ingesta HTMX falló")
	}
}

type fakeRenderer struct {
	calls      int
	lastStatus string
}

func (f *fakeRenderer) Render(_ context.Context, _ []views.GeoPoint, status string) ([]byte, error) {
	f.calls++
	f.lastStatus = status
	return []byte("PNG-FAKE"), nil
}

func TestMapaRealSinJS(t *testing.T) {
	h, _ := newHandlers()
	r := &fakeRenderer{}
	h.MapRenderer = r
	w := serve(h, http.MethodGet, "/map", "", nil)
	body := w.Body.String()
	for _, needle := range []string{`<img class="map-real"`, "/map/image", "OpenStreetMap"} {
		if !strings.Contains(body, needle) {
			t.Errorf("falta %q en la página del mapa", needle)
		}
	}
	if strings.Count(body, `<script`) != 1 {
		t.Error("el mapa no debe añadir JavaScript")
	}
	w = serve(h, http.MethodGet, "/map/image", "", nil)
	if w.Code != http.StatusOK || w.Body.String() != "PNG-FAKE" || w.Header().Get("Content-Type") != "image/png" {
		t.Errorf("PNG del mapa incorrecto: %d %q", w.Code, w.Body.String())
	}
	if r.calls != 1 {
		t.Error("el renderer no se invocó una vez")
	}
}

func serveHeaders(h *Handlers, method, path string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, nil)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.Handler().ServeHTTP(w, req)
	return w
}

// Una respuesta HTMX de página solo puede traer dos swaps OOB: título y #site-nav.
// Si un bloque interno llevara hx-swap-oob, HTMX lo extraería antes de
// insertarlo en #main-content y la página quedaría incompleta al navegar.
func TestNavegacionHTMXNoPierdeBloques(t *testing.T) {
	h, _ := newHandlers()
	w := serveHeaders(h, http.MethodGet, "/", map[string]string{"HX-Request": "true"})
	body := w.Body.String()
	for _, id := range []string{"trip-list", "filters", "insights", "activity"} {
		if !strings.Contains(body, `id="`+id+`"`) {
			t.Errorf("falta el bloque #%s al navegar al panel", id)
		}
	}
	if n := strings.Count(body, `hx-swap-oob="true"`); n != 2 {
		t.Errorf("esperaba 2 swaps OOB (título y navegación), hay %d", n)
	}
}

func TestRefreshDeInsightsConservaEncabezadoYNota(t *testing.T) {
	h, _ := newHandlers()
	body := serveHeaders(h, http.MethodGet, "/insights", map[string]string{"HX-Request": "true"}).Body.String()
	for _, needle := range []string{"stat-grid", "panel-heading-row", "rail-footnote"} {
		if !strings.Contains(body, needle) {
			t.Errorf("el refresh de insights perdió %q", needle)
		}
	}
	if strings.Contains(body, "hx-swap-oob") {
		t.Error("el refresh de insights no debe usar swaps OOB")
	}
}

func TestNearbyFragmentoYUbicacion(t *testing.T) {
	h, _ := newHandlers()
	fragmento := serveHeaders(h, http.MethodGet, "/nearby", map[string]string{
		"HX-Request": "true", "HX-Target": "trip-list",
	})
	if strings.Contains(fragmento.Body.String(), "<html") || strings.Contains(fragmento.Body.String(), "hx-swap-oob") {
		t.Error("el formulario de cercanías debe recibir solo #trip-list, sin OOB")
	}
	if !strings.Contains(fragmento.Body.String(), `id="trip-list"`) {
		t.Error("al formulario de cercanías le falta #trip-list")
	}
	pagina := serveHeaders(h, http.MethodGet, "/nearby", map[string]string{"HX-Request": "true"}).Body.String()
	if strings.Contains(pagina, `id="trip-list" class="trip-list" hx-swap-oob`) {
		t.Error("#trip-list no debe marcarse OOB dentro de la página de cercanías")
	}
	if !strings.Contains(pagina, `id="trip-list"`) {
		t.Error("la página de cercanías debe incluir sus resultados")
	}
}

func TestSimularCorreoDesdeNavRedibujaPanel(t *testing.T) {
	h, _ := newHandlers()
	desdeNav := serveHeaders(h, http.MethodPost, "/demo/ingest", map[string]string{
		"HX-Request": "true", "HX-Target": "main-content",
	}).Body.String()
	if !strings.Contains(desdeNav, `id="trip-list"`) || !strings.Contains(desdeNav, `id="activity"`) {
		t.Error("la navegación global debe redibujar el panel completo")
	}
	if strings.Contains(desdeNav, `id="trip-list" class="trip-list" hx-swap-oob`) {
		t.Error("el panel redibujado no debe marcar #trip-list como OOB")
	}
	desdePanel := serveHeaders(h, http.MethodPost, "/demo/ingest", map[string]string{
		"HX-Request": "true", "HX-Target": "activity",
	}).Body.String()
	if !strings.Contains(desdePanel, `id="activity"`) || !strings.Contains(desdePanel, `hx-swap-oob="true"`) {
		t.Error("el botón del panel debe actualizar #activity y la lista vía OOB")
	}
}

func TestHTMXMantieneTituloYNavegacion(t *testing.T) {
	h, _ := newHandlers()
	req := httptest.NewRequest(http.MethodGet, "/mailboxes", nil)
	req.Header.Set("HX-Request", "true")
	w := httptest.NewRecorder()
	h.Handler().ServeHTTP(w, req)
	body := w.Body.String()
	for _, needle := range []string{`hx-swap-oob="true"`, `id="site-nav"`, "Trotamundos · Buzones", "is-current"} {
		if !strings.Contains(body, needle) {
			t.Errorf("falta %q en la respuesta HTMX", needle)
		}
	}
	if strings.Contains(body, "<html") {
		t.Error("la respuesta HTMX no debe redibujar el documento completo")
	}
}

func TestMapaPorEstado(t *testing.T) {
	h, _ := newHandlers()
	page := serve(h, http.MethodGet, "/map?status=active", "", nil).Body.String()
	for _, needle := range []string{`/map/image?status=active`, "Próximos (1)", "En curso (0)", "Archivados (1)"} {
		if !strings.Contains(page, needle) {
			t.Errorf("falta %q en el mapa filtrado", needle)
		}
	}
	if !strings.Contains(page, `href="/calendar"`) {
		t.Error("el mapa debe enlazar al calendario")
	}

	r := &fakeRenderer{}
	h.MapRenderer = r
	serve(h, http.MethodGet, "/map/image?status=active", "", nil)
	if r.lastStatus != "active" {
		t.Errorf("el renderer recibió el filtro %q, esperaba active", r.lastStatus)
	}
}

func TestCalendarioMensual(t *testing.T) {
	h, f := newHandlers()
	f.trips = []views.TripView{
		{ID: "t1", Name: "Viaje con fechas", Status: "active", StartDate: "2026-12-03", EndDate: "2026-12-06"},
		{ID: "t2", Name: "Viaje sin fechas", Status: "upcoming"},
	}
	body := serve(h, http.MethodGet, "/calendar?month=2026-12", "", nil).Body.String()
	for _, needle := range []string{"Diciembre 2026", "calendar-grid", `<table class="agenda-table"`, "agenda-masthead", "agenda-weeknum", "Viaje con fechas", "Viaje sin fechas", "trip-active", `month=2026-11`, `month=2027-01`} {
		if !strings.Contains(body, needle) {
			t.Errorf("falta %q en el calendario", needle)
		}
	}
	if n := strings.Count(body, `class="agenda-weekday"`); n != 7 {
		t.Errorf("esperaba 7 días de la semana, hay %d", n)
	}
}

func TestAsistenteVirtualDeterministaYDerivacion(t *testing.T) {
	h, f := newHandlers()
	w := serve(h, http.MethodGet, "/assistant", "", nil)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "Mostrador de Asistencia") {
		t.Fatalf("GET /assistant falló: %d", w.Code)
	}

	// 1. Consulta de estado (resuelta en Nivel 1)
	w = serve(h, http.MethodPost, "/assistant/messages", url.Values{
		"message": {"¿En qué estado está mi reserva PNR-1930?"},
	}.Encode(), nil)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), "PNR-1930") || f.tier1 == 0 {
		t.Errorf("consulta de estado Nivel 1 falló: %d", w.Code)
	}

	// 2. Caso complejo derivado a agencia preferida (FR-11, ADR-110/113)
	w = serve(h, http.MethodPost, "/assistant/messages", url.Values{
		"message": {"Solicito reembolso complejo con mi agencia preferida"},
	}.Encode(), nil)
	if w.Code != http.StatusOK || len(f.cases) != 1 || !strings.Contains(w.Body.String(), "case-1930") {
		t.Errorf("derivación a agencia falló: cases=%d", len(f.cases))
	}

	// 3. Confirmar handoff y resolver caso
	w = serve(h, http.MethodPost, "/assistant/cases/case-1930/handoff", "", nil)
	if w.Code != http.StatusOK || f.cases[0].Status != "handed_off" {
		t.Errorf("handoff de caso falló: %+v", f.cases[0])
	}
	w = serve(h, http.MethodPost, "/assistant/cases/case-1930/resolve", "", nil)
	if w.Code != http.StatusOK || f.cases[0].Status != "resolved" {
		t.Errorf("resolución de caso falló: %+v", f.cases[0])
	}
}
