package db

import (
	"context"
	"strings"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"golang.org/x/crypto/bcrypt"
)

func EnsureIndexes(ctx context.Context, d *mongo.Database) error {
	// users: unique email
	if _, err := d.Collection("users").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil { return err }

	// sessions: token unique
	if _, err := d.Collection("sessions").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "token", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil { return err }

	// rooms: unique name
	if _, err := d.Collection("rooms").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "name", Value: 1}},
		Options: options.Index().SetUnique(true),
	}); err != nil { return err }

	// bookings
	if _, err := d.Collection("bookings").Indexes().CreateMany(ctx, []mongo.IndexModel{
		{Keys: bson.D{{Key: "room_id", Value: 1}, {Key: "start_at", Value: 1}, {Key: "end_at", Value: 1}}},
		{Keys: bson.D{{Key: "status", Value: 1}}},
	}); err != nil { return err }

	// waitlist FIFO per (room_id, start, end)
	if _, err := d.Collection("waitlist").Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys: bson.D{{Key: "room_id", Value: 1}, {Key: "start_at", Value: 1}, {Key: "end_at", Value: 1}, {Key: "created_at", Value: 1}},
	}); err != nil { return err }

	return nil
}

func SeedAdminMongo(ctx context.Context, d *mongo.Database, email, password string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || len(password) < 8 { return nil }
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil { return err }
	_, err = d.Collection("users").UpdateOne(
		ctx,
		bson.M{"email": email},
		bson.M{"$set": bson.M{"email": email, "password_hash": hash, "is_admin": true, "created_at": time.Now().UTC()}},
		options.Update().SetUpsert(true),
	)
	return err
}
