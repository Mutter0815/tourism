package model

// Trip представляет планируемую поездку (маршрут), составленный пользователем.
type Trip struct {
	ID     int    `db:"id"`
	UserID int    `db:"user_id"`
	Name   string `db:"name"`
	Status string `db:"status"` // статус поездки, например: "draft", "completed"
}

// TripLocation представляет связь между поездкой и локацией, входящей в маршрут.
type TripLocation struct {
	ID         int `db:"id"`
	TripID     int `db:"trip_id"`
	LocationID int `db:"location_id"`
	Order      int `db:"order_index"` // порядок следования локации в маршруте
}
