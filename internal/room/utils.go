package room

import (
	"context"
	"github.com/redis/go-redis/v9"
)

type Repository interface {
	// Room Management
	CreateRoom(ctx context.Context, room *Room) error
	GetRoom(ctx context.Context, roomID string) (*Room, error)
	// DeleteRoom(ctx context.Context, roomID string) error
	RoomExists(ctx context.Context, roomID string) (bool, error)
	// ListRooms(ctx context.Context) ([]*Room, error)

	// Membership
	AddRoomMember(ctx context.Context, roomID, userID string) error
	RemoveRoomMember(ctx context.Context, roomID, userID string) error
	GetRoomMembers(ctx context.Context, roomID string) ([]string, error)
	IsRoomMember(ctx context.Context, roomID, userID string) (bool, error)

	// PubSub
	SubscribeToRoom(ctx context.Context, roomID string) *redis.PubSub
	PublishRoomEvent(ctx context.Context, roomID string, event interface{}) error

	// Cleanup
	//RemoveAllMembers(ctx context.Context, roomID string) error
}
