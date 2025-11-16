package repository

import "github.com/jmoiron/sqlx"

type ReviewersRepo struct{ db *sqlx.DB }

func NewReviewersRepo(db *sqlx.DB) *ReviewersRepo { return &ReviewersRepo{db: db} }

func (r *ReviewersRepo) ListForPR(prID string) ([]string, error) {
	var ids []string
	err := r.db.Select(&ids, `select reviewer_id from pr_reviewers where pr_id=$1`, prID)
	return ids, err
}

func (r *ReviewersRepo) Add(tx *sqlx.Tx, prID, reviewerID string) error {
	_, err := tx.Exec(`insert into pr_reviewers(pr_id,reviewer_id) values($1,$2) on conflict do nothing`, prID, reviewerID)
	return err
}

func (r *ReviewersRepo) Remove(tx *sqlx.Tx, prID, reviewerID string) error {
	_, err := tx.Exec(`delete from pr_reviewers where pr_id=$1 and reviewer_id=$2`, prID, reviewerID)
	return err
}
