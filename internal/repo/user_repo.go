package repo

import (
	"context"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)
type UserRepo interface {
	Create(email string, passwordHash []byte) (id string, err error)
	GetByEmail(email string) (id string, pwHash []byte, isAdmin bool, err error)
	GetByID(id string) (email string, isAdmin bool, err error)
}

type userRepoMongo struct{ d *mongo.Database }

func NewUserRepoMongo(d *mongo.Database) UserRepo { return &userRepoMongo{d: d} }

func (r *userRepoMongo) Create(email string, passwordHash []byte) (string, error) {
	res, err := r.d.Collection("users").InsertOne(context.Background(), bson.M{
		"email":         email,
		"password_hash": passwordHash,
		"is_admin":      false,
	})
	if err != nil { return "", err }
	return oidHex(res.InsertedID.(primitive.ObjectID)), nil
}

func (r *userRepoMongo) GetByEmail(email string) (string, []byte, bool, error) {
	var doc struct {
		ID    primitive.ObjectID `bson:"_id"`
		Email string             `bson:"email"`
		Hash  []byte             `bson:"password_hash"`
		Admin bool               `bson:"is_admin"`
	}
	err := r.d.Collection("users").FindOne(context.Background(), bson.M{"email": email}).Decode(&doc)
	if err != nil { return "", nil, false, err }
	return oidHex(doc.ID), doc.Hash, doc.Admin, nil
}

func (r *userRepoMongo) GetByID(id string) (string, bool, error) {
	oid, err := mustOID(id); if err != nil { return "", false, err }
	var doc struct {
		Email string `bson:"email"`
		Admin bool   `bson:"is_admin"`
	}
	err = r.d.Collection("users").FindOne(context.Background(), bson.M{"_id": oid}).Decode(&doc)
	if err != nil { return "", false, err }
	return doc.Email, doc.Admin, nil
}