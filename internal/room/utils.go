package room

import (
	"context"
	"github.com/redis/go-redis/v9"
)

type RoomRepository interface {
	// Room Management
	CreateRoom(ctx context.Context, room *Room) error
	GetRoom(ctx context.Context, roomID string) (*Room, error)
	DeleteRoom(ctx context.Context, roomID string) error
	RoomExists(ctx context.Context, roomID string) (bool, error)
	ListRoomIDs(ctx context.Context) ([]string, error)

	// Membership Management
	AddRoomMember(ctx context.Context, roomID, userID string) error
	RemoveRoomMember(ctx context.Context, roomID, userID string) error
	GetRoomMembers(ctx context.Context, roomID string) ([]string, error)
	IsRoomMember(ctx context.Context, roomID, userID string) (bool, error)
	RemoveAllMembers(ctx context.Context, roomID string) error

	// PubSub
	SubscribeToRoom(ctx context.Context, roomID string) *redis.PubSub
	PublishRoomEvent(ctx context.Context, roomID string, event interface{}) error

	// Cleanup
	//RemoveAllMembers(ctx context.Context, roomID string) error
}
