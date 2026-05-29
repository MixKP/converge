// @title           Converge API
// @version         1.0
// @description     API for teacher availability management and bookings
// @host            localhost:8080
// @BasePath        /api

package main

import (
	"log"
	"os"
	"strings"

	_ "github.com/RinZ5/converge/backend/docs"
	"github.com/RinZ5/converge/backend/internal/adapter/db"
	"github.com/RinZ5/converge/backend/internal/adapter/web"
	"github.com/RinZ5/converge/backend/internal/core/service"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

func corsConfig() cors.Config {
	allowedOriginsStr := os.Getenv("ALLOWED_ORIGINS")
	allowedOrigins := []string{"http://localhost:5173"}
	if allowedOriginsStr != "" {
		allowedOrigins = strings.Split(allowedOriginsStr, ",")
	}
	return cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		AllowCredentials: false,
	}
}

func main() {
	database, err := db.InitDB()
	if err != nil {
		log.Fatalf("Database init failed: %v", err)
	}
	defer func() {
		if err := db.CloseDB(database); err != nil {
			log.Printf("Error closing database: %v", err)
		}
	}()

	if err := db.AutoMigrate(database); err != nil {
		log.Printf("Migration failed: %v", err)
		return
	}

	repo := db.NewPostgresRepo(database)
	svc := service.NewAvailabilityService(repo)

	handler := web.NewAvailabilityHandler(svc)

	r := gin.Default()
	r.Use(cors.New(corsConfig()))
	api := r.Group("/api")
	api.GET("/teachers", handler.GetTeachers)
	api.POST("/availability", handler.SubmitWeeklyAvailability)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	log.Println("Server starting on :8080")
	log.Println("API Documentation available on /swagger")
	if err := r.Run(":8080"); err != nil {
		log.Printf("Server failed: %v", err)
		return
	}
}
