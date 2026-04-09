package handlers

import (
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sridhar/sreechat/internal/models"
	"github.com/sridhar/sreechat/internal/pubsub"
	"github.com/sridhar/sreechat/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

// PresenceHandler drives app-wide online/offline (not per-room WebSocket).
type PresenceHandler struct {
	userRepo *repository.UserRepo
	roomRepo *repository.RoomRepo
	pubsub   *pubsub.RedisPubSub
}

func NewPresenceHandler(ur *repository.UserRepo, rr *repository.RoomRepo, ps *pubsub.RedisPubSub) *PresenceHandler {
	return &PresenceHandler{userRepo: ur, roomRepo: rr, pubsub: ps}
}

// Heartbeat refreshes Redis "online" while the user has the app open (any screen).
func (h *PresenceHandler) Heartbeat(c *gin.Context) {
	userID, _ := c.Get("user_id")
	if err := h.pubsub.SetOnline(c.Request.Context(), userID.(string)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed"})
		return
	}
	c.Status(http.StatusNoContent)
}

// Offline updates last_seen, clears Redis online, and notifies all rooms the user belongs to (for open chats).
func (h *PresenceHandler) Offline(c *gin.Context) {
	userIDStr, _ := c.Get("user_id")
	usernameStr, _ := c.Get("username")
	uid := userIDStr.(string)
	username := usernameStr.(string)

	oid, err := primitive.ObjectIDFromHex(uid)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	now := time.Now()
	ctx := c.Request.Context()

	if err := h.userRepo.UpdateLastSeen(ctx, oid, now); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed"})
		return
	}
	if err := h.pubsub.ClearOnline(ctx, uid); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed"})
		return
	}

	rooms, err := h.roomRepo.FindByMember(ctx, oid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed"})
		return
	}

	payload := models.PresencePayload{
		UserID:     uid,
		Username:   username,
		Online:     false,
		LastSeenAt: &now,
	}

	for _, room := range rooms {
		rid := room.ID.Hex()
		out := &models.WSMessage{
			Type:    "presence",
			RoomID:  rid,
			Payload: payload,
		}
		if err := h.pubsub.Publish(ctx, rid, out); err != nil {
			log.Printf("presence offline publish room %s: %v", rid, err)
		}
	}

	c.Status(http.StatusNoContent)
}
