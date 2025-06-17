package service

import (
	"tourism/internal/model"
	"tourism/internal/repository"
)

// LocationService содержит бизнес-логику, связанную с локациями.
type LocationService struct {
	locationRepo *repository.LocationRepository
}

// NewLocationService создает новый сервис локаций.
func NewLocationService(locationRepo *repository.LocationRepository) *LocationService {
	return &LocationService{locationRepo: locationRepo}
}

// SearchLocations выполняет поиск локаций по заданным параметрам фильтрации и/или ключевому слову.
func (s *LocationService) SearchLocations(category string, region string, minRating float64, keyword string) ([]model.Location, error) {
	return s.locationRepo.FindByFilters(category, region, minRating, keyword)
}

// GetLocationDetails получает подробные данные о локации (сам объект и список фото).
func (s *LocationService) GetLocationDetails(locationID int) (*model.Location, []model.LocationPhoto, error) {
	location, err := s.locationRepo.GetByID(locationID)
	if err != nil {
		return nil, nil, err
	}
	photos, err := s.locationRepo.GetPhotos(locationID)
	if err != nil {
		return location, nil, err // возвращаем локацию даже если фото не загрузились
	}
	return location, photos, nil
}

// AddPhoto добавляет фото (FileID) к указанной локации.
func (s *LocationService) AddPhoto(locationID int, fileID string) error {
	return s.locationRepo.AddPhoto(locationID, fileID)
}
