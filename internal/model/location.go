package model

// Location представляет туристическую локацию или объект для посещения/бронирования.
type Location struct {
	ID          int     `db:"id"`
	Name        string  `db:"name"`
	Description string  `db:"description"`
	Category    string  `db:"category"` // категория (тип) локации, например: природная, историческая, жилье и т.п.
	Region      string  `db:"region"`   // регион или район, где находится локация
	Rating      float64 `db:"rating"`   // средний рейтинг (например, от 0 до 5)
	Latitude    float64 `db:"latitude"`
	Longitude   float64 `db:"longitude"`
	ProviderID  *int    `db:"provider_id"` // (опционально) id пользователя-провайдера (если это объект, предоставляемый провайдером)
}
