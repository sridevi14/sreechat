package handlers

import (
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sridhar/sreechat/internal/middleware"
	"github.com/sridhar/sreechat/internal/models"
	"github.com/sridhar/sreechat/internal/pubsub"
	"github.com/sridhar/sreechat/internal/repository"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"golang.org/x/crypto/bcrypt"
)

type AuthHandler struct {
	userRepo  *repository.UserRepo
	jwtSecret string
	pubsub    *pubsub.RedisPubSub
}

func NewAuthHandler(ur *repository.UserRepo, secret string, ps *pubsub.RedisPubSub) *AuthHandler {
	return &AuthHandler{userRepo: ur, jwtSecret: secret, pubsub: ps}
}

func (h *AuthHandler) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hash password"})
		return
	}

	user := &models.User{
		Username:     req.Username,
		Phone:        req.Phone,
		PasswordHash: string(hash),
	}

	if err := h.userRepo.Create(c.Request.Context(), user); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": "user already exists"})
		return
	}

	token, err := middleware.GenerateToken(h.jwtSecret, user.ID.Hex(), user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusCreated, models.AuthResponse{Token: token, User: *user})
}

func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	user, err := h.userRepo.FindByPhone(c.Request.Context(), req.Phone)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "invalid credentials"})
		return
	}

	token, err := middleware.GenerateToken(h.jwtSecret, user.ID.Hex(), user.Username)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, models.AuthResponse{Token: token, User: *user})
}

func (h *AuthHandler) SearchUsers(c *gin.Context) {
	phone := c.Query("phone")
	if phone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "phone query parameter required"})
		return
	}

	userID, _ := c.Get("user_id")
	oid, err := primitive.ObjectIDFromHex(userID.(string))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid user id"})
		return
	}

	users, err := h.userRepo.SearchByPhone(c.Request.Context(), phone, oid)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "search failed"})
		return
	}
	c.JSON(http.StatusOK, users)
}

// GetPresenceBatch returns online/offline and last_seen_at for comma-separated user ids (max 50).
func (h *AuthHandler) GetPresenceBatch(c *gin.Context) {
	idsParam := strings.TrimSpace(c.Query("ids"))
	if idsParam == "" {
		c.JSON(http.StatusOK, gin.H{})
		return
	}

	parts := strings.Split(idsParam, ",")
	if len(parts) > 50 {
		parts = parts[:50]
	}

	out := make(map[string]gin.H)
	ctx := c.Request.Context()

	for _, raw := range parts {
		idStr := strings.TrimSpace(raw)
		if idStr == "" {
			continue
		}
		oid, err := primitive.ObjectIDFromHex(idStr)
		if err != nil {
			continue
		}

		online, err := h.pubsub.IsOnline(ctx, idStr)
		if err != nil {
			log.Printf("presence IsOnline: %v", err)
		}
		if online {
			out[idStr] = gin.H{"online": true}
			continue
		}

		user, err := h.userRepo.FindByID(ctx, oid)
		if err != nil || user == nil {
			out[idStr] = gin.H{"online": false}
			continue
		}

		resp := gin.H{"online": false}
		if user.LastSeenAt != nil {
			resp["last_seen_at"] = user.LastSeenAt.UTC().Format(time.RFC3339)
		}
		out[idStr] = resp
	}

	c.JSON(http.StatusOK, out)
}
