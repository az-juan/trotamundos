// Comando de la unidad Spiritus (Mente): ingesta y analítica en :8082.
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"trotamundos/shared/event"
	"trotamundos/spiritus/internal/api"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	s := api.NewInMemoryServer(env("FILIUS_URL", "http://localhost:8081"))

	// Consumidores JetStream del read-model (arista Filius↔Spiritus, ADR-103/107).
	if natsURL := os.Getenv("NATS_URL"); natsURL != "" {
		ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		defer cancel()
		for _, sub := range []struct{ subject, durable string }{
			{"booking.>", "spiritus-bookings"},
			{"trip.>", "spiritus-trips"},
			{"assistance.>", "spiritus-assistance"},
		} {
			go func(subject, durable string) {
				if err := event.Subscribe(ctx, natsURL, subject, durable, func(e event.Event) error {
					s.Store.Add(e)
					return nil
				}); err != nil {
					log.Printf("suscripción NATS %s: %v", subject, err)
				}
			}(sub.subject, sub.durable)
		}
		log.Printf("Spiritus suscripto a NATS JetStream (%s)", natsURL)
	}

	port := env("SPIRITUS_PORT", "8082")
	log.Printf("Spiritus (Mente) escuchando en :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, s.Handler()))
}
