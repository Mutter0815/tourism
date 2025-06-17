package handler

import (
	"net/http"

	"tourism/internal/service"

	"github.com/gin-gonic/gin"
)

// Handler структурирует зависимости сервисов для обработки HTTP-запросов.
type Handler struct {
	UserService     *service.UserService
	LocationService *service.LocationService
	TripService     *service.TripService
	BookingService  *service.BookingService
	ChatService     *service.ChatService
	OfferService    *service.OfferService
}

// NewHandler создает новый Handler с внедрением зависимостей (сервисов).
func NewHandler(us *service.UserService, ls *service.LocationService, ts *service.TripService,
	bs *service.BookingService, cs *service.ChatService, os *service.OfferService) *Handler {
	return &Handler{
		UserService:     us,
		LocationService: ls,
		TripService:     ts,
		BookingService:  bs,
		ChatService:     cs,
		OfferService:    os,
	}
}

// ListLocations обработчик для GET /api/locations - возвращает список всех локаций.
func (h *Handler) ListLocations(c *gin.Context) {
	locations, err := h.LocationService.SearchLocations("", "", 0, "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Не удалось получить локации"})
		return
	}
	c.JSON(http.StatusOK, locations)
}

// ListUsers обработчик для GET /api/users - возвращает список всех пользователей.
func (h *Handler) ListUsers(c *gin.Context) {
	// Для простоты: возвращаем заглушку, реальный список пользователей не выводится
	c.JSON(http.StatusOK, gin.H{"message": "Список пользователей недоступен в данной версии API"})
}
