package web

// Handlers: vistas templ/HTMX alimentadas por sqlc, sin API JSON ni clientes HTTP.

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"io"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"

	"trotamundos/pater/views"
)

// Store es el puerto de persistencia; en producción lo implementa logic.SQLStore (sqlc).
type Store interface {
	Trips(ctx context.Context, status string) ([]views.TripView, error)
	CreateTrip(ctx context.Context, name, start, end string) error
	AddBooking(ctx context.Context, tripID, kind, provider, ref string) error
	Trip(ctx context.Context, id string) (views.TripView, error)
	Travelers(ctx context.Context, tripID string) ([]views.TravelerView, error)
	ShareTrip(ctx context.Context, id string) (views.ShareResult, error)
	ArchiveTrip(ctx context.Context, id string) error
	Verify(ctx context.Context, email, password string) (views.UserIdentity, error)
	Preferences(ctx context.Context, userID string) (views.PreferencesData, error)
	UpdatePreferences(ctx context.Context, userID string, p views.PreferencesData) error
	Nearby(ctx context.Context, lat, lon, radius string) ([]views.TripView, error)
	SharedTrip(ctx context.Context, token string) (views.SharedData, error)
	GeoPoints(ctx context.Context) ([]views.GeoPoint, error)
	Insights(ctx context.Context) (views.InsightView, error)
	Activity(ctx context.Context) ([]views.ActivityView, error)
	IngestDemo(ctx context.Context) error
	Mailboxes(ctx context.Context) ([]views.MailboxView, error)
	AddMailbox(ctx context.Context, provider, email string) error
	Senders(ctx context.Context) ([]views.SenderView, error)
	AddSender(ctx context.Context, sender string) error
	RemoveSender(ctx context.Context, id string) error
}

type MapRenderer interface {
	Render(ctx context.Context, points []views.GeoPoint) ([]byte, error)
}

type Handlers struct {
	Store       Store
	MapRenderer MapRenderer
}

type component interface {
	Render(context.Context, io.Writer) error
}

func renderComponent(ctx context.Context, w http.ResponseWriter, c component) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = c.Render(ctx, w)
}

func (h *Handlers) page(w http.ResponseWriter, r *http.Request, title, active string, content component) {
	if r.Header.Get("HX-Request") == "true" {
		renderComponent(r.Context(), w, views.HTMXResponse(title, active, h.signedIn(r), content))
		return
	}
	renderComponent(r.Context(), w, views.Layout(title, active, h.signedIn(r), content))
}

func correlation(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Correlation-ID")
		if id == "" {
			b := make([]byte, 6)
			_, _ = rand.Read(b)
			id = hex.EncodeToString(b)
		}
		w.Header().Set("X-Correlation-ID", id)
		next.ServeHTTP(w, r)
	})
}

type visitor struct {
	count int
	reset time.Time
}

