package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Room struct {
	ID        primitive.ObjectID   `json:"id" bson:"_id,omitempty"`
	Name      string               `json:"name" bson:"name"`
	Type      string               `json:"type" bson:"type"` // "direct" or "group"
	Members   []primitive.ObjectID `json:"members" bson:"members"`
	CreatedAt time.Time            `json:"created_at" bson:"created_at"`
}

type CreateRoomRequest struct {
	Name    string   `json:"name" binding:"required"`
	Type    string   `json:"type" binding:"required,oneof=direct group"`
	Members []string `json:"members"`
}

type DirectRoomRequest struct {
	PeerID string `json:"peer_id" binding:"required"`
}
