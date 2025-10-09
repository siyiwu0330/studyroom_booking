package repo

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type BookingRepo interface {
	Create(roomID, userID string, start, end string) (bookingID string, err error)
	Cancel(bookingID string, userID string) error
	HasOverlap(roomID string, start, end string) (bool, error)
	GetByID(bookingID string) (roomID string, userID string, start, end, status string, err error)
}

type bookingRepoMongo struct{ d *mongo.Database }

func NewBookingRepoMongo(d *mongo.Database) BookingRepo { return &bookingRepoMongo{d: d} }

func (r *bookingRepoMongo) Create(roomID, userID string, start, end string) (string, error) {
	roid, err := mustOID(roomID); if err != nil { return "", err }
	uid,  err := mustOID(userID); if err != nil { return "", err }
	res, err := r.d.Collection("bookings").InsertOne(context.Background(), bson.M{
		"room_id": roid, "user_id": uid,
		"start_at": start, "end_at": end, "status": "confirmed",
	})
	if err != nil { return "", err }
	return oidHex(res.InsertedID.(primitive.ObjectID)), nil
}

func (r *bookingRepoMongo) Cancel(bookingID string, userID string) error {
	bid, err := mustOID(bookingID); if err != nil { return err }
	uid, err := mustOID(userID);   if err != nil { return err }
	_, err = r.d.Collection("bookings").UpdateOne(context.Background(),
		bson.M{"_id": bid, "user_id": uid, "status": "confirmed"},
		bson.M{"$set": bson.M{"status": "cancelled"}},
	)
	return err
}

func (r *bookingRepoMongo) HasOverlap(roomID string, start, end string) (bool, error) {
	roid, err := mustOID(roomID); if err != nil { return false, err }
	cnt, err := r.d.Collection("bookings").CountDocuments(context.Background(), bson.M{
		"room_id": roid, "status": "confirmed",
		"$nor": []bson.M{{"end_at": bson.M{"$lte": start}}, {"start_at": bson.M{"$gte": end}}},
	})
	return cnt > 0, err
}

func (r *bookingRepoMongo) GetByID(bookingID string) (string, string, string, string, string, error) {
	bid, err := mustOID(bookingID); if err != nil { return "", "", "", "", "", err }
	var doc struct {
		ID     primitive.ObjectID `bson:"_id"`
		RoomID primitive.ObjectID `bson:"room_id"`
		UserID primitive.ObjectID `bson:"user_id"`
		Start  string             `bson:"start_at"`
		End    string             `bson:"end_at"`
		Status string             `bson:"status"`
	}
	err = r.d.Collection("bookings").FindOne(context.Background(), bson.M{"_id": bid}).Decode(&doc)
	if err != nil { return "", "", "", "", "", err }
	return oidHex(doc.RoomID), oidHex(doc.UserID), doc.Start, doc.End, doc.Status, nil
}