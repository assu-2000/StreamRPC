package room

import (
	"context"
	"log"

	"github.com/assu-2000/StreamRPC/internal/pb"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

type RoomHandler struct {
	pb.UnimplementedRoomGrpcServiceServer
	service *RoomService
}

func NewGRPCHandler(service *RoomService) *RoomHandler {
	return &RoomHandler{service: service}
}

func (h *RoomHandler) CreateRoom(ctx context.Context, req *pb.CreateRoomRequest) (*pb.Room, error) {
	userID, ok := ctx.Value("user_id").(string)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "invalid user")
	}

	room, err := h.service.CreateRoom(ctx, req.Name, userID, req.IsPrivate)
	if err != nil {
		log.Printf("Failed to create room: %v", err)
		return nil, status.Error(codes.Internal, "failed to create room")
	}

	return &pb.Room{
		Id:        room.ID,
		Name:      room.Name,
		IsPrivate: room.IsPrivate,
	}, nil
}

func (h *RoomHandler) JoinRoom(req *pb.JoinRoomRequest, stream pb.RoomGrpcService_JoinRoomServer) error {
	userID, ok := stream.Context().Value("user_id").(string)
	if !ok {
		return status.Error(codes.Unauthenticated, "invalid user")
	}

	events, err := h.service.JoinRoom(stream.Context(), req.RoomId, userID)
	if err != nil {
		return status.Error(codes.Internal, "failed to join room")
	}

	for event := range events {
		var resp *pb.RoomEvent
		switch event.Type {
		case EventUserJoined:
			resp = &pb.RoomEvent{
				Event: &pb.RoomEvent_UserJoined{
					UserJoined: &pb.UserJoined{
						UserId: event.UserID,
					},
				},
			}
		case EventUserLeft:
			resp = &pb.RoomEvent{
				Event: &pb.RoomEvent_UserLeft{
					UserLeft: &pb.UserLeft{
						UserId: event.UserID,
					},
				},
			}
		case EventMessage:
			// Handled by MessageService
			continue
		}

		if err := stream.Send(resp); err != nil {
			h.service.LeaveRoom(stream.Context(), req.RoomId, userID)
			return err
		}
	}

	return nil
}
