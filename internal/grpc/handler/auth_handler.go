package handler

import (
	"context"

	pb "studyroom/api/proto"
	"studyroom/internal/service"
)

type AuthHandler struct {
	pb.UnimplementedAuthServiceServer
	authSvc service.AuthService
}

func NewAuthHandler(authSvc service.AuthService) *AuthHandler {
	return &AuthHandler{authSvc: authSvc}
}

func (h *AuthHandler) Register(ctx context.Context, req *pb.RegisterRequest) (*pb.RegisterResponse, error) {
	err := h.authSvc.Register(req.Email, req.Password)
	if err != nil {
		return &pb.RegisterResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}
	return &pb.RegisterResponse{Success: true}, nil
}

func (h *AuthHandler) Login(ctx context.Context, req *pb.LoginRequest) (*pb.LoginResponse, error) {
	token, _, err := h.authSvc.Login(req.Email, req.Password)
	if err != nil {
		return &pb.LoginResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}
	return &pb.LoginResponse{
		Success:      true,
		SessionToken: token,
	}, nil
}

func (h *AuthHandler) Logout(ctx context.Context, req *pb.LogoutRequest) (*pb.LogoutResponse, error) {
	err := h.authSvc.Logout(req.SessionToken)
	if err != nil {
		return &pb.LogoutResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}
	return &pb.LogoutResponse{Success: true}, nil
}

func (h *AuthHandler) Me(ctx context.Context, req *pb.MeRequest) (*pb.MeResponse, error) {
	user, err := h.authSvc.CurrentUser(req.SessionToken)
	if err != nil {
		return &pb.MeResponse{
			Error: err.Error(),
		}, nil
	}
	return &pb.MeResponse{
		Id:      user.ID,
		Email:   user.Email,
		IsAdmin: user.IsAdmin,
	}, nil
}



