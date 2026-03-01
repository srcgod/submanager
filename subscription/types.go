package subscription

import (
	"fmt"
	"time"
)

const (
	SubjUserConnected    = "user.%d.connected"
	SubjUserDisconnected = "user.%d.disconnected"
)
const (
	SubjMessageCreated = "chat.message.created.%s"
)

type WSclient interface {
	ID() string
	UserID() int64
	SendMessage(data []byte)
}

func UserConnectedSubj(userID int64) string {
	return fmt.Sprintf(SubjUserConnected, userID)
}

func UserDisconnectedSubj(userID int64) string {
	return fmt.Sprintf(SubjUserDisconnected, userID)
}

type UserStatusMsg interface {
	GetUserID() int64
	GetTimestamp() time.Time
}

func MessageCreatedSubj(chatID string) string {
	return fmt.Sprintf(SubjMessageCreated, chatID)
}
