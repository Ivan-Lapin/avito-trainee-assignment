package handlers

import (
	"avito/train-assignment/app/internal/metrics"
	"avito/train-assignment/app/internal/repository"
	"avito/train-assignment/app/internal/service"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

func RegisterPRs(r chi.Router, db *sqlx.DB, rdb *redis.Client) {

	prRepo := repository.NewPRRepo(db)
	revRepo := repository.NewReviewersRepo(db)
	assign := service.NewAssignmentService(db, prRepo, revRepo)
	merge := service.NewMergeService(db, prRepo, rdb)

	r.Post("/pullRequest/create", func(w http.ResponseWriter, r *http.Request) {
		// Пример: ожидаем JSON {id,title,author_id,team_name}
		var req struct {
			ID       string `json:"id"`
			Title    string `json:"title"`
			AuthorID string `json:"author_id"`
			TeamName string `json:"team_name"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error":{"code":"BAD_REQUEST","message":"invalid json"}}`, http.StatusBadRequest)
			return
		}

		// Создать PR в транзакции.
		if err := repository.Tx(db, func(tx *sqlx.Tx) error {
			return prRepo.Create(tx, req.ID, req.Title, req.AuthorID)
		}); err != nil {
			http.Error(w, `{"error":{"code":"PR_EXISTS","message":"pr exists"}}`, http.StatusConflict)
			return
		}

		// Выбрать активных членов команды автора, исключая автора.
		var candidates []string
		const q = `
select u.id from users u
join team_members tm on tm.user_id=u.id
join teams t on t.id=tm.team_id
where t.name=$1 and u.is_active=true and u.id<>$2
`
		if err := db.Select(&candidates, q, req.TeamName, req.AuthorID); err != nil {
			http.Error(w, `{"error":{"code":"NOT_FOUND","message":"team or users"}}`, http.StatusNotFound)
			return
		}
		_ = assign.AssignInitial(req.ID, req.AuthorID, candidates)

		metrics.AssignmentsTotal.Inc()

		// Ответ: короткая форма PR с назначениями (упрощенно).
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":        req.ID,
			"title":     req.Title,
			"author_id": req.AuthorID,
			"status":    "OPEN",
			"reviewers": candidates, // для наглядности; в реале — запросить фактический список из pr_reviewers
		})
	})

	r.Post("/pullRequest/merge", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			ID string `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)

		idemKey := r.Header.Get("Idempotency-Key")
		if err := merge.MergeIdempotent(context.Background(), req.ID, idemKey); err != nil {
			// Если PR не найден.
			if err == sql.ErrNoRows {
				http.Error(w, `{"error":{"code":"NOT_FOUND","message":"pr"}}`, http.StatusNotFound)
				return
			}
			http.Error(w, `{"error":{"code":"INTERNAL","message":"merge failed"}}`, http.StatusInternalServerError)
			return
		}

		metrics.MergesTotal.Inc()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"id":     req.ID,
			"status": "MERGED",
		})
	})

	r.Post("/pullRequest/reassign", func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			PRID        string `json:"pr_id"`
			ReviewerOld string `json:"reviewer_old"`
			TeamName    string `json:"team_name"`
		}
		_ = json.NewDecoder(r.Body).Decode(&req)

		// Проверка статуса PR.
		var status string
		if err := db.Get(&status, `select status from pull_requests where id=$1`, req.PRID); err != nil {
			http.Error(w, `{"error":{"code":"NOT_FOUND","message":"pr"}}`, http.StatusNotFound)
			return
		}
		if status == "MERGED" {
			http.Error(w, `{"error":{"code":"PR_MERGED","message":"cannot reassign merged"}}`, http.StatusConflict)
			return
		}

		// Убедиться, что reviewerOld был назначен.
		var cnt int
		_ = db.Get(&cnt, `select count(1) from pr_reviewers where pr_id=$1 and reviewer_id=$2`, req.PRID, req.ReviewerOld)
		if cnt == 0 {
			http.Error(w, `{"error":{"code":"NOT_ASSIGNED","message":"reviewer not assigned"}}`, http.StatusConflict)
			return
		}

		// Выбрать активных кандидатов из команды reviewerOld.
		var candidates []string
		const q = `
select u.id from users u
join team_members tm on tm.user_id=u.id
join teams t on t.id=tm.team_id
where t.name=$1 and u.is_active=true and u.id<>$2
`
		if err := db.Select(&candidates, q, req.TeamName, req.ReviewerOld); err != nil || len(candidates) == 0 {
			http.Error(w, `{"error":{"code":"NO_CANDIDATE","message":"no active candidate"}}`, http.StatusConflict)
			return
		}
		// Упрощенно берем первого кандидата; можно рандомизировать.
		newCandidate := candidates[0]

		assign := service.NewAssignmentService(db, prRepo, repository.NewReviewersRepo(db))
		if err := assign.Reassign(req.PRID, req.ReviewerOld, newCandidate); err != nil {
			http.Error(w, `{"error":{"code":"INTERNAL","message":"reassign failed"}}`, http.StatusInternalServerError)
			return
		}
		metrics.ReassignmentsTotal.Inc()

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"pr_id":        req.PRID,
			"reviewer_old": req.ReviewerOld,
			"reviewer_new": newCandidate,
			"reassigned":   true,
		})
	})
}
