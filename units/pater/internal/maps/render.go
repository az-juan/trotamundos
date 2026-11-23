// Package maps renderiza un mapa real con teselas OpenStreetMap usando go-staticmaps.
// La imagen se genera en servidor y se sirve como PNG; el navegador no ejecuta JS de mapas.
package maps

import (
	"bytes"
	"context"
	"fmt"
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

// Render devuelve el PNG del mapa con marcadores y traza entre las reservas geolocalizadas.
func (r *Renderer) Render(_ context.Context, points []views.GeoPoint) ([]byte, error) {
	if len(points) == 0 {
		return nil, fmt.Errorf("no hay puntos geolocalizados")
	}
	ctx := sm.NewContext()
	ctx.SetSize(880, 480)
	ctx.SetTileProvider(sm.NewTileProviderOpenStreetMaps())
	ctx.SetUserAgent("Trotamundos-Fase0/1.0 (mapa educativo; contacto: equipo@trotamundos.local)")
	cacheDir := filepath.Join(os.TempDir(), "trotamundos-tiles")
	if err := os.MkdirAll(cacheDir, 0o755); err == nil {
		ctx.SetCache(sm.NewTileCache(cacheDir, 0o755))
	}

	bordeaux := color.RGBA{R: 84, G: 11, B: 14, A: 255}
	bronze := color.RGBA{R: 224, G: 159, B: 62, A: 255}
	positions := make([]s2.LatLng, 0, len(points))
	for _, p := range points {
		pos := s2.LatLngFromDegrees(p.Lat, p.Lon)
		positions = append(positions, pos)
		marker := sm.NewMarker(pos, bordeaux, 14)
		marker.Label = p.Provider + " · " + p.ProviderRef
		marker.SetLabelColor(color.RGBA{R: 51, G: 35, B: 24, A: 255})
		marker.LabelXOffset = 8
		marker.LabelYOffset = -8
		ctx.AddMarker(marker)
	}
	if len(positions) > 1 {
		ctx.AddPath(sm.NewPath(positions, bronze, 4))
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
	} else {
		ctx.SetCenter(positions[0])
		ctx.SetZoom(13)
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
type CachedRenderer struct {
	inner *Renderer
	ttl   time.Duration
	mu    chan struct{}
	data  []byte
	at    time.Time
}

func NewCached() *CachedRenderer {
	return &CachedRenderer{inner: New(), ttl: 15 * time.Minute, mu: make(chan struct{}, 1)}
}

func (c *CachedRenderer) Render(ctx context.Context, points []views.GeoPoint) ([]byte, error) {
	c.mu <- struct{}{}
	defer func() { <-c.mu }()
	if c.data != nil && time.Since(c.at) < c.ttl {
		return c.data, nil
	}
	data, err := c.inner.Render(ctx, points)
	if err != nil {
		return nil, err
	}
	c.data, c.at = data, time.Now()
	return data, nil
}
