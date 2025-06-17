package repository

import (
	"fmt"

	"tourism/internal/model"

	"github.com/jmoiron/sqlx"
)

// UserRepository обеспечивает доступ к данным пользователей в базе данных.
type UserRepository struct {
	db *sqlx.DB
}

// NewUserRepository создаёт новый репозиторий пользователей.
func NewUserRepository(db *sqlx.DB) *UserRepository {
	return &UserRepository{db: db}
}

// Create добавляет нового пользователя в базу. Возвращает ID созданного пользователя.
func (r *UserRepository) Create(user *model.User) (int, error) {
	query := `INSERT INTO users (telegram_id, username, first_name, last_name, role)
	          VALUES ($1, $2, $3, $4, $5) RETURNING id`
	var id int
	err := r.db.QueryRow(query, user.TelegramID, user.Username, user.FirstName, user.LastName, user.Role).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("не удалось создать пользователя: %w", err)
	}
	return id, nil
}

// GetByTelegramID ищет пользователя по его Telegram ID. Возвращает nil, если не найдено.
func (r *UserRepository) GetByTelegramID(telegramID int64) (*model.User, error) {
	var user model.User
	err := r.db.Get(&user, "SELECT * FROM users WHERE telegram_id=$1", telegramID)
	if err != nil {
		// sqlx.Get возвращает ошибку, если не найдено (sql.ErrNoRows и др.)
		return nil, err
	}
	return &user, nil
}

// GetByID возвращает пользователя по внутреннему идентификатору.
func (r *UserRepository) GetByID(id int) (*model.User, error) {
	var user model.User
	err := r.db.Get(&user, "SELECT * FROM users WHERE id=$1", id)
	if err != nil {
		return nil, err
	}
	return &user, nil
}
