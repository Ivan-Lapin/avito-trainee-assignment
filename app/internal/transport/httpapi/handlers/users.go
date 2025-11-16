package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

func RegisterUsers(r chi.Router, db *sqlx.DB) {
	r.Post("/user/activity", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			UserID   string `json:"user_id"`
			IsActive bool   `json:"is_active"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		if _, err := db.Exec(`update users set is_active=$2 where id=$1`, req.UserID, req.IsActive); err != nil {
			http.Error(w, `{"error":{"code":"NOT_FOUND","message":"user"}}`, http.StatusNotFound)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}
