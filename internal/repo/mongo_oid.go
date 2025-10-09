package repo

import (
	"errors"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

func mustOID(hex string) (primitive.ObjectID, error) {
	if hex == "" {
		return primitive.NilObjectID, errors.New("empty id")
	}
	return primitive.ObjectIDFromHex(hex)
}

func oidHex(id primitive.ObjectID) string {
	return id.Hex()
}
