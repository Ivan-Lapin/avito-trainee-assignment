package handlers

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

func RegisterTeams(r chi.Router, db *sqlx.DB) {
	r.Post("/team/add", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Name string `json:"team_name"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)
		id := uuid.NewString()
		if _, err := db.Exec(`insert into teams(id,name) values($1,$2)`, id, req.Name); err != nil {
			http.Error(w, `{"error":{"code":"TEAM_EXISTS","message":"team exists"}}`, http.StatusConflict)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"team_name": req.Name,
			"team_id":   id,
		})
	})
}
