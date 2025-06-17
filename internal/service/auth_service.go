package service

import (
	"database/sql"
	"fmt"

	"tourism/internal/model"
	"tourism/internal/repository"
)

// AuthService отвечает за регистрацию/авторизацию пользователей (по Telegram ID).
type AuthService struct {
	userRepo *repository.UserRepository
}

// NewAuthService создает новый сервис аутентификации.
func NewAuthService(userRepo *repository.UserRepository) *AuthService {
	return &AuthService{userRepo: userRepo}
}

// AuthUser проверяет наличие пользователя с данным TelegramID и регистрирует нового, если не найден.
// Возвращает структуру пользователя (существующего или новосозданного).
func (s *AuthService) AuthUser(telegramID int64, username, firstName, lastName string) (*model.User, error) {
	user, err := s.userRepo.GetByTelegramID(telegramID)
	if err != nil {
		if err == sql.ErrNoRows {
			// Пользователь не зарегистрирован - создаем новую запись
			newUser := &model.User{
				TelegramID: telegramID,
				Username:   username,
				FirstName:  firstName,
				LastName:   lastName,
				Role:       "user", // по умолчанию все новые пользователи - туристы
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
