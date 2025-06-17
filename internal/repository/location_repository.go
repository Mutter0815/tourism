package repository

import (
	"fmt"
	"strings"

	"tourism/internal/model"

	"github.com/jmoiron/sqlx"
)

// LocationRepository обеспечивает доступ к данным локаций в базе данных.
type LocationRepository struct {
	db *sqlx.DB
}

// NewLocationRepository создает новый репозиторий для локаций.
func NewLocationRepository(db *sqlx.DB) *LocationRepository {
	return &LocationRepository{db: db}
}

// FindAll возвращает все локации (без фильтрации).
func (r *LocationRepository) FindAll() ([]model.Location, error) {
	locations := []model.Location{}
	err := r.db.Select(&locations, "SELECT * FROM locations")
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении списка локаций: %w", err)
	}
	return locations, nil
}

// FindByFilters выполняет поиск локаций по заданным фильтрам (категория, регион, минимальный рейтинг) и ключевому слову.
func (r *LocationRepository) FindByFilters(category string, region string, minRating float64, keyword string) ([]model.Location, error) {
	query := "SELECT * FROM locations WHERE 1=1"
	args := []interface{}{}
	if category != "" && strings.ToLower(category) != "any" {
		query += " AND LOWER(category)=LOWER(?)"
		args = append(args, category)
	}
	if region != "" && strings.ToLower(region) != "any" {
		query += " AND LOWER(region)=LOWER(?)"
		args = append(args, region)
	}
	if minRating > 0 {
		query += " AND rating >= ?"
		args = append(args, minRating)
	}
	if keyword != "" {
		kw := "%" + strings.ToLower(keyword) + "%"
		query += " AND (LOWER(name) LIKE ? OR LOWER(description) LIKE ?)"
		args = append(args, kw, kw)
	}
	query = sqlx.Rebind(sqlx.DOLLAR, query)
	locations := []model.Location{}
	if err := r.db.Select(&locations, query, args...); err != nil {
		return nil, fmt.Errorf("ошибка при поиске локаций: %w", err)
	}
	return locations, nil
}

// GetByID получает локацию по ее идентификатору.
func (r *LocationRepository) GetByID(id int) (*model.Location, error) {
	var location model.Location
	err := r.db.Get(&location, "SELECT * FROM locations WHERE id=$1", id)
	if err != nil {
		return nil, err
	}
	return &location, nil
}

// AddPhoto сохраняет новый идентификатор фото, связанного с локацией.
func (r *LocationRepository) AddPhoto(locationID int, fileID string) error {
	_, err := r.db.Exec("INSERT INTO location_photos (location_id, file_id) VALUES ($1, $2)", locationID, fileID)
	if err != nil {
		return fmt.Errorf("ошибка при сохранении фото локации: %w", err)
	}
	return nil
}

// GetPhotos возвращает все сохраненные фотографии для указанной локации.
func (r *LocationRepository) GetPhotos(locationID int) ([]model.LocationPhoto, error) {
	photos := []model.LocationPhoto{}
	err := r.db.Select(&photos, "SELECT * FROM location_photos WHERE location_id=$1", locationID)
	if err != nil {
		return nil, fmt.Errorf("ошибка при получении фотографий локации: %w", err)
	}
	return photos, nil
}
