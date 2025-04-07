package room

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/emptypb"
	"google.golang.org/protobuf/types/known/timestamppb"
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
	userID, ok := ctx.Value("user_id").(uuid.UUID)
	if !ok {
		return nil, status.Error(codes.Unauthenticated, "invalid user")
	}

	room, err := h.service.CreateRoom(ctx, req.Name, userID.String(), req.IsPrivate)
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
	userID, ok := stream.Context().Value("user_id").(uuid.UUID)
	fmt.Println("userid ", userID)
	if !ok {
		return status.Error(codes.Unauthenticated, "invalid user")
	}

	events, err := h.service.JoinRoom(stream.Context(), req.RoomId, userID.String())
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
		default:
			fmt.Println("unhandled case")
		}

		if err := stream.Send(resp); err != nil {
			err := h.service.LeaveRoom(stream.Context(), req.RoomId, userID.String())
			if err != nil {
				fmt.Printf("Failed to send response to user: %v", err)
				return err
			}
			return err
		}
	}

	return nil
}

func (h *RoomHandler) GetRoomStats(ctx context.Context, req *pb.RoomID) (*pb.RoomStatsResponse, error) {
	stats, err := h.service.RoomStats(ctx, req.Id)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "room not found")
	}

	return &pb.RoomStatsResponse{
		Room:          convertToPbRoom(stats.Room),
		TotalMembers:  int32(stats.TotalMembers),
		ActiveMembers: int32(stats.ActiveMembers),
	}, nil
}

func (h *RoomHandler) LeaveRoom(ctx context.Context, req *pb.LeaveRoomRequest) (*emptypb.Empty, error) {
	if err := h.service.LeaveRoom(ctx, req.RoomId, req.UserId); err != nil {
		log.Printf("LeaveRoom failed: %v", err)
		return nil, status.Error(codes.Internal, "failed to leave room")
	}
	return &emptypb.Empty{}, nil
}

func (h *RoomHandler) GetRoom(ctx context.Context, req *pb.GetRoomRequest) (*pb.Room, error) {
	room, err := h.service.GetRoom(ctx, req.RoomId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "room not found")
	}

	return &pb.Room{
		Id:          room.ID,
		Name:        room.Name,
		CreatedBy:   room.CreatedBy,
		CreatedAt:   timestamppb.New(room.CreatedAt),
		IsPrivate:   room.IsPrivate,
		MemberCount: uint32(0), // TODO -readjust the struct
	}, nil
}

func (h *RoomHandler) ListRooms(ctx context.Context, _ *emptypb.Empty) (*pb.ListRoomsResponse, error) {
	rooms, err := h.service.ListRooms(ctx)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list rooms")
	}

	pbRooms := make([]*pb.Room, 0, len(rooms))
	for _, r := range rooms {
		pbRooms = append(pbRooms, &pb.Room{
			Id:        r.ID,
			Name:      r.Name,
			CreatedBy: r.CreatedBy,
			CreatedAt: timestamppb.New(r.CreatedAt),
		})
	}

	return &pb.ListRoomsResponse{Rooms: pbRooms}, nil
}

func (h *RoomHandler) DeleteRoom(ctx context.Context, req *pb.DeleteRoomRequest) (*emptypb.Empty, error) {
	if err := h.service.DeleteRoom(ctx, req.RoomId); err != nil {
		return nil, status.Error(codes.Internal, "failed to delete room")
	}
	return &emptypb.Empty{}, nil
}

func (h *RoomHandler) GetRoomMembers(ctx context.Context, req *pb.GetRoomRequest) (*pb.RoomMembers, error) {
	members, err := h.service.GetRoomMembers(ctx, req.RoomId)
	if err != nil {
		return nil, status.Error(codes.NotFound, "room not found")
	}

	return &pb.RoomMembers{UserIds: members}, nil
}

func convertToPbRoom(room *Room) *pb.Room {
	if room == nil {
		return nil
	}

	return &pb.Room{
		Id:        room.ID,
		Name:      room.Name,
		CreatedBy: room.CreatedBy,
		CreatedAt: timestamppb.New(room.CreatedAt),
		IsPrivate: room.IsPrivate,
	}
}
