package repository

import (
	"fmt"

	"tourism/internal/model"

	"github.com/jmoiron/sqlx"
)

// MessageRepository обеспечивает сохранение и получение сообщений чата из базы данных.
type MessageRepository struct {
	db *sqlx.DB
}

// NewMessageRepository создает новый репозиторий сообщений.
func NewMessageRepository(db *sqlx.DB) *MessageRepository {
	return &MessageRepository{db: db}
}

// Save сохраняет новое сообщение чата.
func (r *MessageRepository) Save(msg *model.Message) error {
	_, err := r.db.Exec(`INSERT INTO messages (from_user_id, to_user_id, booking_id, content, is_support)
	                      VALUES ($1, $2, $3, $4, $5)`,
		msg.FromUserID, msg.ToUserID, msg.BookingID, msg.Content, msg.IsSupport)
	if err != nil {
		return fmt.Errorf("ошибка при сохранении сообщения: %w", err)
	}
	return nil
}

// ListByBooking получает все сообщения для заданного бронирования (чат турист-провайдер).
func (r *MessageRepository) ListByBooking(bookingID int) ([]model.Message, error) {
	messages := []model.Message{}
	err := r.db.Select(&messages, "SELECT * FROM messages WHERE booking_id=$1 ORDER BY id", bookingID)
	if err != nil {
		return nil, err
	}
	return messages, nil
}

// ListSupportMessages получает все сообщения чата поддержки (по пользователю).
func (r *MessageRepository) ListSupportMessages(userID int) ([]model.Message, error) {
	messages := []model.Message{}
	err := r.db.Select(&messages, "SELECT * FROM messages WHERE is_support=true AND (from_user_id=$1 OR to_user_id=$1) ORDER BY id", userID)
	if err != nil {
		return nil, err
	}
	return messages, nil
}
