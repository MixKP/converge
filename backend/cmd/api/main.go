// @title           Converge API
// @version         1.0
// @description     API for teacher availability management and bookings
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
	availSvc := service.NewAvailabilityService(repo)

	bookingRepo := db.NewBookingRepository(database)
	scorer := service.NewWeightedScorer()
	clpEngine := service.NewCLPEngine(bookingRepo, scorer, nil, nil)
	bookingSvc := service.NewBookingService(bookingRepo, clpEngine)

	availHandler := web.NewAvailabilityHandler(availSvc)
	bookingHandler := web.NewBookingHandler(bookingSvc)

	r := gin.Default()
	r.Use(cors.New(corsConfig()))
	api := r.Group("/api")
	api.GET("/teachers", availHandler.GetTeachers)
	api.GET("/availability", availHandler.GetAllAvailability)
	api.POST("/availability", availHandler.SubmitWeeklyAvailability)
	api.GET("/branches", availHandler.GetBranches)
	api.GET("/subjects", availHandler.GetSubjects)
	api.POST("/bookings", bookingHandler.CreateBooking)
	api.POST("/bookings/confirm", bookingHandler.ConfirmBooking)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	log.Println("Server starting on :8080")
	log.Println("API Documentation available on /swagger")
	if err := r.Run(":8080"); err != nil {
		log.Printf("Server failed: %v", err)
		return
	}
}
