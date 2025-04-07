package room

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

type RoomService struct {
	repo          RoomRepository
	activeRooms   map[string]*RoomContext
	activeRoomsMu sync.RWMutex
}

type RoomContext struct {
	Members    map[string]struct{}
	CancelFunc context.CancelFunc
}

func NewRoomService(repo RoomRepository) *RoomService {
	return &RoomService{
		repo:        repo,
		activeRooms: make(map[string]*RoomContext),
	}
}

func (s *RoomService) CreateRoom(ctx context.Context, name string, creatorID string, isPrivate bool) (*Room, error) {
	room := &Room{
		ID:        uuid.New().String(),
		Name:      name,
		CreatedAt: time.Now(),
		CreatedBy: creatorID,
		IsPrivate: isPrivate,
	}

	if err := s.repo.CreateRoom(ctx, room); err != nil {
		return nil, err
	}

	return room, nil
}

func (s *RoomService) JoinRoom(ctx context.Context, roomID, userID string) (<-chan RoomEvent, error) {
	// checks if room does exist
	exists, err := s.repo.RoomExists(ctx, roomID)
	if err != nil || !exists {
		return nil, errors.New("room does not exist")
	}

	// Adds the user into the room
	if err := s.repo.AddRoomMember(ctx, roomID, userID); err != nil {
		return nil, err
	}

	// creates channel for events
	events := make(chan RoomEvent, 10)

	// handles Redis PubSub connection
	roomCtx, cancel := context.WithCancel(ctx)
	pubsub := s.repo.SubscribeToRoom(roomCtx, roomID)

	// Stocke le room actif
	s.activeRoomsMu.Lock()
	s.activeRooms[roomID] = &RoomContext{
		Members:    make(map[string]struct{}),
		CancelFunc: cancel,
	}
	s.activeRooms[roomID].Members[userID] = struct{}{}
	s.activeRoomsMu.Unlock()

	// Goroutine to handle messages
	go s.handlePubSubMessages(roomID, pubsub, events)
	// notifies other users
	s.broadcastRoomEvent(roomID, RoomEvent{
		Type:   EventUserJoined,
		UserID: userID,
		RoomID: roomID,
	})

	return events, nil
}

func (s *RoomService) LeaveRoom(ctx context.Context, roomID, userID string) error {
	s.activeRoomsMu.Lock()
	defer s.activeRoomsMu.Unlock()

	if err := s.repo.RemoveRoomMember(ctx, roomID, userID); err != nil {
		return err
	}

	if roomCtx, exists := s.activeRooms[roomID]; exists {
		delete(roomCtx.Members, userID)
		if len(roomCtx.Members) == 0 {
			roomCtx.CancelFunc()
			delete(s.activeRooms, roomID)
		}
	}

	// notifies other users
	s.broadcastRoomEvent(roomID, RoomEvent{
		Type:   EventUserLeft,
		UserID: userID,
		RoomID: roomID,
	})

	return nil
}

func (s *RoomService) broadcastRoomEvent(roomID string, event RoomEvent) {
	err := s.repo.PublishRoomEvent(context.Background(), roomID, event)
	if err != nil {
		fmt.Println("publish room event error:", err)
		return
	}
}

// GetRoom retrieves details on a specific room
func (s *RoomService) GetRoom(ctx context.Context, roomID string) (*Room, error) {
	return s.repo.GetRoom(ctx, roomID)
}

// ListRooms returns all available rooms
func (s *RoomService) ListRooms(ctx context.Context) ([]*Room, error) {
	roomIDs, err := s.repo.ListRoomIDs(ctx)
	if err != nil {
		return nil, err
	}

	var rooms []*Room
	for _, id := range roomIDs {
		room, err := s.repo.GetRoom(ctx, id)
		if err != nil {
			continue // ou retourner l'erreur selon le cas
		}
		rooms = append(rooms, room)
	}

	return rooms, nil
}

// DeleteRoom deletes room and its members
func (s *RoomService) DeleteRoom(ctx context.Context, roomID string) error {
	// 1. notifies other users
	s.broadcastRoomEvent(roomID, RoomEvent{
		Type:   EventRoomDeleted,
		RoomID: roomID,
	})

	// 2. delets room
	return s.repo.DeleteRoom(ctx, roomID)
}

// GetRoomMembers returns members of a given room
func (s *RoomService) GetRoomMembers(ctx context.Context, roomID string) ([]string, error) {
	return s.repo.GetRoomMembers(ctx, roomID)
}

// RoomStats returns a room's stats
func (s *RoomService) RoomStats(ctx context.Context, roomID string) (*RoomStats, error) {
	members, err := s.repo.GetRoomMembers(ctx, roomID)
	if err != nil {
		return nil, err
	}

	room, err := s.repo.GetRoom(ctx, roomID)
	if err != nil {
		return nil, err
	}

	s.activeRoomsMu.RLock()
	activeCount := len(s.activeRooms[roomID].Members)
	s.activeRoomsMu.RUnlock()

	return &RoomStats{
		Room:          room,
		TotalMembers:  len(members),
		ActiveMembers: activeCount,
	}, nil
}

func (s *RoomService) handlePubSubMessages(roomID string, pubsub *redis.PubSub, events chan<- RoomEvent) {
	defer close(events)
	defer pubsub.Close()

	for {
		msg, err := pubsub.ReceiveMessage(context.Background())
		if err != nil {
			if errors.Is(err, redis.ErrClosed) {
				return
			}
			log.Printf("PubSub error: %v", err)
			continue
		}

		var event RoomEvent
		if err := json.Unmarshal([]byte(msg.Payload), &event); err != nil {
			log.Printf("Failed to unmarshal event: %v", err)
			continue
		}

		select {
		case events <- event:
		default:
			log.Println("Event channel full, dropping event")
		}
	}
}
