package room

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	roomsKey             = "rooms"
	roomMembersKeyFormat = "room:%s:members"
	roomKey              = "room"
	roomKeyFormat        = "%s:%s"
)

type RedisRepository struct {
	client *redis.Client
}

func NewRedisRepository(client *redis.Client) *RedisRepository {
	return &RedisRepository{client: client}
}

func (r *RedisRepository) CreateRoom(ctx context.Context, room *Room) error {
	roomKey := fmt.Sprintf(roomKeyFormat, roomKey, room.ID)
	pipe := r.client.Pipeline()

	pipe.HSet(ctx, roomKey,
		"name", room.Name,
		"created_at", room.CreatedAt.Format(time.RFC3339),
		"created_by", room.CreatedBy,
		"is_private", room.IsPrivate,
	)

	pipe.SAdd(ctx, roomsKey, room.ID)
	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisRepository) GetRoom(ctx context.Context, roomID string) (*Room, error) {
	roomKey := fmt.Sprintf(roomKeyFormat, roomKey, roomID)
	result, err := r.client.HGetAll(ctx, roomKey).Result()
	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		ErrRoomNotFound := errors.New("room not found")
		return nil, ErrRoomNotFound
	}

	createdAt, _ := time.Parse(time.RFC3339, result["created_at"])
	isPrivate, _ := strconv.ParseBool(result["is_private"])

	return &Room{
		ID:        roomID,
		Name:      result["name"],
		CreatedAt: createdAt,
		CreatedBy: result["created_by"],
		IsPrivate: isPrivate,
	}, nil
}

func (r *RedisRepository) RoomExists(ctx context.Context, roomID string) (bool, error) {
	roomKey := fmt.Sprintf(roomKeyFormat, roomKey, roomID)
	exists, err := r.client.Exists(ctx, roomKey).Result()
	return exists > 0, err
}

func (r *RedisRepository) AddRoomMember(ctx context.Context, roomID, userID string) error {
	memberKey := fmt.Sprintf(roomMembersKeyFormat, roomID)
	return r.client.SAdd(ctx, memberKey, userID).Err()
}

func (r *RedisRepository) RemoveRoomMember(ctx context.Context, roomID, userID string) error {
	memberKey := fmt.Sprintf(roomMembersKeyFormat, roomID)
	return r.client.SRem(ctx, memberKey, userID).Err()
}

func (r *RedisRepository) GetRoomMembers(ctx context.Context, roomID string) ([]string, error) {
	memberKey := fmt.Sprintf(roomMembersKeyFormat, roomID)
	return r.client.SMembers(ctx, memberKey).Result()
}

func (r *RedisRepository) IsRoomMember(ctx context.Context, roomID, userID string) (bool, error) {
	memberKey := fmt.Sprintf(roomMembersKeyFormat, roomID)
	return r.client.SIsMember(ctx, memberKey, userID).Result()
}

func (r *RedisRepository) SubscribeToRoom(ctx context.Context, roomID string) *redis.PubSub {
	channel := fmt.Sprintf(roomKeyFormat, roomKey, roomID)
	return r.client.Subscribe(ctx, channel)
}

func (r *RedisRepository) PublishRoomEvent(ctx context.Context, roomID string, event interface{}) error {
	channel := fmt.Sprintf(roomKeyFormat, roomKey, roomID)
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}
	return r.client.Publish(ctx, channel, payload).Err()
}

func (r *RedisRepository) ListRoomIDs(ctx context.Context) ([]string, error) {
	return r.client.SMembers(ctx, roomsKey).Result()
}

func (r *RedisRepository) DeleteRoom(ctx context.Context, roomID string) error {
	pipe := r.client.Pipeline()

	// Deletes room's metadata
	pipe.Del(ctx, fmt.Sprintf(roomKeyFormat, roomKey, roomID))

	// Deletes the list of members
	pipe.Del(ctx, fmt.Sprintf(roomMembersKeyFormat, roomID))

	// removes from the global list
	pipe.SRem(ctx, "rooms", roomID)

	_, err := pipe.Exec(ctx)
	return err
}

func (r *RedisRepository) RemoveAllMembers(ctx context.Context, roomID string) error {
	key := fmt.Sprintf(roomMembersKeyFormat, roomID)
	return r.client.Del(ctx, key).Err()
}
