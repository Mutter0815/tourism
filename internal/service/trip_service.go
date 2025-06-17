package service

import (
	"math"
	"tourism/internal/model"
	"tourism/internal/repository"
)

// TripService содержит бизнес-логику, связанную с планированием поездок (маршрутов).
type TripService struct {
	tripRepo     *repository.TripRepository
	locationRepo *repository.LocationRepository
}

// NewTripService создает новый сервис для работы с маршрутами.
func NewTripService(tripRepo *repository.TripRepository, locationRepo *repository.LocationRepository) *TripService {
	return &TripService{tripRepo: tripRepo, locationRepo: locationRepo}
}

// CreateTrip создает новый маршрут (поездку) для пользователя.
func (s *TripService) CreateTrip(userID int, name string) (int, error) {
	return s.tripRepo.Create(userID, name)
}

// AddLocationToTrip добавляет локацию в маршрут.
func (s *TripService) AddLocationToTrip(tripID int, locationID int) error {
	return s.tripRepo.AddLocation(tripID, locationID)
}

// OptimizeTrip оптимизирует порядок точек в маршруте по географической близости (наивный алгоритм).
// Возвращает упорядоченный список локаций.
func (s *TripService) OptimizeTrip(tripID int) ([]model.Location, error) {
	locations, err := s.tripRepo.GetLocations(tripID)
	if err != nil {
		return nil, err
	}
	if len(locations) < 2 {
		return locations, nil // нечего оптимизировать
	}
	optimized := []model.Location{}
	used := make([]bool, len(locations))
	optimized = append(optimized, locations[0])
	used[0] = true
	for i := 1; i < len(locations); i++ {
		last := optimized[len(optimized)-1]
		minDist := math.MaxFloat64
		minIndex := -1
		for j, loc := range locations {
			if !used[j] {
				dx := last.Latitude - loc.Latitude
				dy := last.Longitude - loc.Longitude
				dist := dx*dx + dy*dy
				if dist < minDist {
					minDist = dist
					minIndex = j
				}
			}
		}
		if minIndex >= 0 {
			used[minIndex] = true
			optimized = append(optimized, locations[minIndex])
		}
	}
	var locIDs []int
	for _, loc := range optimized {
		locIDs = append(locIDs, loc.ID)
	}
	s.tripRepo.UpdateOrder(tripID, locIDs)
	return optimized, nil
}

// GetTripLocations возвращает локации маршрута в текущем порядке.
func (s *TripService) GetTripLocations(tripID int) ([]model.Location, error) {
	return s.tripRepo.GetLocations(tripID)
}
