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

type MessageRepo struct {
	col *mongo.Collection
}

func NewMessageRepo(db *mongo.Database) *MessageRepo {
	col := db.Collection("messages")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	col.Indexes().CreateMany(ctx, []mongo.IndexModel{
		{
			Keys:    bson.D{{Key: "room_id", Value: 1}, {Key: "seq", Value: 1}},
			Options: options.Index().SetUnique(true),
		},
	})

	return &MessageRepo{col: col}
}

func (r *MessageRepo) Create(ctx context.Context, msg *models.Message) error {
	msg.CreatedAt = time.Now()
	res, err := r.col.InsertOne(ctx, msg)
	if err != nil {
		return err
	}
	msg.ID = res.InsertedID.(primitive.ObjectID)
	return nil
}

// GetHistory returns messages for a room, paginated by seq number (cursor-based).
// afterSeq=0 means fetch from the latest. Returns messages in descending seq order.
func (r *MessageRepo) GetHistory(ctx context.Context, roomID primitive.ObjectID, afterSeq int64, limit int64) ([]models.Message, error) {
	filter := bson.M{"room_id": roomID}
	if afterSeq > 0 {
		filter["seq"] = bson.M{"$lt": afterSeq}
	}

	opts := options.Find().
		SetSort(bson.D{{Key: "seq", Value: -1}}).
		SetLimit(limit)

	cursor, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	var messages []models.Message
	if err := cursor.All(ctx, &messages); err != nil {
		return nil, err
	}
	return messages, nil
}

// GetBySeqRange fetches messages within a seq range (for gap filling).
func (r *MessageRepo) GetBySeqRange(ctx context.Context, roomID primitive.ObjectID, fromSeq, toSeq int64) ([]models.Message, error) {
	filter := bson.M{
		"room_id": roomID,
		"seq":     bson.M{"$gte": fromSeq, "$lte": toSeq},
	}
	opts := options.Find().SetSort(bson.D{{Key: "seq", Value: 1}})

	cursor, err := r.col.Find(ctx, filter, opts)
	if err != nil {
		return nil, err
	}
	var messages []models.Message
	if err := cursor.All(ctx, &messages); err != nil {
		return nil, err
	}
	return messages, nil
}
