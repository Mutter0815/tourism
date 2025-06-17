package model

// OfferSubscription представляет подписку пользователя на рассылку предложений.
type OfferSubscription struct {
	ID     int `db:"id"`
	UserID int `db:"user_id"`
}
