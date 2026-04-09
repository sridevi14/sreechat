package main

import (
	"context"
	"log"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/sridhar/sreechat/internal/config"
	"github.com/sridhar/sreechat/internal/handlers"
	"github.com/sridhar/sreechat/internal/hub"
	mw "github.com/sridhar/sreechat/internal/middleware"
	"github.com/sridhar/sreechat/internal/pubsub"
	"github.com/sridhar/sreechat/internal/repository"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func main() {
	cfg := config.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	mongoClient, err := mongo.Connect(ctx, options.Client().ApplyURI(cfg.MongoURI))
	if err != nil {
		log.Fatalf("mongo connect: %v", err)
	}
	defer mongoClient.Disconnect(context.Background())

	if err := mongoClient.Ping(ctx, nil); err != nil {
		log.Fatalf("mongo ping: %v", err)
	}
	log.Println("connected to MongoDB")

	db := mongoClient.Database(cfg.MongoDB)

	rdb := redis.NewClient(&redis.Options{Addr: cfg.RedisAddr, Password: cfg.RedisPassword})
	if err := rdb.Ping(ctx).Err(); err != nil {
		log.Fatalf("redis ping: %v", err)
	}
	log.Println("connected to Redis")

	userRepo := repository.NewUserRepo(db)
	roomRepo := repository.NewRoomRepo(db)
	msgRepo := repository.NewMessageRepo(db)

	wsHub := hub.NewHub()
	go wsHub.Run()

	ps := pubsub.NewRedisPubSub(rdb, wsHub)

	authHandler := handlers.NewAuthHandler(userRepo, cfg.JWTSecret, ps)
	roomHandler := handlers.NewRoomHandler(roomRepo, msgRepo, userRepo)
	presenceHandler := handlers.NewPresenceHandler(userRepo, roomRepo, ps)
	wsHandler := handlers.NewWSHandler(wsHub, ps, msgRepo, roomRepo, cfg.JWTSecret)

	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.CORSOrigin},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Authorization", "Content-Type"},
		AllowCredentials: true,
	}))

	api := r.Group("/api")
	{
		auth := api.Group("/auth")
		{
			auth.POST("/register", authHandler.Register)
			auth.POST("/login", authHandler.Login)
		}

		protected := api.Group("")
		protected.Use(mw.JWTAuth(cfg.JWTSecret))
		{
			protected.GET("/users/search", authHandler.SearchUsers)
			protected.GET("/users/presence", authHandler.GetPresenceBatch)
			protected.POST("/presence/heartbeat", presenceHandler.Heartbeat)
			protected.POST("/presence/offline", presenceHandler.Offline)
			protected.GET("/rooms", roomHandler.GetRooms)
			protected.POST("/rooms", roomHandler.CreateRoom)
			protected.POST("/rooms/direct", roomHandler.StartDirectChat)
			protected.GET("/rooms/:id/messages", roomHandler.GetMessages)
		}
	}

	r.GET("/ws", wsHandler.HandleWS)

	log.Printf("server starting on :%s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server error: %v", err)
	}
}
