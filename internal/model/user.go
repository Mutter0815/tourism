package model

type User struct {
	ID         int    `db:"id"`
	TelegramID int64  `db:"telegram_id"`
	Username   string `db:"username"`
	FirstName  string `db:"first_name"`
	LastName   string `db:"last_name"`
	Role       string `db:"role"`
}
