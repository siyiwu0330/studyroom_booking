package repo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type WaitlistRepo interface {
	Enqueue(roomID, userID string, start, end string) error
	DequeueFirst(roomID string, start, end string) (userID string, ok bool, err error)
	Delete(roomID, userID string, start, end string) error
}

type waitlistRepoMongo struct{ d *mongo.Database }

func NewWaitlistRepoMongo(d *mongo.Database) WaitlistRepo { return &waitlistRepoMongo{d: d} }

func (r *waitlistRepoMongo) Enqueue(roomID, userID string, start, end string) error {
	roid, err := mustOID(roomID); if err != nil { return err }
	uid,  err := mustOID(userID); if err != nil { return err }
	_, err = r.d.Collection("waitlist").InsertOne(context.Background(), bson.M{
		"room_id": roid, "user_id": uid,
		"start_at": start, "end_at": end, "created_at": time.Now().UTC(),
	})
	return err
}

func (r *waitlistRepoMongo) DequeueFirst(roomID string, start, end string) (string, bool, error) {
	roid, err := mustOID(roomID); if err != nil { return "", false, err }
	var doc struct{ UserID primitive.ObjectID `bson:"user_id"` }
	err = r.d.Collection("waitlist").FindOneAndDelete(
		context.Background(),
		bson.M{"room_id": roid, "start_at": start, "end_at": end},
		options.FindOneAndDelete().SetSort(bson.D{{Key: "created_at", Value: 1}}),
	).Decode(&doc)
	if err == mongo.ErrNoDocuments { return "", false, nil }
	if err != nil { return "", false, err }
	return oidHex(doc.UserID), true, nil
}

func (r *waitlistRepoMongo) Delete(roomID, userID string, start, end string) error {
	roid, err := mustOID(roomID); if err != nil { return err }
	uid,  err := mustOID(userID); if err != nil { return err }
	_, err = r.d.Collection("waitlist").DeleteOne(context.Background(),
		bson.M{"room_id": roid, "user_id": uid, "start_at": start, "end_at": end})
	return err
}