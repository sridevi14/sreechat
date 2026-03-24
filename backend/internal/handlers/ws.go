package handlers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/gorilla/websocket"
	"github.com/sridhar/sreechat/internal/hub"
	"github.com/sridhar/sreechat/internal/models"
	"github.com/sridhar/sreechat/internal/pubsub"
	"github.com/sridhar/sreechat/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type WSHandler struct {
	hub       *hub.Hub
	pubsub    *pubsub.RedisPubSub
	msgRepo   *repository.MessageRepo
	roomRepo  *repository.RoomRepo
	jwtSecret string
}

func NewWSHandler(h *hub.Hub, ps *pubsub.RedisPubSub, mr *repository.MessageRepo, rr *repository.RoomRepo, secret string) *WSHandler {
	return &WSHandler{hub: h, pubsub: ps, msgRepo: mr, roomRepo: rr, jwtSecret: secret}
}

func (h *WSHandler) HandleWS(c *gin.Context) {
	tokenStr := c.Query("token")
	if tokenStr == "" {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "missing token"})
		return
	}

	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		return []byte(h.jwtSecret), nil
	})
	if err != nil || !token.Valid {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid token"})
		return
	}
	claims := token.Claims.(jwt.MapClaims)
	userID := claims["sub"].(string)
	username := claims["username"].(string)
	fmt.Println(userID, username)

	roomID := c.Query("room_id")
	fmt.Printf("User %s joining room %s\n", userID, roomID)
	if roomID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing room_id"})
		return
	}

	roomOID, err := primitive.ObjectIDFromHex(roomID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room_id"})
		return
	}

	userOID, err := primitive.ObjectIDFromHex(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	room, err := h.roomRepo.FindByID(c.Request.Context(), roomOID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "room not found"})
		return
	}

	isMember := false
	for _, m := range room.Members {
		if m == userOID {
			isMember = true
			break
		}
	}
	if !isMember {
		c.JSON(http.StatusForbidden, gin.H{"error": "not a member of this room"})
		return
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Printf("ws upgrade error: %v", err)
		return
	}

	client := &hub.Client{
		ID:     userID + ":" + roomID,
		UserID: userID,
		RoomID: roomID,
		Conn:   conn,
		Send:   make(chan []byte, 256),
	}

	h.hub.Register <- client

	h.pubsub.Subscribe(roomID)
	h.pubsub.SetOnline(context.Background(), userID)

	go hub.WritePump(client)
	go h.readPump(client, username)
}

func (h *WSHandler) readPump(client *hub.Client, username string) {
	defer func() {
		h.hub.Unregister <- client
		client.Conn.Close()
	}()

	for {
		_, raw, err := client.Conn.ReadMessage()
		if err != nil {
			break
		}

		var wsMsg models.WSMessage
		if err := json.Unmarshal(raw, &wsMsg); err != nil {
			continue
		}

		switch wsMsg.Type {
		case "message":
			h.handleChatMessage(client, username, &wsMsg)
		case "typing":
			h.handleTyping(client, username, &wsMsg)
		}
	}
}

func (h *WSHandler) handleChatMessage(client *hub.Client, username string, wsMsg *models.WSMessage) {
	payloadBytes, _ := json.Marshal(wsMsg.Payload)
	var chatPayload models.ChatPayload
	json.Unmarshal(payloadBytes, &chatPayload)

	ctx := context.Background()

	seq, err := h.pubsub.NextSeq(ctx, client.RoomID)
	if err != nil {
		log.Printf("seq error: %v", err)
		return
	}

	roomOID, _ := primitive.ObjectIDFromHex(client.RoomID)
	senderOID, _ := primitive.ObjectIDFromHex(client.UserID)

	msg := &models.Message{
		RoomID:   roomOID,
		SenderID: senderOID,
		Content:  chatPayload.Content,
		Seq:      seq,
	}

	if err := h.msgRepo.Create(ctx, msg); err != nil {
		log.Printf("save message error: %v", err)
		return
	}

	outPayload := models.ChatPayload{
		Content:  chatPayload.Content,
		SenderID: client.UserID,
		Username: username,
		Seq:      seq,
	}
	outMsg := &models.WSMessage{
		Type:    "message",
		RoomID:  client.RoomID,
		Payload: outPayload,
	}

	if err := h.pubsub.Publish(ctx, client.RoomID, outMsg); err != nil {
		log.Printf("publish error: %v", err)
	}

	// Clear typing indicator for this user after sending
	stopTyping := &models.WSMessage{
		Type:   "typing",
		RoomID: client.RoomID,
		Payload: models.TypingPayload{
			SenderID: client.UserID,
			Username: username,
			IsTyping: false,
		},
	}
	h.pubsub.Publish(ctx, client.RoomID, stopTyping)
}

func (h *WSHandler) handleTyping(client *hub.Client, username string, wsMsg *models.WSMessage) {
	payloadBytes, _ := json.Marshal(wsMsg.Payload)
	var tp models.TypingPayload
	json.Unmarshal(payloadBytes, &tp)

	outMsg := &models.WSMessage{
		Type:   "typing",
		RoomID: client.RoomID,
		Payload: models.TypingPayload{
			SenderID: client.UserID,
			Username: username,
			IsTyping: tp.IsTyping,
		},
	}
	h.pubsub.Publish(context.Background(), client.RoomID, outMsg)
}
