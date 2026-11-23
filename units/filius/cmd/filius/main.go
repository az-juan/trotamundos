// Comando de la unidad Filius (Núcleo): API REST en :8081.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"trotamundos/filius/internal/api"
	"trotamundos/filius/internal/store"
	"trotamundos/shared/event"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	// Arista Filius↔Spiritus: JetStream si NATS_URL está disponible; HTTP como fallback del prototipo.
	var publisher event.Publisher = event.HTTPPublisher{
		BaseURL: env("SPIRITUS_URL", "http://localhost:8082"),
		Path:    "/api/events",
	}
	if natsURL := os.Getenv("NATS_URL"); natsURL != "" {
		np, err := event.NewNATSPublisher(natsURL)
		if err != nil {
			log.Printf("NATS no disponible, se usa HTTP: %v", err)
		} else {
			publisher = np
			defer np.Close()
			log.Printf("Filius publica en NATS JetStream (%s)", natsURL)
		}
	}
	s := &api.Server{Store: store.New(), Publisher: publisher}
	s.Store.Seed(time.Now())
	u := s.Store.SeedUser("dev@trotamundos.app", "dev", "Viajero Dev")
	log.Printf("usuario demo: %s", u.Email)
	if trips := s.Store.List(""); len(trips) > 0 {
		s.Store.AddTraveler(trips[0].ID, "Ana Ruiz", "12345678Z", "ana@example.com")
		for _, b := range trips[0].Bookings {
			s.Store.SetBookingCoords(b.ProviderRef, -34.6037, -58.3816)
		}
	}

	// FR-5: auto-archivado periódico de viajes finalizados (job en Filius).
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			for _, t := range s.Store.ArchiveCompleted(time.Now()) {
				_ = publisher.Publish(context.Background(), event.New(event.TripArchived, "filius",
					map[string]any{"trip_id": t.ID, "archived_at": t.ArchivedAt, "reason": "completed"}))
			}
		}
	}()

	port := env("FILIUS_PORT", "8081")
	log.Printf("Filius (Núcleo) escuchando en :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, s.Handler()))
}
