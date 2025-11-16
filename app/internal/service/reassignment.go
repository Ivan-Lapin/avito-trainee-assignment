package service

import (
	"avito/train-assignment/app/internal/repository"
	"database/sql"
	"errors"

	"github.com/jmoiron/sqlx"
)

type ReassignService struct {
	db    *sqlx.DB
	repoP *repository.PRRepo
	repoR *repository.ReviewersRepo
}

func NewReassignService(db *sqlx.DB, pr *repository.PRRepo, revs *repository.ReviewersRepo) *ReassignService {
	return &ReassignService{db: db, repoP: pr, repoR: revs}
}

var (
	ErrMerged      = errors.New("pr merged")
	ErrNotAssigned = errors.New("reviewer not assigned")
	ErrNoCandidate = errors.New("no active candidate")
)

// Reassign выполняет замену reviewerOld на reviewerNew в рамках транзакции.
func (s *ReassignService) Reassign(prID, reviewerOld, reviewerNew string) error {
	return repository.Tx(s.db, func(tx *sqlx.Tx) error {
		// Проверяем статус PR.
		var status string
		if err := tx.Get(&status, `select status from pull_requests where id=$1`, prID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return err
			}
			return err
		}
		if status == "MERGED" {
			return ErrMerged
		}

		// Проверяем, что старый ревьювер назначен.
		var cnt int
		if err := tx.Get(&cnt, `select count(1) from pr_reviewers where pr_id=$1 and reviewer_id=$2`, prID, reviewerOld); err != nil {
			return err
		}
		if cnt == 0 {
			return ErrNotAssigned
		}

		// Проверяем, что новый ещё не назначен и не превышаем лимит 2.
		var reviewers []string
		if err := tx.Select(&reviewers, `select reviewer_id from pr_reviewers where pr_id=$1`, prID); err != nil {
			return err
		}
		if len(reviewers) == 0 {
			// Формально допустимо, но в контексте «замены» ожидается >=1.
		}
		for _, id := range reviewers {
			if id == reviewerNew {
				// Уже есть — ничего не делаем.
				return nil
			}
		}
		// Удаляем старого и добавляем нового атомарно.
		if _, err := tx.Exec(`delete from pr_reviewers where pr_id=$1 and reviewer_id=$2`, prID, reviewerOld); err != nil {
			return err
		}
		if _, err := tx.Exec(`insert into pr_reviewers(pr_id,reviewer_id) values($1,$2) on conflict do nothing`, prID, reviewerNew); err != nil {
			return err
		}
		return nil
	})
}
