package main

import (
	"log"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"

	"pollapp/config"
	"pollapp/db"
	"pollapp/handlers"
	"pollapp/middleware"
)

func main() {
	cfg := config.Load()

	mongoDB := db.ConnectMongo(cfg.MongoURI, cfg.MongoDBName)
	redisClient := db.ConnectRedis(cfg.RedisAddr, cfg.RedisPass)

	usersCol := mongoDB.Collection("users")
	pollsCol := mongoDB.Collection("polls")
	votesCol := mongoDB.Collection("votes")

	hub := handlers.NewHub(redisClient)
	authHandler := handlers.NewAuthHandler(usersCol, cfg)
	pollHandler := handlers.NewPollHandler(pollsCol, votesCol, redisClient, hub)

	router := gin.Default()
	router.Use(cors.New(cors.Config{
		AllowOrigins:     []string{cfg.FrontendURL},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Authorization"},
		AllowCredentials: true,
	}))

	router.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	api := router.Group("/api")
	{
		api.POST("/auth/signup", authHandler.Signup)
		api.POST("/auth/login", authHandler.Login)

		// Public: anyone with the link can view a poll and vote — no login needed to vote.
		api.GET("/polls/:id", pollHandler.GetPoll)
		api.POST("/polls/:id/vote", pollHandler.Vote)
		api.GET("/polls/:id/ws", handlers.PollSocket(hub, pollHandler))

		// Protected: only an authenticated user can create/manage polls.
		authGroup := api.Group("/")
		authGroup.Use(middleware.RequireAuth(cfg.JWTSecret))
		{
			authGroup.POST("polls", pollHandler.CreatePoll)
			authGroup.GET("polls", pollHandler.ListMyPolls)
		}
	}

	log.Printf("server listening on :%s", cfg.Port)
	if err := router.Run(":" + cfg.Port); err != nil {
		log.Fatalf("server failed: %v", err)
	}
}
