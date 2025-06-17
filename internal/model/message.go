package model

// Message представляет сообщение чата между пользователем и провайдером или пользователя с поддержкой.
type Message struct {
	ID         int    `db:"id"`
	FromUserID int    `db:"from_user_id"`
	ToUserID   int    `db:"to_user_id"`
	BookingID  *int   `db:"booking_id"` // если сообщение относится к чату по конкретному бронированию (турист-провайдер), иначе NULL
	Content    string `db:"content"`
	IsSupport  bool   `db:"is_support"` // признак сообщения в чате поддержки
}
