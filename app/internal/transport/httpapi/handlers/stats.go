package handlers

import (
	"avito/train-assignment/app/internal/service"
	"context"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

func RegisterStats(r chi.Router, db *sqlx.DB, rdb *redis.Client) {

	svc := service.NewStatsService(db, rdb)
	r.Get("/stats/assignments", func(w http.ResponseWriter, r *http.Request) {
		out, err := svc.Compute(context.Background())
		if err != nil {
			Internal(w, "stats error")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(out)
	})
}
