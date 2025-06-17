package service

import (
	"tourism/internal/model"
	"tourism/internal/repository"
)

// UserService содержит бизнес-логику, связанную с пользователями.
type UserService struct {
	userRepo *repository.UserRepository
}

// NewUserService создает новый сервис пользователей.
func NewUserService(userRepo *repository.UserRepository) *UserService {
	return &UserService{userRepo: userRepo}
}

// GetByID возвращает пользователя по ID (обертка над репозиторием).
func (s *UserService) GetByID(id int) (*model.User, error) {
	return s.userRepo.GetByID(id)
}
