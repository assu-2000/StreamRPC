package room

import (
	"time"
)

type Room struct {
	ID        string
	Name      string
	CreatedAt time.Time
	CreatedBy string
	IsPrivate bool
}

type RoomEvent struct {
	Type    EventType
	UserID  string
	RoomID  string
	Payload interface{}
}

type EventType int

const (
	EventUserJoined EventType = iota
	EventUserLeft
	EventRoomDeleted
	EventMessage
)

type ChatMessage struct {
	ID        string
	RoomID    string
	UserID    string
	Content   string
	Timestamp time.Time
}
