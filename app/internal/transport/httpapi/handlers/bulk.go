package handlers

import (
	"avito/train-assignment/app/internal/service"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
)

func RegisterBulk(r chi.Router, db *sqlx.DB) {

	svc := service.NewBulkService(db)
	r.Post("/team/deactivateAndReassign", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			TeamName string `json:"team_name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.TeamName == "" {
			BadRequest(w, "team_name required")
			return
		}
		if err := svc.DeactivateTeamAndReassign(r.Context(), req.TeamName); err != nil {
			Internal(w, "bulk op failed")
			return
		}
		w.WriteHeader(http.StatusOK)
	})
}