func rateLimit(next http.Handler) http.Handler {
	var mu sync.Mutex
	visitors := map[string]*visitor{}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		v, ok := visitors[r.RemoteAddr]
		if !ok || time.Now().After(v.reset) {
			v = &visitor{reset: time.Now().Add(time.Minute)}
			visitors[r.RemoteAddr] = v
		}
		v.count++
		mu.Unlock()
		if v.count > 240 {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "RATE_LIMITED", http.StatusTooManyRequests)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func (h *Handlers) data(ctx context.Context, status string) views.DashboardData {
	data := views.DashboardData{Active: status, Trips: []views.TripView{}, Activity: []views.ActivityView{}}
	trips, err := h.Store.Trips(ctx, status)
	if err != nil {
		log.Printf("pater/sqlc: no pudo listar viajes: %v", err)
	} else {
		data.Trips = trips
	}
	ins, err := h.Store.Insights(ctx)
	if err != nil {
		log.Printf("pater/sqlc: no pudo leer insights: %v", err)
	} else {
		data.Insights = ins
	}
	act, err := h.Store.Activity(ctx)
	if err != nil {
		log.Printf("pater/sqlc: no pudo leer actividad: %v", err)
	} else {
		data.Activity = act
	}
	return data
}

func (h *Handlers) tripListFragment(w http.ResponseWriter, r *http.Request, status string) {
	trips, err := h.Store.Trips(r.Context(), status)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	renderComponent(r.Context(), w, views.TripList(trips, status, true))
	renderComponent(r.Context(), w, views.FilterBar(status, true))
}

// Handler registra únicamente rutas HTML/HTMX.
func (h *Handlers) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(correlation, rateLimit)

	r.Get("/", func(w http.ResponseWriter, req *http.Request) {
		h.page(w, req, "Trotamundos · Panel de viajes", "panel", views.DashboardPage(h.data(req.Context(), "")))
	})
	r.Get("/trips", func(w http.ResponseWriter, req *http.Request) {
		h.tripListFragment(w, req, req.URL.Query().Get("status"))
	})
	r.Post("/trips", func(w http.ResponseWriter, req *http.Request) {
		if err := req.ParseForm(); err != nil {
			http.Error(w, "formulario inválido", http.StatusBadRequest)
			return
		}
		if err := h.Store.CreateTrip(req.Context(), req.FormValue("name"), req.FormValue("start_date"), req.FormValue("end_date")); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		h.tripListFragment(w, req, req.FormValue("status"))
	})
	r.Post("/bookings", func(w http.ResponseWriter, req *http.Request) {
		if err := req.ParseForm(); err != nil {
			http.Error(w, "formulario inválido", http.StatusBadRequest)
			return
		}
		if err := h.Store.AddBooking(req.Context(), req.FormValue("trip_id"), req.FormValue("type"),
			req.FormValue("provider"), req.FormValue("provider_ref")); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		h.tripListFragment(w, req, "")
	})
	r.Get("/insights", func(w http.ResponseWriter, req *http.Request) {
		ins, err := h.Store.Insights(req.Context())
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		renderComponent(req.Context(), w, views.InsightsBody(ins))
	})
	r.Post("/demo/ingest", func(w http.ResponseWriter, req *http.Request) {
		if err := h.Store.IngestDemo(req.Context()); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		// Desde la navegación global no existe #activity: se redibuja el panel
		// completo para que el botón nunca quede sin efecto.
		if req.Header.Get("HX-Target") != "activity" {
			h.page(w, req, "Trotamundos · Panel de viajes", "panel", views.DashboardPage(h.data(req.Context(), "")))
			return
		}
		act, _ := h.Store.Activity(req.Context())
		renderComponent(req.Context(), w, views.ActivityFeed(act))
		h.tripListFragment(w, req, "")
	})

	r.Get("/trips/view", func(w http.ResponseWriter, req *http.Request) {
		h.page(w, req, "Trotamundos · Viajes", "panel", views.TripsPage(h.data(req.Context(), req.URL.Query().Get("status"))))
	})
	r.Get("/trips/{id}", func(w http.ResponseWriter, req *http.Request) {
		h.renderTripDetail(w, req, chi.URLParam(req, "id"), "")
	})
	r.Post("/trips/{id}/share", h.shareTrip)
	r.Post("/trips/{id}/archive", h.archiveTrip)
	r.Get("/insights/view", h.insightsPage)
	r.Get("/mailboxes", h.mailboxes)
	r.Post("/mailboxes", h.addMailbox)
	r.Get("/whitelists", h.whitelists)
	r.Post("/whitelists", h.addSender)
	r.Delete("/whitelists/{id}", h.removeSender)
	r.Get("/preferences", h.preferences)
	r.Post("/preferences", h.updatePreferences)
	r.Get("/shared/{token}", h.sharedTrip)
	r.Get("/nearby", h.nearby)
	r.Get("/map", h.mapPage)
	r.Get("/map/image", h.mapImage)
	r.Get("/login", func(w http.ResponseWriter, req *http.Request) {
		h.page(w, req, "Trotamundos · Acceso", "panel", views.LoginPage(""))
	})
	r.Post("/auth/login", h.login)
	r.Post("/auth/logout", func(w http.ResponseWriter, req *http.Request) {
		ClearSession(w)
		h.page(w, req, "Trotamundos · Acceso", "panel", views.LoginPage("Sesión cerrada."))
	})
	r.Get("/healthz", func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		_, _ = w.Write([]byte(`{"status":"ok","unit":"pater"}`))
	})
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir("static"))))
	return r
}

