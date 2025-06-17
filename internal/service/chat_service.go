package service

import (
	"fmt"
	"sync"
	"tourism/internal/repository"
)

// ChatService управляет состоянием чат-сессий между туристами и провайдерами.
type ChatService struct {
	bookingRepo  *repository.BookingRepository
	userRepo     *repository.UserRepository
	locationRepo *repository.LocationRepository
	chatBooking  map[int64]int   // соответствие TelegramID пользователя -> BookingID чата
	activeChats  map[int64]int64 // соответствие TelegramID пользователя -> TelegramID собеседника
	mu           sync.Mutex
}

// NewChatService создает новый сервис чата.
func NewChatService(bookingRepo *repository.BookingRepository, userRepo *repository.UserRepository, locationRepo *repository.LocationRepository) *ChatService {
	return &ChatService{
		bookingRepo:  bookingRepo,
		userRepo:     userRepo,
		locationRepo: locationRepo,
		chatBooking:  make(map[int64]int),
		activeChats:  make(map[int64]int64),
	}
}

// StartChat инициирует чат между пользователем с telegramID и вторым участником по указанному ID бронирования.
func (s *ChatService) StartChat(telegramID int64, bookingID int) (int64, error) {
	booking, err := s.bookingRepo.GetByID(bookingID)
	if err != nil {
		return 0, fmt.Errorf("бронирование не найдено")
	}
	var partnerTelegramID int64
	bookingUser, err := s.userRepo.GetByID(booking.UserID)
	if err != nil {
		return 0, fmt.Errorf("не найден пользователь заявки")
	}
	partnerUserID := 0
	if bookingUser.TelegramID == telegramID {
		// текущий пользователь - турист, второй участник - провайдер
		location, err := s.locationRepo.GetByID(booking.LocationID)
		if err != nil || location.ProviderID == nil {
			return 0, fmt.Errorf("не удалось определить провайдера для чата")
		}
		partnerUserID = *location.ProviderID
	} else {
		// текущий пользователь - провайдер, второй участник - турист
		partnerUserID = booking.UserID
	}
	partnerUser, err := s.userRepo.GetByID(partnerUserID)
	if err != nil {
		return 0, fmt.Errorf("не найден второй участник чата")
	}
	partnerTelegramID = partnerUser.TelegramID

	// Сохраняем состояние активного чата в памяти
	s.mu.Lock()
	s.activeChats[telegramID] = partnerTelegramID
	s.activeChats[partnerTelegramID] = telegramID
	s.chatBooking[telegramID] = bookingID
	s.chatBooking[partnerTelegramID] = bookingID
	s.mu.Unlock()
	return partnerTelegramID, nil
}

// EndChat завершает активный чат для указанного пользователя (и второго участника).
func (s *ChatService) EndChat(telegramID int64) {
	s.mu.Lock()
	partnerID, ok := s.activeChats[telegramID]
	if ok {
		// удаляем оба соответствия
		delete(s.activeChats, telegramID)
		delete(s.activeChats, partnerID)
		delete(s.chatBooking, telegramID)
		delete(s.chatBooking, partnerID)
	}
	s.mu.Unlock()
}

// GetChatPartner возвращает TelegramID собеседника, если пользователь находится в чате, иначе 0.
func (s *ChatService) GetChatPartner(telegramID int64) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	if partner, ok := s.activeChats[telegramID]; ok {
		return partner
	}
	return 0
}

// GetChatBookingID возвращает идентификатор бронирования чата, в котором участвует пользователь.
func (s *ChatService) GetChatBookingID(telegramID int64) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	if bookingID, ok := s.chatBooking[telegramID]; ok {
		return bookingID
	}
	return 0
}
