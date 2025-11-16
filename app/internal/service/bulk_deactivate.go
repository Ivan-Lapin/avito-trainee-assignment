package service

import (
	"avito/train-assignment/app/internal/repository"
	"context"
	"database/sql"

	"github.com/jmoiron/sqlx"
)

type BulkService struct {
	db *sqlx.DB
}

func NewBulkService(db *sqlx.DB) *BulkService {
	return &BulkService{db: db}
}

// DeactivateTeamAndReassign:
//  1. Деактивирует всех пользователей team_name.
//  2. Для OPEN PR, где они назначены ревьюверами, пытается подобрать активных кандидатов из команды заменяемого
//     и заменить; если нет кандидатов, удаляет ревьювера (слот остаётся свободным).
func (s *BulkService) DeactivateTeamAndReassign(ctx context.Context, teamName string) error {
	return repository.Tx(s.db, func(tx *sqlx.Tx) error {
		// Деактивация пользователей команды.
		_, err := tx.Exec(`
update users set is_active=false
where id in (
  select u.id from users u
  join team_members tm on tm.user_id=u.id
  join teams t on t.id=tm.team_id
  where t.name=$1
)`, teamName)
		if err != nil {
			return err
		}

		// Найти (pr_id, reviewer_id) для OPEN PR, где reviewer из этой команды.
		type pair struct{ PRID, ReviewerID string }
		var pairs []pair
		if err := tx.Select(&pairs, `
select pr.id as pr_id, prr.reviewer_id
from pr_reviewers prr
join pull_requests pr on pr.id = prr.pr_id
where pr.status='OPEN' and prr.reviewer_id in (
  select u.id from users u
  join team_members tm on tm.user_id=u.id
  join teams t on t.id=tm.team_id
  where t.name=$1
)
`, teamName); err != nil && err != sql.ErrNoRows {
			return err
		}

		// Для каждого заменить на активного из команды заменяемого ревьювера.
		for _, p := range pairs {
			// Кандидаты из команды заменяемого ревьювера, активные, исключая автора и уже назначенных.
			var candidate string
			// Автор PR:
			var author string
			if err := tx.Get(&author, `select author_id from pull_requests where id=$1`, p.PRID); err != nil {
				return err
			}
			// Уже назначенные ревьюверы:
			var current []string
			if err := tx.Select(&current, `select reviewer_id from pr_reviewers where pr_id=$1`, p.PRID); err != nil {
				return err
			}

			// Найти активного кандидата из команды reviewer_id
			if err := tx.Get(&candidate, `
select u.id
from users u
join team_members tm on tm.user_id=u.id
join teams t on t.id=tm.team_id
where u.is_active=true
  and t.id = (select tm2.team_id from team_members tm2 where tm2.user_id=$1 limit 1)
  and u.id<>$1 and u.id<>$2
  and not exists (select 1 from pr_reviewers r where r.pr_id=$3 and r.reviewer_id=u.id)
limit 1
`, p.ReviewerID, author, p.PRID); err == nil && candidate != "" {
				// swap: удалить старого, добавить нового
				if _, err := tx.Exec(`delete from pr_reviewers where pr_id=$1 and reviewer_id=$2`, p.PRID, p.ReviewerID); err != nil {
					return err
				}
				if _, err := tx.Exec(`insert into pr_reviewers(pr_id,reviewer_id) values($1,$2) on conflict do nothing`, p.PRID, candidate); err != nil {
					return err
				}
			} else {
				// Кандидатов нет — просто удалить неактивного ревьювера
				if _, err := tx.Exec(`delete from pr_reviewers where pr_id=$1 and reviewer_id=$2`, p.PRID, p.ReviewerID); err != nil {
					return err
				}
			}
		}
		return nil
	})
}
