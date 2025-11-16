package service

import (
	"avito/internal/domain"
	"avito/internal/errs"
	"avito/internal/repository"
	"context"
	"errors"
)

type TeamService struct {
	teams repository.TeamRepository
	users repository.UserRepository
	prs   repository.PullRequestRepository
	prSvc *PullRequestService
}

func NewTeamService(tr repository.TeamRepository, ur repository.UserRepository, pr repository.PullRequestRepository) *TeamService {
	return &TeamService{
		teams: tr,
		users: ur,
		prs:   pr,
	}
}

func (s *TeamService) SetPullRequestService(prSvc *PullRequestService) {
	s.prSvc = prSvc
}

type BulkDeactivateResult struct {
	TeamName            string
	DeactivatedUsers    int64
	ReassignedReviewers int64
}

func (s *TeamService) BulkDeactivateTeam(ctx context.Context, teamName string) (*BulkDeactivateResult, error) {
	// получить всех открытых назначений для команды до деактивации
	assignments, err := s.prs.GetOpenAssignmentsByTeam(ctx, teamName)
	if err != nil {
		return nil, err
	}

	// деактивировать пользователей команды
	deactivated, err := s.users.DeactivateByTeam(ctx, teamName)
	if err != nil {
		return nil, err
	}

	var reassigned int64
	// безопасно перебрать все назначения и вызвать нашу обычную Reassign‑логику
	for _, a := range assignments {
		err := s.prSvc.ReassignReviewer(ctx, a.PRID, a.UserID)
		if err == nil {
			reassigned++
			continue
		}
	}

	return &BulkDeactivateResult{
		TeamName:            teamName,
		DeactivatedUsers:    deactivated,
		ReassignedReviewers: reassigned,
	}, nil
}

func (s *TeamService) CreateTeam(team domain.Team) (*domain.Team, error) {
	// Проверяем, что команда ещё не существует
	if _, err := s.teams.GetTeam(team.TeamName); err == nil {
		return nil, errs.New(errs.CodeTeamExists, "team_name already exists")
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	if err := s.teams.CreateTeam(team); err != nil {
		return nil, err
	}

	for _, m := range team.Members {
		u := domain.User{
			ID:       m.UserID,
			Username: m.Username,
			TeamName: team.TeamName,
			IsActive: m.IsActive,
		}
		if err := s.users.UpsertUser(u); err != nil {
			return nil, err
		}
	}

	return &team, nil
}

func (s *TeamService) GetTeam(name string) (*domain.Team, error) {
	team, err := s.teams.GetTeam(name)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errs.New(errs.CodeNotFound, "team not found")
		}
		return nil, err
	}
	return team, nil
}
