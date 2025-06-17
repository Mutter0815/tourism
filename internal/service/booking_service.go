package service

import (
	"tourism/internal/model"
	"tourism/internal/repository"
)

// BookingService содержит бизнес-логику, связанную с бронированиями.
type BookingService struct {
	bookingRepo *repository.BookingRepository
}

// NewBookingService создает новый сервис бронирований.
func NewBookingService(bookingRepo *repository.BookingRepository) *BookingService {
	return &BookingService{bookingRepo: bookingRepo}
}

// CreateBooking создает новую заявку на бронирование для пользователя.
func (s *BookingService) CreateBooking(userID int, locationID int, details string) (int, error) {
	booking := &model.Booking{
		UserID:     userID,
		LocationID: locationID,
		Details:    details,
		Status:     "pending",
	}
	return s.bookingRepo.Create(booking)
}

// ConfirmBooking устанавливает статус бронирования "confirmed".
func (s *BookingService) ConfirmBooking(bookingID int) error {
	return s.bookingRepo.UpdateStatus(bookingID, "confirmed")
}

// RejectBooking устанавливает статус бронирования "rejected".
func (s *BookingService) RejectBooking(bookingID int) error {
	return s.bookingRepo.UpdateStatus(bookingID, "rejected")
}

// GetBooking возвращает бронирование по ID.
func (s *BookingService) GetBooking(bookingID int) (*model.Booking, error) {
	return s.bookingRepo.GetByID(bookingID)
}
