package models

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
}

// TypingPayload is the payload for type="typing".
type TypingPayload struct {
	SenderID string `json:"sender_id"`
	Username string `json:"username"`
	IsTyping bool   `json:"is_typing"`
}
