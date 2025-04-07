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
	EventRoomUpdated
)

type ChatMessage struct {
	ID        string
	RoomID    string
	UserID    string
	Content   string
	Timestamp time.Time
}

type RoomStats struct {
	Room          *Room
	TotalMembers  int
	ActiveMembers int
}
