package repository

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

// SubscriptionRepository обеспечивает доступ к данным подписчиков на рассылки.
type SubscriptionRepository struct {
	db *sqlx.DB
}

// NewSubscriptionRepository создает новый репозиторий подписок.
func NewSubscriptionRepository(db *sqlx.DB) *SubscriptionRepository {
	return &SubscriptionRepository{db: db}
}

// Subscribe добавляет пользователя в список подписчиков (если еще не подписан).
func (r *SubscriptionRepository) Subscribe(userID int) error {
	_, err := r.db.Exec("INSERT INTO offer_subscriptions (user_id) VALUES ($1) ON CONFLICT DO NOTHING", userID)
	if err != nil {
		return fmt.Errorf("не удалось оформить подписку: %w", err)
	}
	return nil
}

// Unsubscribe удаляет пользователя из подписчиков.
func (r *SubscriptionRepository) Unsubscribe(userID int) error {
	_, err := r.db.Exec("DELETE FROM offer_subscriptions WHERE user_id=$1", userID)
	if err != nil {
		return fmt.Errorf("не удалось отменить подписку: %w", err)
	}
	return nil
}

// GetAllSubscriberTelegramIDs возвращает Telegram ID всех пользователей, подписанных на предложения.
func (r *SubscriptionRepository) GetAllSubscriberTelegramIDs() ([]int64, error) {
	ids := []int64{}
	err := r.db.Select(&ids,
		`SELECT u.telegram_id FROM offer_subscriptions s 
		 JOIN users u ON s.user_id = u.id`)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении списка подписчиков: %w", err)
	}
	return ids, nil
}
