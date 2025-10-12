package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"studyroom/internal/db"
	handlers "studyroom/internal/http/handler"
	"studyroom/internal/http/middleware"
	"studyroom/internal/repo"
	"studyroom/internal/service"
)

func main() {
	ctx := context.Background()

	// --- Mongo ---
	mongoURI := getenv("MONGODB_URI", "mongodb://root:example@localhost:27017/?authSource=admin")
	dbName := getenv("MONGODB_DB", "studyroom")
	mc, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil { log.Fatal(err) }
	if err := mc.Ping(ctx, nil); err != nil { log.Fatal(err) }
	mdb := mc.Database(dbName)
	if err := db.EnsureIndexes(ctx, mdb); err != nil { log.Fatal(err) }

	// --- Redis ---
	redisAddr := getenv("REDIS_ADDR", "localhost:6379")
	redisPass := os.Getenv("REDIS_PASSWORD") // empty is fine for dev
	redisDB  := 0
	rdb := redis.NewClient(&redis.Options{
		Addr:        redisAddr,
		Password:    redisPass,
		DB:          redisDB,
		ReadTimeout:  500 * time.Millisecond,
		WriteTimeout: 500 * time.Millisecond,
	})
	if err := rdb.Ping(ctx).Err(); err != nil { log.Fatalf("redis ping: %v", err) }

	// Optional admin seed
	if os.Getenv("ADMIN_EMAIL") != "" && os.Getenv("ADMIN_PASSWORD") != "" {
		if err := db.SeedAdminMongo(ctx, mdb, os.Getenv("ADMIN_EMAIL"), os.Getenv("ADMIN_PASSWORD")); err != nil {
			log.Printf("admin seed: %v", err)
		}
	}

	// --- Repos ---
	userRepo := repo.NewUserRepoMongo(mdb)
	sessRepo := repo.NewSessionRepoRedis(rdb) // <-- swapped to Redis
	roomRepo := repo.NewRoomRepoMongo(mdb)
	bookingRepo := repo.NewBookingRepoMongo(mdb)
	waitRepo := repo.NewWaitlistRepoMongo(mdb)

	// --- Services ---
	authSvc := service.NewAuthService(userRepo, sessRepo)
	bookingSvc := service.NewBookingService(roomRepo, bookingRepo, waitRepo)
	searchSvc := service.NewSearchService(roomRepo, bookingRepo)

	// --- HTTP ---
	r := gin.Default()
	r.GET("/", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true, "db": "mongo", "sessions": "redis"}) })

	authH := handlers.NewAuthHandler(authSvc)
	adminH := handlers.NewAdminHandler(bookingSvc)
	bookH := handlers.NewBookingHandler(bookingSvc)
	searchH := handlers.NewSearchHandler(searchSvc)

	r.POST("/register", authH.Register)
	r.POST("/login", authH.Login)
	r.POST("/logout", authH.Logout)
	r.GET("/me", middleware.Auth(authSvc), authH.Me)

	r.POST("/bookings", middleware.Auth(authSvc), bookH.Create)
	r.DELETE("/bookings/:id", middleware.Auth(authSvc), bookH.Cancel)
	r.POST("/waitlist", middleware.Auth(authSvc), bookH.JoinWaitlist)
	r.GET("/search", middleware.Auth(authSvc), searchH.SearchRooms)

	admin := r.Group("/admin", middleware.Auth(authSvc), middleware.Admin())
	{
		admin.POST("/rooms", adminH.CreateRoom)
		admin.GET("/rooms", adminH.ListRooms)
		admin.POST("/admin/rooms/:id/schedule", adminH.SetRoomSchedule)
	}

	log.Println("listening on http://localhost:8080")
	log.Fatal(r.Run(":8080"))
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" { return v }
	return def
}
