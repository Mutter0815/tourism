package repository

import (
	"fmt"

	"tourism/internal/model"

	"github.com/jmoiron/sqlx"
)

// TripRepository обеспечивает доступ к данным маршрутов (поездок) в базе данных.
type TripRepository struct {
	db *sqlx.DB
}

// NewTripRepository создает новый репозиторий для поездок.
func NewTripRepository(db *sqlx.DB) *TripRepository {
	return &TripRepository{db: db}
}

// Create создает новую поездку (маршрут) для указанного пользователя.
func (r *TripRepository) Create(userID int, name string) (int, error) {
	query := `INSERT INTO trips (user_id, name, status) VALUES ($1, $2, $3) RETURNING id`
	var id int
	err := r.db.QueryRow(query, userID, name, "draft").Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("не удалось создать поездку: %w", err)
	}
	return id, nil
}

// AddLocation добавляет локацию в маршрут (в конец списка).
func (r *TripRepository) AddLocation(tripID int, locationID int) error {
	var order int
	r.db.Get(&order, "SELECT COALESCE(MAX(order_index), 0) + 1 FROM trip_locations WHERE trip_id=$1", tripID)
	_, err := r.db.Exec("INSERT INTO trip_locations (trip_id, location_id, order_index) VALUES ($1, $2, $3)", tripID, locationID, order)
	if err != nil {
		return fmt.Errorf("ошибка при добавлении локации в маршрут: %w", err)
	}
	return nil
}

// UpdateOrder обновляет порядок следования локаций в маршруте.
func (r *TripRepository) UpdateOrder(tripID int, locationOrder []int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	for idx, locID := range locationOrder {
		_, err := tx.Exec("UPDATE trip_locations SET order_index=$1 WHERE trip_id=$2 AND location_id=$3", idx+1, tripID, locID)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("не удалось обновить порядок маршрута: %w", err)
		}
	}
	return tx.Commit()
}

// GetLocations возвращает список локаций (с полями Location) для заданного маршрута в текущем порядке.
func (r *TripRepository) GetLocations(tripID int) ([]model.Location, error) {
	locations := []model.Location{}
	err := r.db.Select(&locations,
		`SELECT l.* FROM trip_locations tl 
		 JOIN locations l ON tl.location_id = l.id 
		 WHERE tl.trip_id=$1 
		 ORDER BY tl.order_index`, tripID)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении локаций маршрута: %w", err)
	}
	return locations, nil
}
