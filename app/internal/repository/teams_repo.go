package repository

import "github.com/jmoiron/sqlx"

type TeamsRepo struct{ db *sqlx.DB }

func NewTeamsRepo(db *sqlx.DB) *TeamsRepo { return &TeamsRepo{db: db} }

func (r *TeamsRepo) CreateTeam(name, id string) error {
	_, err := r.db.Exec(`insert into teams(id,name) values($1,$2)`, id, name)
	return err
}
