package repository

import (
	"fmt"

	"tourism/internal/model"

	"github.com/jmoiron/sqlx"
)

// BookingRepository обеспечивает доступ к данным бронирований в базе данных.
type BookingRepository struct {
	db *sqlx.DB
}

// NewBookingRepository создает новый репозиторий для бронирований.
func NewBookingRepository(db *sqlx.DB) *BookingRepository {
	return &BookingRepository{db: db}
}

// Create создает новую заявку на бронирование.
func (r *BookingRepository) Create(booking *model.Booking) (int, error) {
	query := `INSERT INTO bookings (user_id, location_id, details, status) VALUES ($1, $2, $3, $4) RETURNING id`
	var id int
	err := r.db.QueryRow(query, booking.UserID, booking.LocationID, booking.Details, booking.Status).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("не удалось создать бронирование: %w", err)
	}
	return id, nil
}

// GetByID возвращает бронирование по ID.
func (r *BookingRepository) GetByID(id int) (*model.Booking, error) {
	var booking model.Booking
	err := r.db.Get(&booking, "SELECT * FROM bookings WHERE id=$1", id)
	if err != nil {
		return nil, err
	}
	return &booking, nil
}

// UpdateStatus обновляет статус бронирования.
func (r *BookingRepository) UpdateStatus(id int, status string) error {
	_, err := r.db.Exec("UPDATE bookings SET status=$1 WHERE id=$2", status, id)
	if err != nil {
		return fmt.Errorf("не удалось обновить статус бронирования: %w", err)
	}
	return nil
}
