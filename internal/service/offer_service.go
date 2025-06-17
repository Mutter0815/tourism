package service

import "tourism/internal/repository"

// OfferService содержит логику подписки на рассылки интересных предложений.
type OfferService struct {
	subRepo *repository.SubscriptionRepository
}

// NewOfferService создает новый сервис предложений.
func NewOfferService(subRepo *repository.SubscriptionRepository) *OfferService {
	return &OfferService{subRepo: subRepo}
}

// Subscribe оформляет подписку пользователя на рассылку.
func (s *OfferService) Subscribe(userID int) error {
	return s.subRepo.Subscribe(userID)
}

// Unsubscribe отменяет подписку пользователя.
func (s *OfferService) Unsubscribe(userID int) error {
	return s.subRepo.Unsubscribe(userID)
}

// GetSubscriberIDs возвращает Telegram ID всех подписчиков.
func (s *OfferService) GetSubscriberIDs() ([]int64, error) {
	return s.subRepo.GetAllSubscriberTelegramIDs()
}