func wantsHTMXFragment(r *http.Request) bool { return r.Header.Get("HX-Request") == "true" }

func (h *Handlers) renderTripDetail(w http.ResponseWriter, r *http.Request, id, notice string) {
	trip, err := h.Store.Trip(r.Context(), id)
	if err != nil {
		http.Error(w, "Viaje no encontrado", http.StatusNotFound)
		return
	}
	travelers, _ := h.Store.Travelers(r.Context(), id)
	h.page(w, r, "Trotamundos · "+trip.Name, "panel", views.TripDetailPage(views.TripDetailData{
		Trip: trip, Travelers: travelers, Notice: notice,
	}))
}

func (h *Handlers) shareTrip(w http.ResponseWriter, req *http.Request) {
	result, err := h.Store.ShareTrip(req.Context(), chi.URLParam(req, "id"))
	if err != nil {
		result = views.ShareResult{}
	}
	renderComponent(req.Context(), w, views.ShareLink(result))
}

func (h *Handlers) archiveTrip(w http.ResponseWriter, req *http.Request) {
	id := chi.URLParam(req, "id")
	if err := h.Store.ArchiveTrip(req.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	h.renderTripDetail(w, req, id, "Viaje archivado correctamente.")
}

func (h *Handlers) insightsPage(w http.ResponseWriter, req *http.Request) {
	data := views.InsightPageData{Insights: h.data(req.Context(), "").Insights}
	data.Activity, _ = h.Store.Activity(req.Context())
	h.page(w, req, "Trotamundos · Boletín", "insights", views.InsightsPage(data))
}

func (h *Handlers) mailboxes(w http.ResponseWriter, req *http.Request) {
	items, err := h.Store.Mailboxes(req.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	h.page(w, req, "Trotamundos · Buzones", "mailboxes", views.MailboxesPage(items, req.URL.Query().Get("notice")))
}

func (h *Handlers) addMailbox(w http.ResponseWriter, req *http.Request) {
	if err := req.ParseForm(); err != nil {
		http.Error(w, "formulario inválido", http.StatusBadRequest)
		return
	}
	notice := "Buzón vinculado."
	if err := h.Store.AddMailbox(req.Context(), req.FormValue("provider"), req.FormValue("email")); err != nil {
		notice = "No se pudo vincular: " + err.Error()
	}
	items, _ := h.Store.Mailboxes(req.Context())
	h.page(w, req, "Trotamundos · Buzones", "mailboxes", views.MailboxesPage(items, notice))
}

func (h *Handlers) whitelists(w http.ResponseWriter, req *http.Request) {
	items, err := h.Store.Senders(req.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	h.page(w, req, "Trotamundos · Remitentes", "whitelists", views.WhitelistsPage(items, req.URL.Query().Get("notice")))
}

func (h *Handlers) addSender(w http.ResponseWriter, req *http.Request) {
	if err := req.ParseForm(); err != nil {
		http.Error(w, "formulario inválido", http.StatusBadRequest)
		return
	}
	notice := "Remitente autorizado."
	if err := h.Store.AddSender(req.Context(), req.FormValue("sender")); err != nil {
		notice = "No se pudo autorizar: " + err.Error()
	}
	items, _ := h.Store.Senders(req.Context())
	h.page(w, req, "Trotamundos · Remitentes", "whitelists", views.WhitelistsPage(items, notice))
}

func (h *Handlers) removeSender(w http.ResponseWriter, req *http.Request) {
	if err := h.Store.RemoveSender(req.Context(), chi.URLParam(req, "id")); err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.WriteHeader(http.StatusOK)
}

func (h *Handlers) preferences(w http.ResponseWriter, req *http.Request) {
	user := h.currentUser(req)
	if user.ID == "" {
		h.page(w, req, "Trotamundos · Acceso", "preferences", views.LoginPage("Identifíquese para editar sus preferencias."))
		return
	}
	p, err := h.Store.Preferences(req.Context(), user.ID)
	if err != nil {
		p = views.PreferencesData{Locale: "es", Currency: "USD", GpsRadius: "50"}
	}
	h.page(w, req, "Trotamundos · Preferencias", "preferences", views.PreferencesPage(p, req.URL.Query().Get("notice")))
}

func (h *Handlers) updatePreferences(w http.ResponseWriter, req *http.Request) {
	user := h.currentUser(req)
	if user.ID == "" {
		http.Error(w, "UNAUTHORIZED", http.StatusUnauthorized)
		return
	}
	if err := req.ParseForm(); err != nil {
		http.Error(w, "formulario inválido", http.StatusBadRequest)
		return
	}
	p := views.PreferencesData{
		Locale: req.FormValue("locale"), Currency: req.FormValue("currency"),
		GpsRadius: req.FormValue("gps_radius_km"), PreferredAgency: req.FormValue("preferred_agency_id"),
	}
	notice := "Preferencias guardadas."
	if err := h.Store.UpdatePreferences(req.Context(), user.ID, p); err != nil {
		notice = "No se pudieron guardar: " + err.Error()
	}
	h.page(w, req, "Trotamundos · Preferencias", "preferences", views.PreferencesPage(p, notice))
}

func (h *Handlers) sharedTrip(w http.ResponseWriter, req *http.Request) {
	data, err := h.Store.SharedTrip(req.Context(), chi.URLParam(req, "token"))
	if err != nil {
		http.Error(w, "Enlace inválido o vencido", http.StatusNotFound)
		return
	}
	h.page(w, req, "Trotamundos · Itinerario compartido", "panel", views.SharedPage(data))
}

func (h *Handlers) nearby(w http.ResponseWriter, req *http.Request) {
	data := views.NearbyData{
		Lat: req.URL.Query().Get("lat"), Lon: req.URL.Query().Get("lon"), Radius: req.URL.Query().Get("radius_km"),
	}
	if data.Lat == "" {
		data.Lat = "-34.6037"
	}
	if data.Lon == "" {
		data.Lon = "-58.3816"
	}
	if data.Radius == "" {
		data.Radius = "50"
	}
	trips, err := h.Store.Nearby(req.Context(), data.Lat, data.Lon, data.Radius)
	if err == nil {
		data.Trips = trips
	}
	// El formulario actualiza solo #trip-list; una respuesta de página aquí
	// reemplazaría el formulario con el propio fragmento.
	if wantsHTMXFragment(req) && req.Header.Get("HX-Target") == "trip-list" {
		renderComponent(req.Context(), w, views.TripList(data.Trips, "", false))
		return
	}
	h.page(w, req, "Trotamundos · Cercanías", "panel", views.NearbyPage(data))
}

func (h *Handlers) login(w http.ResponseWriter, req *http.Request) {
	if err := req.ParseForm(); err != nil {
		http.Error(w, "formulario inválido", http.StatusBadRequest)
		return
	}
	identity, err := h.Store.Verify(req.Context(), req.FormValue("email"), req.FormValue("password"))
	if err != nil {
		h.page(w, req, "Trotamundos · Acceso", "panel", views.LoginPage("Credenciales incorrectas."))
		return
	}
	if err := SetSession(w, identity); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	h.page(w, req, "Trotamundos · Panel de viajes", "panel", views.DashboardPage(h.data(req.Context(), "")))
}

func (h *Handlers) mapPage(w http.ResponseWriter, req *http.Request) {
	points, err := h.Store.GeoPoints(req.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	h.page(w, req, "Trotamundos · Mapa de viajes", "map", views.MapPage(points))
}

func (h *Handlers) mapImage(w http.ResponseWriter, req *http.Request) {
	if h.MapRenderer == nil {
		http.Error(w, "renderizador de mapa no configurado", http.StatusServiceUnavailable)
		return
	}
	points, err := h.Store.GeoPoints(req.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	png, err := h.MapRenderer.Render(req.Context(), points)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	w.Header().Set("Content-Type", "image/png")
	w.Header().Set("Cache-Control", "public, max-age=900")
	w.Write(png)
}
