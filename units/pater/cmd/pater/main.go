// Comando de la unidad Pater (Rostro): SSR con templ/HTMX + sqlc en :8080.
package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"

	"trotamundos/pater/internal/logic"
	"trotamundos/pater/internal/maps"
	"trotamundos/pater/internal/web"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func main() {
	dsn := os.Getenv("PATER_PG_DSN")
	if dsn == "" {
		log.Fatal("PATER_PG_DSN es obligatorio (ADR-111: los handlers usan sqlc directo)")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("pool PostgreSQL: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("PostgreSQL no disponible: %v", err)
	}

	h := &web.Handlers{Store: logic.New(pool), MapRenderer: maps.NewCached()}
	port := env("PATER_PORT", "8080")
	log.Printf("Pater (Rostro + sqlc) escuchando en :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, h.Handler()))
}
