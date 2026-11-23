// Package maps renderiza un mapa real con teselas OpenStreetMap usando go-staticmaps.
// La imagen se genera en servidor y se sirve como PNG; el navegador no ejecuta JS de mapas.
package maps

import (
	"bytes"
	"context"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"time"

	sm "github.com/flopp/go-staticmaps"
	"github.com/golang/geo/s2"

	"trotamundos/pater/views"
)

type Renderer struct{}

func New() *Renderer { return &Renderer{} }

// Un color por estado de viaje: próximos (bronce), en curso (pizarra) y
// archivados (burdeos). La leyenda del panel usa los mismos colores.
var statusColors = map[string]color.RGBA{
	"upcoming": {R: 224, G: 159, B: 62, A: 255},
	"active":   {R: 51, G: 92, B: 103, A: 255},
	"archived": {R: 84, G: 11, B: 14, A: 255},
}

func markerColor(status string) color.RGBA {
	if c, ok := statusColors[status]; ok {
		return c
	}
	return statusColors["upcoming"]
}

// Render devuelve el PNG del mapa con marcadores por estado y una traza por viaje.
func (r *Renderer) Render(_ context.Context, points []views.GeoPoint) ([]byte, error) {
	ctx := sm.NewContext()
	ctx.SetSize(880, 480)
	ctx.SetTileProvider(sm.NewTileProviderOpenStreetMaps())
	ctx.SetUserAgent("Trotamundos-Fase0/1.0 (mapa educativo; contacto: equipo@trotamundos.local)")
	cacheDir := filepath.Join(os.TempDir(), "trotamundos-tiles")
	if err := os.MkdirAll(cacheDir, 0o755); err == nil {
		ctx.SetCache(sm.NewTileCache(cacheDir, 0o755))
	}

	positions := make([]s2.LatLng, 0, len(points))
	paths := map[string][]s2.LatLng{}
	tripStatus := map[string]string{}
	order := []string{}
	for _, p := range points {
		pos := s2.LatLngFromDegrees(p.Lat, p.Lon)
		positions = append(positions, pos)
		marker := sm.NewMarker(pos, markerColor(p.Status), 14)
		marker.Label = views.StatusLabel(p.Status) + " · " + p.Provider
		marker.SetLabelColor(color.RGBA{R: 51, G: 35, B: 24, A: 255})
		marker.LabelXOffset = 8
		marker.LabelYOffset = -8
		ctx.AddMarker(marker)

		if _, seen := paths[p.TripID]; !seen {
			order = append(order, p.TripID)
			tripStatus[p.TripID] = p.Status
		}
		paths[p.TripID] = append(paths[p.TripID], pos)
	}
	for _, id := range order {
		if len(paths[id]) > 1 {
			ctx.AddPath(sm.NewPath(paths[id], markerColor(tripStatus[id]), 3))
		}
	}

	// Encaje de la vista: si hay varias reservas, calcular límites con margen; si no, zoom fijo.
	if len(positions) > 1 {
		north, south := -90.0, 90.0
		west, east := 180.0, -180.0
		for _, p := range points {
			north = math.Max(north, p.Lat)
			south = math.Min(south, p.Lat)
			west = math.Min(west, p.Lon)
			east = math.Max(east, p.Lon)
		}
		margin := 0.010
		bbox, err := sm.CreateBBox(north+margin, west-margin, south-margin, east+margin)
		if err == nil {
			ctx.SetBoundingBox(*bbox)
		} else {
			ctx.SetCenter(positions[0])
			ctx.SetZoom(13)
		}
	} else if len(positions) == 1 {
		ctx.SetCenter(positions[0])
		ctx.SetZoom(13)
	} else {
		// Filtro sin reservas geolocalizadas: vista de contexto sin marcadores.
		ctx.SetCenter(s2.LatLngFromDegrees(-34.6037, -58.3816))
		ctx.SetZoom(5)
	}
	ctx.SetMaxZoom(16)

	img, err := ctx.Render()
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

// Cache simple en memoria para no descargar teselas en cada petición del navegador.
type cachedImage struct {
	data []byte
	at   time.Time
}

type CachedRenderer struct {
	inner  *Renderer
	ttl    time.Duration
	mu     chan struct{}
	images map[string]cachedImage
}

func NewCached() *CachedRenderer {
	return &CachedRenderer{inner: New(), ttl: 15 * time.Minute, mu: make(chan struct{}, 1), images: map[string]cachedImage{}}
}

// Render cachea por filtro de estado: el mapa de "en curso" no debe servir la
// imagen de "archivados" aunque cambie el query param.
func (c *CachedRenderer) Render(ctx context.Context, points []views.GeoPoint, status string) ([]byte, error) {
	c.mu <- struct{}{}
	defer func() { <-c.mu }()
	if img, ok := c.images[status]; ok && time.Since(img.at) < c.ttl {
		return img.data, nil
	}
	data, err := c.inner.Render(ctx, points)
	if err != nil {
		return nil, err
	}
	c.images[status] = cachedImage{data: data, at: time.Now()}
	return data, nil
}
