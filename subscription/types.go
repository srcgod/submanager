package subscription

type WSclient interface {
	ID() string
	UserID() int64
	SendMessage(data []byte)
}
