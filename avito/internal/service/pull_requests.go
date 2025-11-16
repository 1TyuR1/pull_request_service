package service

import (
	"avito/internal/domain"
	"avito/internal/errs"
	"avito/internal/repository"
	"context"
	"errors"
	"math/rand"
	"slices"
	"time"
)

type PullRequestService struct {
	prs   repository.PullRequestRepository
	users repository.UserRepository
	teams repository.TeamRepository
}

func NewPullRequestService(
	prRepo repository.PullRequestRepository,
	userRepo repository.UserRepository,
	teamRepo repository.TeamRepository,
) *PullRequestService {
	return &PullRequestService{
		prs:   prRepo,
		users: userRepo,
		teams: teamRepo,
	}
}

// Create создаёт новый PR и автоматически назначает до двух активных ревьюверов
// из команды автора, исключая самого автора.
func (s *PullRequestService) Create(
	ctx context.Context,
	id string,
	name string,
	authorID string,
) (*domain.PullRequest, error) {
	// проверяем, что PR с таким ID ещё не существует
	if _, err := s.prs.GetPR(id); err == nil {
		return nil, errs.New(errs.CodePRExists, "pull_request_id already exists")
	} else if !errors.Is(err, repository.ErrNotFound) {
		return nil, err
	}

	// получаем автора и его команду
	author, err := s.users.GetUser(authorID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errs.New(errs.CodeNotFound, "author not found")
		}
		return nil, err
	}

	team, err := s.teams.GetTeam(author.TeamName)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errs.New(errs.CodeNotFound, "team not found")
		}
		return nil, err
	}

	// собираем активных кандидатов, исключая автора
	candidates := make([]string, 0, len(team.Members))
	for _, m := range team.Members {
		if !m.IsActive {
			continue
		}
		if m.UserID == authorID {
			continue
		}
		candidates = append(candidates, m.UserID)
	}

	// перемешиваем и берём до двух
	rand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})

	assigned := candidates
	if len(assigned) > 2 {
		assigned = assigned[:2]
	}

	now := time.Now().UTC()
	pr := domain.PullRequest{
		ID:                id,
		Name:              name,
		AuthorID:          authorID,
		Status:            domain.PRStatusOpen, // "OPEN"
		AssignedReviewers: assigned,
		CreatedAt:         &now,
	}

	if err := s.prs.CreatePR(pr); err != nil {
		if errors.Is(err, repository.ErrAlreadyExists) {
			return nil, errs.New(errs.CodePRExists, "pull_request_id already exists")
		}
		return nil, err
	}

	return &pr, nil
}

// Merge переводит PR в статус MERGED и устанавливает mergedAt.
func (s *PullRequestService) Merge(ctx context.Context, prID string) (*domain.PullRequest, error) {
	pr, err := s.prs.GetPR(prID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errs.New(errs.CodeNotFound, "pull request not found")
		}
		return nil, err
	}

	if pr.Status == domain.PRStatusMerged {
		return pr, nil
	}

	now := time.Now().UTC()
	pr.Status = domain.PRStatusMerged // "MERGED"
	pr.MergedAt = &now

	if err := s.prs.UpdatePR(*pr); err != nil {
		return nil, err
	}

	return pr, nil
}

// Reassign выполняет переназначение одного ревьювера на другого
// и возвращает новый PR и ID подставленного ревьювера.
func (s *PullRequestService) Reassign(
	ctx context.Context,
	prID string,
	oldUserID string,
) (*domain.PullRequest, string, error) {
	pr, err := s.prs.GetPR(prID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, "", errs.New(errs.CodeNotFound, "pull request not found")
		}
		return nil, "", err
	}

	if pr.Status == domain.PRStatusMerged {
		return nil, "", errs.New(errs.CodePRMerged, "pull request already merged")
	}

	// проверяем, что oldUserID действительно назначен ревьювером
	if !slices.Contains(pr.AssignedReviewers, oldUserID) {
		return nil, "", errs.New(errs.CodeNotAssigned, "reviewer is not assigned to this PR")
	}

	// получаем пользователя и его команду
	reviewer, err := s.users.GetUser(oldUserID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, "", errs.New(errs.CodeNotFound, "reviewer not found")
		}
		return nil, "", err
	}

	team, err := s.teams.GetTeam(reviewer.TeamName)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, "", errs.New(errs.CodeNotFound, "team not found")
		}
		return nil, "", err
	}

	// кандидаты: активные из команды oldUserID, кроме автора и уже назначенных
	already := make(map[string]struct{}, len(pr.AssignedReviewers)+1)
	already[pr.AuthorID] = struct{}{}
	for _, rid := range pr.AssignedReviewers {
		already[rid] = struct{}{}
	}

	candidates := make([]string, 0, len(team.Members))
	for _, m := range team.Members {
		if !m.IsActive {
			continue
		}
		if _, ok := already[m.UserID]; ok {
			continue
		}
		candidates = append(candidates, m.UserID)
	}

	if len(candidates) == 0 {
		return nil, "", errs.New(errs.CodeNoCandidate, "no active replacement candidate in team")
	}

	// выбираем случайного кандидата
	rand.Shuffle(len(candidates), func(i, j int) {
		candidates[i], candidates[j] = candidates[j], candidates[i]
	})
	replacement := candidates[0]

	// заменяем oldUserID на replacement
	for i, rid := range pr.AssignedReviewers {
		if rid == oldUserID {
			pr.AssignedReviewers[i] = replacement
			break
		}
	}

	if err := s.prs.UpdatePR(*pr); err != nil {
		return nil, "", err
	}

	return pr, replacement, nil
}

// GetUserReviews возвращает список PR, назначенных на конкретного пользователя,
// в виде полного доменного объекта PR; хендлер уже маппит его в PullRequestShort.
func (s *PullRequestService) GetUserReviews(ctx context.Context, userID string) ([]domain.PullRequest, error) {

	if _, err := s.users.GetUser(userID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errs.New(errs.CodeNotFound, "user not found")
		}
		return nil, err
	}

	prs, err := s.prs.GetPRsByReviewer(userID)
	if err != nil {
		return nil, err
	}

	return prs, nil
}

// ReassignReviewer — тонкая обёртка над Reassign, которая
// возвращает только ошибку; используется в массовой деактивации.
func (s *PullRequestService) ReassignReviewer(ctx context.Context, prID, oldUserID string) error {
	_, _, err := s.Reassign(ctx, prID, oldUserID)
	return err
}

// GetStats прокидывает запрос статистики в репозиторий.
func (s *PullRequestService) GetStats(ctx context.Context) (repository.Stats, error) {
	return s.prs.GetStats(ctx)
}
