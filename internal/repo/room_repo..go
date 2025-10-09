package repo

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)


type RoomRepo interface {
	Create(name string, capacity int) (id string, err error)
	List() ([]RoomRow, error)
	SetSchedule(roomID string, start, end string, isOpen bool) error
	IsWithinOpenSchedule(roomID string, start, end string) (bool, error)
	FindAvailable(minCapacity int, start, end string) ([]RoomRow, error)
}

type RoomRow struct {
	ID       string
	Name     string
	Capacity int
}

type roomRepoMongo struct{ d *mongo.Database }

func NewRoomRepoMongo(d *mongo.Database) RoomRepo { return &roomRepoMongo{d: d} }

func (r *roomRepoMongo) Create(name string, capacity int) (string, error) {
	res, err := r.d.Collection("rooms").InsertOne(context.Background(), bson.M{
		"name": name, "capacity": capacity,
	})
	if err != nil { return "", err }
	return oidHex(res.InsertedID.(primitive.ObjectID)), nil
}

func (r *roomRepoMongo) List() ([]RoomRow, error) {
	cur, err := r.d.Collection("rooms").Find(context.Background(), bson.M{})
	if err != nil { return nil, err }
	defer cur.Close(context.Background())
	var out []RoomRow
	for cur.Next(context.Background()) {
		var doc struct {
			ID       primitive.ObjectID `bson:"_id"`
			Name     string             `bson:"name"`
			Capacity int                `bson:"capacity"`
		}
		if err := cur.Decode(&doc); err != nil { return nil, err }
		out = append(out, RoomRow{ID: oidHex(doc.ID), Name: doc.Name, Capacity: doc.Capacity})
	}
	return out, cur.Err()
}

func (r *roomRepoMongo) SetSchedule(roomID string, start, end string, isOpen bool) error {
	oid, err := mustOID(roomID); if err != nil { return err }
	_, err = r.d.Collection("room_schedules").InsertOne(context.Background(), bson.M{
		"room_id": oid, "start_at": start, "end_at": end, "is_open": isOpen,
	})
	return err
}

func (r *roomRepoMongo) IsWithinOpenSchedule(roomID string, start, end string) (bool, error) {
	oid, err := mustOID(roomID); if err != nil { return false, err }
	cnt, err := r.d.Collection("room_schedules").CountDocuments(context.Background(), bson.M{
		"room_id": oid, "is_open": true,
		"start_at": bson.M{"$lte": start},
		"end_at":   bson.M{"$gte": end},
	})
	return cnt > 0, err
}

func (r *roomRepoMongo) FindAvailable(minCapacity int, start, end string) ([]RoomRow, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
    defer cancel()

    // 1) filter by capacity
    cur, err := r.d.Collection("rooms").Find(ctx, bson.M{
        "capacity": bson.M{"$gte": minCapacity},
    }, options.Find().SetProjection(bson.M{"name": 1, "capacity": 1}))
    if err != nil { return nil, err }
    defer cur.Close(ctx)

    var out []RoomRow
    for cur.Next(ctx) {
        var rr struct {
            ID       primitive.ObjectID `bson:"_id"`
            Name     string             `bson:"name"`
            Capacity int                `bson:"capacity"`
        }
        if err := cur.Decode(&rr); err != nil { return nil, err }

		fmt.Println(start)

        // 2) must have an OPEN schedule that fully covers [start, end)
        openCnt, err := r.d.Collection("room_schedules").CountDocuments(ctx, bson.M{
            "room_id":  rr.ID,
            "is_open":  true,
            "start_at": bson.M{"$lte": start},
            "end_at":   bson.M{"$gte": end},
        })
        if err != nil { return nil, err }
        if openCnt == 0 { continue } // no open window => skip

		fmt.Println(openCnt)

        // 3) must NOT have an overlapping confirmed booking
        bookCnt, err := r.d.Collection("bookings").CountDocuments(ctx, bson.M{
            "room_id": rr.ID,
            "status":  "confirmed",
            // overlap if NOT (end_at <= start || start_at >= end)
            "$nor": []bson.M{
                {"end_at":   bson.M{"$lte": start}},
                {"start_at": bson.M{"$gte": end}},
            },
        })
        if err != nil { return nil, err }
        if bookCnt > 0 { continue } // conflict => skip

        out = append(out, RoomRow{ID: rr.ID.Hex(), Name: rr.Name, Capacity: rr.Capacity})
    }
    return out, cur.Err()
}
