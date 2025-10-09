package repo

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type SessionRepo interface {
	Create(token string, userID string, expiresRFC3339 string) error
	Delete(token string) error
	Lookup(token string) (userID string, expiresRFC3339 string, err error)
}

type sessionRepoMongo struct{ d *mongo.Database }

func NewSessionRepoMongo(d *mongo.Database) SessionRepo { return &sessionRepoMongo{d: d} }

func (r *sessionRepoMongo) Create(token string, userID string, expires string) error {
	oid, err := mustOID(userID); if err != nil { return err }
	_, err = r.d.Collection("sessions").InsertOne(context.Background(), bson.M{
		"token":      token,
		"user_id":    oid,       // store as ObjectID
		"expires_at": expires,
		"created_at": time.Now().UTC(),
	})
	return err
}

func (r *sessionRepoMongo) Delete(token string) error {
	_, err := r.d.Collection("sessions").DeleteOne(context.Background(), bson.M{"token": token})
	return err
}

func (r *sessionRepoMongo) Lookup(token string) (string, string, error) {
	var doc struct {
		UserID    primitive.ObjectID `bson:"user_id"`
		ExpiresAt string             `bson:"expires_at"`
	}
	err := r.d.Collection("sessions").FindOne(context.Background(), bson.M{"token": token}).Decode(&doc)
	if err != nil { return "", "", err }
	return oidHex(doc.UserID), doc.ExpiresAt, nil
}