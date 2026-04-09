package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/sridhar/sreechat/internal/models"
	"github.com/sridhar/sreechat/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type RoomHandler struct {
	roomRepo *repository.RoomRepo
	msgRepo  *repository.MessageRepo
	userRepo *repository.UserRepo
}

func NewRoomHandler(rr *repository.RoomRepo, mr *repository.MessageRepo, ur *repository.UserRepo) *RoomHandler {
	return &RoomHandler{roomRepo: rr, msgRepo: mr, userRepo: ur}
}

func (h *RoomHandler) CreateRoom(c *gin.Context) {
	var req models.CreateRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	creatorID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	members := []primitive.ObjectID{creatorID}
	for _, mid := range req.Members {
		oid, err := primitive.ObjectIDFromHex(mid)
		if err != nil {
			continue
		}
		if oid != creatorID {
			members = append(members, oid)
		}
	}

	room := &models.Room{
		Name:    req.Name,
		Type:    req.Type,
		Members: members,
	}

	if err := h.roomRepo.Create(c.Request.Context(), room); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create room"})
		return
	}

	c.JSON(http.StatusCreated, room)
}

func (h *RoomHandler) GetRooms(c *gin.Context) {
	userID, _ := c.Get("user_id")
	oid, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	rooms, err := h.roomRepo.FindByMember(c.Request.Context(), oid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch rooms"})
		return
	}

	// For direct rooms, set the name to the other member's username
	for i := range rooms {
		if rooms[i].Type == "direct" && len(rooms[i].Members) == 2 {
			var otherID primitive.ObjectID
			for _, m := range rooms[i].Members {
				if m != oid {
					otherID = m
					break
				}
			}
			if !otherID.IsZero() {
				otherUser, err := h.userRepo.FindByID(c.Request.Context(), otherID)
				if err == nil && otherUser != nil {
					rooms[i].Name = otherUser.Username
				}
			}
		}
	}

	c.JSON(http.StatusOK, rooms)
}

func (h *RoomHandler) GetMessages(c *gin.Context) {
	roomID, err := primitive.ObjectIDFromHex(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid room id"})
		return
	}

	afterSeq := int64(0)
	if s := c.Query("after"); s != "" {
		afterSeq, _ = strconv.ParseInt(s, 10, 64)
	}

	limit := int64(50)
	if l := c.Query("limit"); l != "" {
		limit, _ = strconv.ParseInt(l, 10, 64)
		if limit > 100 {
			limit = 100
		}
	}

	messages, err := h.msgRepo.GetHistory(c.Request.Context(), roomID, afterSeq, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch messages"})
		return
	}

	ctx := c.Request.Context()
	seen := make(map[primitive.ObjectID]struct{})
	var senderIDs []primitive.ObjectID
	for _, m := range messages {
		if _, ok := seen[m.SenderID]; ok {
			continue
		}
		seen[m.SenderID] = struct{}{}
		senderIDs = append(senderIDs, m.SenderID)
	}
	if len(senderIDs) > 0 {
		users, err := h.userRepo.FindByIDs(ctx, senderIDs)
		if err == nil {
			nameByID := make(map[string]string, len(users))
			for _, u := range users {
				nameByID[u.ID.Hex()] = u.Username
			}
			for i := range messages {
				if name, ok := nameByID[messages[i].SenderID.Hex()]; ok {
					messages[i].Username = name
				}
			}
		}
	}

	c.JSON(http.StatusOK, messages)
}

// StartDirectChat finds or creates a direct room between the caller and a peer.
func (h *RoomHandler) StartDirectChat(c *gin.Context) {
	var req models.DirectRoomRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	userID, _ := c.Get("user_id")
	callerID, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	peerID, err := primitive.ObjectIDFromHex(req.PeerID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid peer id"})
		return
	}

	if callerID == peerID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cannot chat with yourself"})
		return
	}

	// Check if a direct room already exists
	existing, err := h.roomRepo.FindDirectRoom(c.Request.Context(), callerID, peerID)
	if err == nil && existing != nil {
		c.JSON(http.StatusOK, existing)
		return
	}
	if err != nil && err != mongo.ErrNoDocuments {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to check existing room"})
		return
	}

	// Look up peer to use their username as the room name
	peer, err := h.userRepo.FindByID(c.Request.Context(), peerID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "peer not found"})
		return
	}

	room := &models.Room{
		Name:    peer.Username,
		Type:    "direct",
		Members: []primitive.ObjectID{callerID, peerID},
	}

	if err := h.roomRepo.Create(c.Request.Context(), room); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create room"})
		return
	}

	c.JSON(http.StatusCreated, room)
}
