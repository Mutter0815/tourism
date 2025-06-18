package service

import (
	"database/sql"
	"fmt"

	"tourism/internal/model"
	"tourism/internal/repository"
)

type AuthService struct {
	userRepo *repository.UserRepository
}

func NewAuthService(userRepo *repository.UserRepository) *AuthService {
	return &AuthService{userRepo: userRepo}
}

func (s *AuthService) AuthUser(telegramID int64, username, firstName, lastName string) (*model.User, error) {
	user, err := s.userRepo.GetByTelegramID(telegramID)
	if err != nil {
		if err == sql.ErrNoRows {

			newUser := &model.User{
				TelegramID: telegramID,
				Username:   username,
				FirstName:  firstName,
				LastName:   lastName,
				Role:       "user",
			}
			id, err := s.userRepo.Create(newUser)
			if err != nil {
				return nil, err
			}
			newUser.ID = id
			return newUser, nil
		}
		// Другая ошибка выполнения запроса
		return nil, fmt.Errorf("ошибка при поиске пользователя: %w", err)
	}
	// Пользователь найден, возвращаем его
	return user, nil
}
