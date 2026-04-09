package models

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type User struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Username     string             `json:"username" bson:"username"`
	Phone        string             `json:"phone" bson:"phone"`
	PasswordHash string             `json:"-" bson:"password_hash"`
	Avatar       string             `json:"avatar" bson:"avatar"`
	LastSeenAt   *time.Time         `json:"last_seen_at,omitempty" bson:"last_seen_at,omitempty"`
	CreatedAt    time.Time          `json:"created_at" bson:"created_at"`
}

type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=30"`
	Phone    string `json:"phone" binding:"required,min=10,max=15"`
	Password string `json:"password" binding:"required,min=6"`
}

type LoginRequest struct {
	Phone    string `json:"phone" binding:"required,min=10,max=15"`
	Password string `json:"password" binding:"required"`
}

type AuthResponse struct {
	Token string `json:"token"`
	User  User   `json:"user"`
}
