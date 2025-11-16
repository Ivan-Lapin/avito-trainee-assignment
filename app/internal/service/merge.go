package service

import (
	"avito/train-assignment/app/internal/repository"
	"context"
	"errors"
	"fmt"

	"github.com/jmoiron/sqlx"
	"github.com/redis/go-redis/v9"
)

type MergeService struct {
	db  *sqlx.DB
	pr  *repository.PRRepo
	rdb *redis.Client
}

func NewMergeService(db *sqlx.DB, pr *repository.PRRepo, rdb *redis.Client) *MergeService {
	return &MergeService{db: db, pr: pr, rdb: rdb}
}

// MergeIdempotent выполняет merge c идемпотентностью по ключу.
// Если ключ не передан, идемпотентность обеспечивается по состоянию PR (MERGED повторно не изменит его).
func (s *MergeService) MergeIdempotent(ctx context.Context, prID, idemKey string) error {
	if idemKey != "" {
		cacheKey := fmt.Sprintf("idem:merge:%s", idemKey)
		// Если уже есть запись — считаем, что операция уже успешно прошла.
		exists, err := s.rdb.Exists(ctx, cacheKey).Result()
		if err == nil && exists == 1 {
			return nil
		}
	}

	// Атомарно перевести OPEN -> MERGED; повторный вызов безопасен.
	err := repository.Tx(s.db, func(tx *sqlx.Tx) error {
		// Простое обновление; если уже MERGED — update не изменит данные.
		return s.pr.Merge(tx, prID)
	})
	if err != nil && !errors.Is(err, nil) {
		return err
	}

	if idemKey != "" {
		// Фиксируем факт успешной операции для повторов в окне TTL.
		_ = s.rdb.Set(ctx, "idem:merge:"+idemKey, "ok", 60*60).Err()
	}
	return nil
}
