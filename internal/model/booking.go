package model

// Booking представляет заявку на бронирование услуги (размещение, тур и т.д.) на основе локации.
type Booking struct {
	ID         int    `db:"id"`
	UserID     int    `db:"user_id"`     // пользователь (турист), создавший заявку
	LocationID int    `db:"location_id"` // локация (например, жилье или тур), которую бронируют
	Details    string `db:"details"`     // текстовые детали бронирования (даты, количество участников)
	Status     string `db:"status"`      // статус заявки: "pending", "confirmed", "rejected"
}
