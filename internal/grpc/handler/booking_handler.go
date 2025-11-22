package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	pb "studyroom/api/proto"
	"studyroom/internal/models"
	"studyroom/internal/service"
	"studyroom/internal/twopc"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type BookingHandler struct {
	pb.UnimplementedBookingServiceServer
	bookingSvc     service.BookingService
	authSvc        service.AuthService
	coordinator    *twopc.Coordinator
	nodeID         string
	peers          map[string]string // Raft addresses (port 50052)
	peerGRPCAddrs  map[string]string // gRPC addresses (port 50051)
	raftNode       interface{ IsLeader() bool }
}

func NewBookingHandler(bookingSvc service.BookingService, authSvc service.AuthService, coordinator *twopc.Coordinator, nodeID string, peers map[string]string, raftNode interface{ IsLeader() bool }) *BookingHandler {
	// Convert Raft addresses to gRPC addresses (change port from 50052 to 50051)
	peerGRPCAddrs := make(map[string]string)
	for peerID, raftAddr := range peers {
		// Replace port 50052 with 50051 for gRPC
		grpcAddr := strings.Replace(raftAddr, ":50052", ":50051", 1)
		peerGRPCAddrs[peerID] = grpcAddr
	}
	
	return &BookingHandler{
		bookingSvc:    bookingSvc,
		authSvc:       authSvc,
		coordinator:   coordinator,
		nodeID:        nodeID,
		peers:         peers,
		peerGRPCAddrs: peerGRPCAddrs,
		raftNode:      raftNode,
	}
}

func (h *BookingHandler) CreateBooking(ctx context.Context, req *pb.CreateBookingRequest) (*pb.CreateBookingResponse, error) {
	// Q4: If this node is not the leader, forward the request to the leader
	if h.raftNode != nil && !h.raftNode.IsLeader() {
		return h.forwardToLeader(ctx, func(client pb.BookingServiceClient) (*pb.CreateBookingResponse, error) {
			return client.CreateBooking(ctx, req)
		})
	}

	// Get user from session token
	user, err := h.getUserFromToken(req.SessionToken)
	if err != nil {
		return &pb.CreateBookingResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// If coordinator is available and we have peers, use 2PC
	if h.coordinator != nil && len(h.peers) > 0 {
		return h.createBookingWith2PC(ctx, req, user.ID)
	}

	// Otherwise, use direct booking
	bookingID, err := h.bookingSvc.CreateBooking(req.RoomId, user.ID, req.Start, req.End)
	if err != nil {
		return &pb.CreateBookingResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &pb.CreateBookingResponse{
		Success:   true,
		BookingId: bookingID,
	}, nil
}

func (h *BookingHandler) createBookingWith2PC(ctx context.Context, req *pb.CreateBookingRequest, userID string) (*pb.CreateBookingResponse, error) {
	// Prepare operation data
	opData := map[string]interface{}{
		"type":    "create_booking",
		"room_id": req.RoomId,
		"user_id": userID,
		"start":   req.Start,
		"end":     req.End,
	}
	opJSON, _ := json.Marshal(opData)

	// Prepare participants
	participants := []twopc.Participant{
		{NodeID: h.nodeID, Address: h.peers[h.nodeID]},
	}
	for peerID, peerAddr := range h.peers {
		if peerID != h.nodeID {
			participants = append(participants, twopc.Participant{
				NodeID:  peerID,
				Address: peerAddr,
			})
		}
	}

	// Generate transaction ID
	txnID := generateTxnID()

	// Execute 2PC transaction
	err := h.coordinator.ExecuteTransaction(ctx, txnID, participants, string(opJSON))
	if err != nil {
		return &pb.CreateBookingResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// After successful 2PC, create booking locally
	bookingID, err := h.bookingSvc.CreateBooking(req.RoomId, userID, req.Start, req.End)
	if err != nil {
		return &pb.CreateBookingResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &pb.CreateBookingResponse{
		Success:   true,
		BookingId: bookingID,
	}, nil
}

func (h *BookingHandler) CancelBooking(ctx context.Context, req *pb.CancelBookingRequest) (*pb.CancelBookingResponse, error) {
	user, err := h.getUserFromToken(req.SessionToken)
	if err != nil {
		return &pb.CancelBookingResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	err = h.bookingSvc.CancelBooking(req.BookingId, user.ID)
	if err != nil {
		return &pb.CancelBookingResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &pb.CancelBookingResponse{Success: true}, nil
}

func (h *BookingHandler) JoinWaitlist(ctx context.Context, req *pb.JoinWaitlistRequest) (*pb.JoinWaitlistResponse, error) {
	user, err := h.getUserFromToken(req.SessionToken)
	if err != nil {
		return &pb.JoinWaitlistResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	err = h.bookingSvc.JoinWaitlist(req.RoomId, user.ID, req.Start, req.End)
	if err != nil {
		return &pb.JoinWaitlistResponse{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &pb.JoinWaitlistResponse{Success: true}, nil
}

// Helper functions
func (h *BookingHandler) getUserFromToken(token string) (*models.User, error) {
	if h.authSvc == nil {
		return nil, fmt.Errorf("auth service not available")
	}
	return h.authSvc.CurrentUser(token)
}

func generateTxnID() string {
	return fmt.Sprintf("txn-%d", time.Now().UnixNano())
}

// forwardToLeader forwards a request to the leader node
// Q4: Client request forwarding when connected to follower
// This tries all peers to find the leader (the one that can process the request)
func (h *BookingHandler) forwardToLeader(ctx context.Context, fn func(pb.BookingServiceClient) (*pb.CreateBookingResponse, error)) (*pb.CreateBookingResponse, error) {
	// Try each peer to find the leader
	// The leader is the one that can successfully process the request
	for peerID, grpcAddr := range h.peerGRPCAddrs {
		if peerID == h.nodeID {
			continue // Skip self
		}
		
		conn, err := grpc.Dial(grpcAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			continue // Try next peer
		}

		client := pb.NewBookingServiceClient(conn)
		resp, err := fn(client)
		conn.Close()
		
		if err == nil && resp != nil {
			// Successfully forwarded to leader
			return resp, nil
		}
		// If error, try next peer
	}

	return &pb.CreateBookingResponse{
		Success: false,
		Error:   "failed to forward request to leader: no leader found or all peers failed",
	}, nil
}

