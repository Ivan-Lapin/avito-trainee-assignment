package repository

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
)

type PR struct {
	ID       string `db:"id"`
	Title    string `db:"title"`
	AuthorID string `db:"author_id"`
	Status   string `db:"status"`
}

type PRRepo struct{ db *sqlx.DB }

func NewPRRepo(db *sqlx.DB) *PRRepo { return &PRRepo{db: db} }

func (r *PRRepo) Create(tx *sqlx.Tx, id, title, authorID string) error {
	_, err := tx.Exec(`insert into pull_requests(id,title,author_id,status) values($1,$2,$3,'OPEN')`, id, title, authorID)
	return err
}

func (r *PRRepo) Get(id string) (PR, error) {
	var pr PR
	err := r.db.Get(&pr, `select id,title,author_id,status from pull_requests where id=$1`, id)
	if err == sql.ErrNoRows {
		return PR{}, sql.ErrNoRows
	}
	return pr, err
}

func (r *PRRepo) Merge(tx *sqlx.Tx, id string) error {
	_, err := tx.Exec(`update pull_requests set status='MERGED', merged_at=now() where id=$1 and status='OPEN'`, id)
	return err
}
