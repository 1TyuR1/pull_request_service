package repository

import (
	"avito/internal/domain"
	"context"
	"errors"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrAlreadyExists = errors.New("already exists")
)

type Stats struct {
	PerReviewer map[string]int64
	PerStatus   map[string]int64
}

type ReviewerAssignment struct {
	PRID   string
	UserID string
}

type TeamRepository interface {
	CreateTeam(team domain.Team) error
	GetTeam(name string) (*domain.Team, error)
}

type UserRepository interface {
	UpsertUser(u domain.User) error
	GetUser(userID string) (*domain.User, error)
	SetUserActive(userID string, isActive bool) (*domain.User, error)
	GetActiveUsersByTeam(teamName string, excludeIDs []string) ([]domain.User, error)
	DeactivateByTeam(ctx context.Context, teamName string) (int64, error)
}

type PullRequestRepository interface {
	CreatePR(pr domain.PullRequest) error
	GetPR(id string) (*domain.PullRequest, error)
	UpdatePR(pr domain.PullRequest) error
	GetPRsByReviewer(userID string) ([]domain.PullRequest, error)
	GetOpenAssignmentsByTeam(ctx context.Context, teamName string) ([]ReviewerAssignment, error)
	GetStats(ctx context.Context) (Stats, error)
}
