package model

// LocationPhoto представляет фото, связанное с определенной локацией.
type LocationPhoto struct {
	ID         int    `db:"id"`
	LocationID int    `db:"location_id"`
	FileID     string `db:"file_id"` // FileID фотографии в Telegram (для повторной отправки без загрузки)
}
