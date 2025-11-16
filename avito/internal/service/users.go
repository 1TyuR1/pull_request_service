package service

import (
	"avito/internal/domain"
	"avito/internal/errs"
	"avito/internal/repository"
	"errors"
)

type UserService struct {
	users repository.UserRepository
}

func NewUserService(ur repository.UserRepository) *UserService {
	return &UserService{users: ur}
}

func (s *UserService) SetIsActive(userID string, active bool) (*domain.User, error) {
	u, err := s.users.SetUserActive(userID, active)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errs.New(errs.CodeNotFound, "user not found")
		}
		return nil, err
	}
	return u, nil
}

func (s *UserService) GetUser(userID string) (*domain.User, error) {
	u, err := s.users.GetUser(userID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, errs.New(errs.CodeNotFound, "user not found")
		}
		return nil, err
	}
	return u, nil
}
