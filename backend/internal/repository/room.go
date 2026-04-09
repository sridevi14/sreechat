package repository

import (
	"context"
	"time"

	"github.com/sreechat/internal/models"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type RoomRepo struct {
	col *mongo.Collection
}

func NewRoomRepo(db *mongo.Database) *RoomRepo {
	col := db.Collection("rooms")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	col.Indexes().CreateOne(ctx, mongo.IndexModel{
		Keys:    bson.D{{Key: "members", Value: 1}},
		Options: options.Index(),
	})

	return &RoomRepo{col: col}
}

func (r *RoomRepo) Create(ctx context.Context, room *models.Room) error {
	room.CreatedAt = time.Now()
	res, err := r.col.InsertOne(ctx, room)
	if err != nil {
		return err
	}
	room.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

func (r *RoomRepo) FindByID(ctx context.Context, id primitive.ObjectID) (*models.Room, error) {
	var room models.Room
	err := r.col.FindOne(ctx, bson.M{"_id": id}).Decode(&room)
	if err != nil {
		return nil, err
	}
	return &room, nil
}

// FindByMember returns all rooms a user belongs to.
func (r *RoomRepo) FindByMember(ctx context.Context, userID primitive.ObjectID) ([]models.Room, error) {
	cursor, err := r.col.Find(ctx, bson.M{"members": userID})
	if err != nil {
		return nil, err
	}
	var rooms []models.Room
	if err := cursor.All(ctx, &rooms); err != nil {
		return nil, err
	}
	return rooms, nil
}

// FindDirectRoom finds an existing direct room between two users.
func (r *RoomRepo) FindDirectRoom(ctx context.Context, userA, userB primitive.ObjectID) (*models.Room, error) {
	var room models.Room
	err := r.col.FindOne(ctx, bson.M{
		"type":    "direct",
		"members": bson.M{"$all": bson.A{userA, userB}},
	}).Decode(&room)
	if err != nil {
		return nil, err
	}
	return &room, nil
}
