package repo

import (
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// Convert Mongo _id (ObjectID or int64 stored in _int_id) into our int64 identity.
// In this project we store a parallel integer key as _int_id to keep interfaces unchanged.
// If you prefer ObjectID everywhere, change interfaces to use string/ObjectID.
func toIntID(v interface{}) int64 {
	switch t := v.(type) {
	case int64:
		return t
	case int32:
		return int64(t)
	case float64:
		return int64(t)
	case primitive.ObjectID:
		// fallback hash: take last 8 bytes as int64 (not collision-safe; better store _int_id explicitly)
		b := t[:]
		var out int64
		for i := 12 - 8; i < 12; i++ { out = (out << 8) | int64(b[i]) }
		return out
	default:
		return 0
	}
}
