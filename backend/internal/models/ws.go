package models

import "time"

// WSMessage is the envelope for all WebSocket communication.
type WSMessage struct {
	Type    string      `json:"type"`
	RoomID  string      `json:"room_id"`
	Payload interface{} `json:"payload"`
}

// ChatPayload is the payload for type="message".
type ChatPayload struct {
	Content  string `json:"content"`
	SenderID string `json:"sender_id"`
	Username string `json:"username"`
	Seq      int64  `json:"seq"`
	CreatedAt string `json:"created_at,omitempty"`
}

// TypingPayload is the payload for type="typing".
type TypingPayload struct {
	SenderID string `json:"sender_id"`
	Username string `json:"username"`
	IsTyping bool   `json:"is_typing"`
}

// PresencePayload is the payload for type="presence".
type PresencePayload struct {
	UserID     string     `json:"user_id"`
	Username   string     `json:"username"`
	Online     bool       `json:"online"`
	LastSeenAt *time.Time `json:"last_seen_at,omitempty"`
}
