package room

import (
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
)

type RoomService struct {
	repo          Repository
	activeRooms   map[string]*RoomContext
	activeRoomsMu sync.RWMutex
}

type RoomContext struct {
	Members    map[string]struct{}
	CancelFunc context.CancelFunc
}

func NewRoomService(repo Repository) *RoomService {
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
	// Vérifie si le room existe
	exists, err := s.repo.RoomExists(ctx, roomID)
	if err != nil || !exists {
		return nil, errors.New("room does not exist")
	}

	// Ajoute l'utilisateur au room
	if err := s.repo.AddRoomMember(ctx, roomID, userID); err != nil {
		return nil, err
	}

	// Crée le channel d'événements
	events := make(chan RoomEvent, 10)

	// Gère la connexion Redis PubSub
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

	// Goroutine pour gérer les messages
	go func() {
		defer close(events)
		defer func(pubsub *redis.PubSub) {
			err := pubsub.Close()
			if err != nil {
				fmt.Println("pubsub close error:", err)
				return
			}
		}(pubsub)

		for {
			select {
			case <-roomCtx.Done():
				return
			default:
				msg, err := pubsub.ReceiveMessage(roomCtx)
				if err != nil {
					log.Printf("Error receiving message: %v", err)
					continue
				}

				events <- RoomEvent{
					Type:   EventUserJoined,
					RoomID: roomID,
					Payload: ChatMessage{
						Content: msg.Payload,
					},
				}
			}
		}
	}()

	// Notifie les autres utilisateurs
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

	// Notifie les autres utilisateurs
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
