package httpapi

import (
	"avito/train-assignment/app/internal/config"
	"avito/train-assignment/app/internal/transport/httpapi/handlers"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

// Mount регистрирует все конечные точки API.
func Mount(r *chi.Mux, db *sqlx.DB, rdb *redis.Client, cfg config.Config) {
	// Здесь можно навесить middleware rate limiting / idempotency / dedupe.
	// r.Use(ratelimit.Middleware(rdb, cfg))
	// r.Use(idempotency.Middleware(rdb))
	// r.Use(dedupe.Middleware(rdb))

	r.Route("/api", func(api chi.Router) {
		api.Get("/health", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("ok"))
		})

		handlers.RegisterTeams(api, db)
		handlers.RegisterUsers(api, db)
		handlers.RegisterPRs(api, db, rdb)
		handlers.RegisterStats(api, db, rdb)
		handlers.RegisterBulk(api, db)
	})
}
