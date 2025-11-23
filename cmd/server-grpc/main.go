package main

import (
	"context"
	"log"
	"net"
	"os"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"google.golang.org/grpc"

	"studyroom/api/proto"
	"studyroom/internal/db"
	grpchandler "studyroom/internal/grpc/handler"
	"studyroom/internal/raft"
	"studyroom/internal/repo"
	"studyroom/internal/service"
	"studyroom/internal/twopc"
)

func main() {
	ctx := context.Background()

	// Get configuration from environment
	nodeID := getenv("NODE_ID", "node1")
	grpcPort := getenv("GRPC_PORT", "50051")
	raftPort := getenv("RAFT_PORT", "50052")
	peersStr := getenv("PEERS", "") // Format: "node1:localhost:50052,node2:localhost:50053"

	// Parse peers
	peers := parsePeers(peersStr)

	// --- MongoDB ---
	mongoURI := getenv("MONGODB_URI", "mongodb://root:example@localhost:27017/?authSource=admin")
	dbName := getenv("MONGODB_DB", "studyroom")
	mc, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		log.Fatal(err)
	}
	if err := mc.Ping(ctx, nil); err != nil {
		log.Fatal(err)
	}
	mdb := mc.Database(dbName)
	if err := db.EnsureIndexes(ctx, mdb); err != nil {
		log.Fatal(err)
	}

	// --- Redis ---
	redisAddr := getenv("REDIS_ADDR", "localhost:6379")
	redisPass := os.Getenv("REDIS_PASSWORD")
	rdb := redis.NewClient(&redis.Options{
		Addr:        redisAddr,
		Password:    redisPass,
		DB:          0,
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 500 * time.Millisecond,
	})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis ping: %v", err)
	}

	// Optional admin seed
	if os.Getenv("ADMIN_EMAIL") != "" && os.Getenv("ADMIN_PASSWORD") != "" {
		if err := db.SeedAdminMongo(ctx, mdb, os.Getenv("ADMIN_EMAIL"), os.Getenv("ADMIN_PASSWORD")); err != nil {
			log.Printf("admin seed: %v", err)
		}
	}

	// --- Repos ---
	userRepo := repo.NewUserRepoMongo(mdb)
	sessRepo := repo.NewSessionRepoRedis(rdb)
	roomRepo := repo.NewRoomRepoMongo(mdb)
	bookingRepo := repo.NewBookingRepoMongo(mdb)
	waitRepo := repo.NewWaitlistRepoMongo(mdb)

	// --- Services ---
	authSvc := service.NewAuthService(userRepo, sessRepo)
	bookingSvc := service.NewBookingService(roomRepo, bookingRepo, waitRepo)
	searchSvc := service.NewSearchService(roomRepo, bookingRepo)

	// --- Raft Node ---
	raftNode := raft.NewNode(nodeID, "localhost:"+raftPort, peers)
	raftNode.Start()
	defer raftNode.Stop()

	// --- 2PC Coordinator ---
	coordinatorAddress := "localhost:" + grpcPort
	coordinator := twopc.NewCoordinator(raftNode, nodeID, coordinatorAddress)

	// --- 2PC Participant ---
	participant := twopc.NewParticipantNode(nodeID)
	participant.SetPrepareFunc(func(operation string, data map[string]interface{}) error {
		// Validate operation can be prepared
		log.Printf("[2PC] Preparing operation: %s", operation)
		return nil
	})
	participant.SetCommitFunc(func(operation string, data map[string]interface{}) error {
		// Execute the operation
		log.Printf("[2PC] Committing operation: %s", operation)
		return nil
	})
	participant.SetAbortFunc(func(operation string, data map[string]interface{}) error {
		// Rollback the operation
		log.Printf("[2PC] Aborting operation: %s", operation)
		return nil
	})

	// --- gRPC Handlers ---
	authH := grpchandler.NewAuthHandler(authSvc)
	bookingH := grpchandler.NewBookingHandler(bookingSvc, authSvc, coordinator, nodeID, peers, raftNode)
	searchH := grpchandler.NewSearchHandler(searchSvc, authSvc)
	adminH := grpchandler.NewAdminHandler(bookingSvc, authSvc)

	// --- gRPC Server ---
	grpcServer := grpc.NewServer()

	// Register services
	proto.RegisterAuthServiceServer(grpcServer, authH)
	proto.RegisterBookingServiceServer(grpcServer, bookingH)
	proto.RegisterSearchServiceServer(grpcServer, searchH)
	proto.RegisterAdminServiceServer(grpcServer, adminH)

	// Register Raft service
	raftServer := raft.NewRaftServer(raftNode)
	proto.RegisterRaftServiceServer(grpcServer, raftServer)

	// Register 2PC service (with coordinator support for phase-to-phase gRPC)
	twopcServer := twopc.NewTwoPCServerWithCoordinator(participant, coordinator)
	proto.RegisterTwoPCServiceServer(grpcServer, twopcServer)

	// Start gRPC server
	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}

	log.Printf("gRPC server listening on :%s", grpcPort)
	log.Printf("Raft node %s started", nodeID)
	if err := grpcServer.Serve(lis); err != nil {
		log.Fatalf("failed to serve: %v", err)
	}
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func parsePeers(peersStr string) map[string]string {
	peers := make(map[string]string)
	if peersStr == "" {
		return peers
	}
	// Format: "node1:localhost:50052,node2:localhost:50053"
	parts := strings.Split(peersStr, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		kv := strings.SplitN(part, ":", 2)
		if len(kv) == 2 {
			peers[kv[0]] = kv[1]
		}
	}
	return peers
}

