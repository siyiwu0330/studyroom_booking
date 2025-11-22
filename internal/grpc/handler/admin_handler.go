package handler

import (
	"context"

	pb "studyroom/api/proto"
	"studyroom/internal/service"
)

type AdminHandler struct {
	pb.UnimplementedAdminServiceServer
	bookingSvc service.BookingService
	authSvc    service.AuthService
}

func NewAdminHandler(bookingSvc service.BookingService, authSvc service.AuthService) *AdminHandler {
	return &AdminHandler{
		bookingSvc: bookingSvc,
		authSvc:    authSvc,
	}
}

func (h *AdminHandler) CreateRoom(ctx context.Context, req *pb.CreateRoomRequest) (*pb.CreateRoomResponse, error) {
	user, err := h.authSvc.CurrentUser(req.SessionToken)
	if err != nil {
		return &pb.CreateRoomResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	if !user.IsAdmin {
		return &pb.CreateRoomResponse{
			Success: false,
			Error:   "unauthorized: admin access required",
		}, nil
	}

	roomID, err := h.bookingSvc.CreateRoom(req.Name, int(req.Capacity))
	if err != nil {
		return &pb.CreateRoomResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &pb.CreateRoomResponse{
		Success: true,
		RoomId:   roomID,
	}, nil
}

func (h *AdminHandler) ListRooms(ctx context.Context, req *pb.ListRoomsRequest) (*pb.ListRoomsResponse, error) {
	user, err := h.authSvc.CurrentUser(req.SessionToken)
	if err != nil {
		return &pb.ListRoomsResponse{
			Error: err.Error(),
		}, nil
	}

	if !user.IsAdmin {
		return &pb.ListRoomsResponse{
			Error: "unauthorized: admin access required",
		}, nil
	}

	rooms, err := h.bookingSvc.ListRooms()
	if err != nil {
		return &pb.ListRoomsResponse{
			Error: err.Error(),
		}, nil
	}

	pbRooms := make([]*pb.Room, len(rooms))
	for i, r := range rooms {
		pbRooms[i] = &pb.Room{
			Id:       r.ID,
			Name:     r.Name,
			Capacity: int32(r.Capacity),
		}
	}

	return &pb.ListRoomsResponse{
		Rooms: pbRooms,
	}, nil
}

func (h *AdminHandler) SetRoomSchedule(ctx context.Context, req *pb.SetRoomScheduleRequest) (*pb.SetRoomScheduleResponse, error) {
	user, err := h.authSvc.CurrentUser(req.SessionToken)
	if err != nil {
		return &pb.SetRoomScheduleResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	if !user.IsAdmin {
		return &pb.SetRoomScheduleResponse{
			Success: false,
			Error:   "unauthorized: admin access required",
		}, nil
	}

	err = h.bookingSvc.SetRoomSchedule(req.RoomId, req.Start, req.End, req.IsOpen)
	if err != nil {
		return &pb.SetRoomScheduleResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &pb.SetRoomScheduleResponse{
		Success: true,
	}, nil
}



