package domain

import "errors"

type PRStatus string

const (
	PROpen   PRStatus = "OPEN"
	PRMerged PRStatus = "MERGED"
)

var (
	ErrNotFound    = errors.New("not found")
	ErrPRMerged    = errors.New("pr merged")
	ErrNoCandidate = errors.New("no candidate")
	ErrNotAssigned = errors.New("not assigned")
)

type User struct {
	ID       string
	Username string
	IsActive bool
}

type Team struct {
	ID   string
	Name string
}

type PullRequest struct {
	ID        string
	Title     string
	AuthorID  string
	Status    PRStatus
	CreatedAt int64
	MergedAt  *int64
}
