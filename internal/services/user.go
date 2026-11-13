package services

import (
	"context"
	"time"

	"top1/internal/models"
	"top1/internal/repo"
)

type UserService struct {
	userRepo *repo.AuthRepository
}

func NewUserService(userRepo *repo.AuthRepository) *UserService {
	return &UserService{
		userRepo: userRepo,
	}
}

func (s *UserService) GetUser(
	ctx context.Context,
	userID string,
) (*models.User, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	return s.userRepo.GetUserByID(ctx, userID)
}
