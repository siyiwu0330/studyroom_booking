package main

import (
	"context"
	"log"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
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

	// connect to Mongo (docker-compose creds)
	mongoURI := getenv("MONGODB_URI", "mongodb://root:example@localhost:27017/?authSource=admin")
	dbName := getenv("MONGODB_DB", "studyroom")

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil { log.Print("oops") 
	log.Fatal(err) }
	if err := client.Ping(ctx, nil); err != nil { log.Fatal(err) }
	mdb := client.Database(dbName)

	// indexes + optional admin seed
	if err := db.EnsureIndexes(ctx, mdb); err != nil { log.Fatal(err) }
		if err := db.SeedAdminMongo(ctx, mdb, "admin", "adminadmin"); err != nil {
		log.Printf("admin seed: %v", err)
	}

	// Mongo repos
	userRepo := repo.NewUserRepoMongo(mdb)
	sessRepo := repo.NewSessionRepoMongo(mdb)
	roomRepo := repo.NewRoomRepoMongo(mdb)
	bookingRepo := repo.NewBookingRepoMongo(mdb)
	waitRepo := repo.NewWaitlistRepoMongo(mdb)

	// services (unchanged)
	authSvc := service.NewAuthService(userRepo, sessRepo)
	bookingSvc := service.NewBookingService(roomRepo, bookingRepo, waitRepo)
	searchSvc := service.NewSearchService(roomRepo, bookingRepo)

	// HTTP (unchanged)
	r := gin.Default()
	r.GET("/", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"ok": true, "db": "mongo"}) })

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
		admin.POST("/rooms/:id/schedule", adminH.SetRoomSchedule)
	}

	log.Println("listening on http://localhost:8080")
	log.Fatal(r.Run(":8080"))
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" { return v }
	return def
}
