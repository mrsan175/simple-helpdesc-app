package service

import (
	"context"

	"backend/internal/domain"
)

type UserService struct {
	userRepo domain.UserRepository
}

func NewUserService(userRepo domain.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

func (s *UserService) List(ctx context.Context, role string) ([]*domain.User, error) {
	return s.userRepo.List(ctx, role)
}

func (s *UserService) UpdateRole(ctx context.Context, actor *domain.User, id string, newRole domain.UserRole) error {
	if actor.Role != domain.RoleAdmin {
		return domain.ErrForbidden
	}

	targetUser, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	if targetUser.ID == actor.ID && newRole != domain.RoleAdmin {
		adminCount, err := s.userRepo.CountByRole(ctx, domain.RoleAdmin)
		if err != nil {
			return err
		}
		if adminCount <= 1 {
			return domain.ErrForbidden
		}
	}

	return s.userRepo.UpdateRole(ctx, id, newRole)
}
