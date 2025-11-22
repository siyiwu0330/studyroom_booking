package handler

import (
	"context"

	pb "studyroom/api/proto"
	"studyroom/internal/service"
)

type SearchHandler struct {
	pb.UnimplementedSearchServiceServer
	searchSvc service.SearchService
	authSvc  service.AuthService
}

func NewSearchHandler(searchSvc service.SearchService, authSvc service.AuthService) *SearchHandler {
	return &SearchHandler{
		searchSvc: searchSvc,
		authSvc:   authSvc,
	}
}

func (h *SearchHandler) SearchRooms(ctx context.Context, req *pb.SearchRoomsRequest) (*pb.SearchRoomsResponse, error) {
	// Verify session
	_, err := h.authSvc.CurrentUser(req.SessionToken)
	if err != nil {
		return &pb.SearchRoomsResponse{
			Error: err.Error(),
		}, nil
	}

	rooms, err := h.searchSvc.FindAvailable(int(req.MinCapacity), req.Start, req.End)
	if err != nil {
		return &pb.SearchRoomsResponse{
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

	return &pb.SearchRoomsResponse{
		Rooms: pbRooms,
	}, nil
}

