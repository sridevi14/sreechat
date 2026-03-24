package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Message struct {
	ID        primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	RoomID    primitive.ObjectID `json:"room_id" bson:"room_id"`
	SenderID  primitive.ObjectID `json:"sender_id" bson:"sender_id"`
	Content   string             `json:"content" bson:"content"`
	Seq       int64              `json:"seq" bson:"seq"`
	CreatedAt time.Time          `json:"created_at" bson:"created_at"`
}
