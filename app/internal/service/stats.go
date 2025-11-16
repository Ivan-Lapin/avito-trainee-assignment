package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

type StatsService struct {
	db  *sqlx.DB
	rdb *redis.Client
}

func NewStatsService(db *sqlx.DB, rdb *redis.Client) *StatsService {
	return &StatsService{db: db, rdb: rdb}
}

type AssignStats struct {
	ByReviewer map[string]int `json:"by_reviewer"`
	ByPR       map[string]int `json:"by_pr"`
}

// Compute возвращает агрегаты с 30-секундным кэшированием.
func (s *StatsService) Compute(ctx context.Context) (AssignStats, error) {
	// Попробуем из кеша.
	if b, err := s.rdb.Get(ctx, "stats:assignments").Bytes(); err == nil && len(b) > 0 {
		var cached AssignStats
		if json.Unmarshal(b, &cached) == nil {
			return cached, nil
		}
	}

	var byReviewer = map[string]int{}
	var byPR = map[string]int{}

	type row struct {
		ID  string `db:"id"`
		Cnt int    `db:"cnt"`
	}
	var r1 []row
	if err := s.db.Select(&r1, `select reviewer_id as id, count(*) as cnt from pr_reviewers group by reviewer_id`); err == nil {
		for _, v := range r1 {
			byReviewer[v.ID] = v.Cnt
		}
	}
	var r2 []row
	if err := s.db.Select(&r2, `select pr_id as id, count(*) as cnt from pr_reviewers group by pr_id`); err == nil {
		for _, v := range r2 {
			byPR[v.ID] = v.Cnt
		}
	}

	out := AssignStats{ByReviewer: byReviewer, ByPR: byPR}
	if b, err := json.Marshal(out); err == nil {
		_ = s.rdb.Set(ctx, "stats:assignments", b, 30*time.Second).Err()
	}
	return out, nil
}
