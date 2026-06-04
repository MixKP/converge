// @title           Converge API
// @version         1.0
// @description     API for teacher availability management and bookings
// @BasePath        /api

package main

import (
	"log/slog"
	"os"
	"strings"

	_ "github.com/RinZ5/converge/backend/docs"
	"github.com/RinZ5/converge/backend/internal/adapter/db"
	"github.com/RinZ5/converge/backend/internal/adapter/web"
	"github.com/RinZ5/converge/backend/internal/commute"
	"github.com/RinZ5/converge/backend/internal/room"
	"github.com/RinZ5/converge/backend/internal/scheduling"
	"github.com/RinZ5/converge/backend/internal/teacher"

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
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))

	database, err := db.InitDB()
	if err != nil {
		logger.Error("database init failed", "error", err)
		os.Exit(1)
	}
	defer func() {
		if err := db.CloseDB(database); err != nil {
			logger.Error("closing database", "error", err)
		}
	}()

	if err := db.AutoMigrate(database); err != nil {
		logger.Error("auto migrate failed", "error", err)
		return
	}

	availRepo := db.NewPostgresRepo(database)
	bookingRepo := db.NewBookingRepository(database)

	scorer := scheduling.NewWeightedScorer()
	clpEngine := scheduling.NewCLPEngine(bookingRepo, availRepo, scorer, &commute.Service{}, &room.Service{}, logger)

	schedulingSvc := scheduling.NewSchedulingService(bookingRepo, availRepo, clpEngine)
	teacherSvc := teacher.NewService(availRepo, logger)

	availHandler := web.NewAvailabilityHandler(teacherSvc, logger)
	bookingHandler := web.NewBookingHandler(schedulingSvc, logger)

	r := gin.Default()
	r.Use(cors.New(corsConfig()))
	api := r.Group("/api")
	api.GET("/teachers", availHandler.GetTeachers)
	api.GET("/availability", availHandler.GetAllAvailability)
	api.POST("/availability", availHandler.SubmitWeeklyAvailability)
	api.GET("/branches", availHandler.GetBranches)
	api.GET("/subjects", availHandler.GetSubjects)
	api.POST("/bookings", bookingHandler.CreateBooking)
	api.GET("/bookings", bookingHandler.ListBookings)
	api.POST("/bookings/confirm", bookingHandler.ConfirmBooking)
	api.DELETE("/bookings/:id", bookingHandler.CancelBooking)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	logger.Info("server starting", "addr", ":8080")
	if err := r.Run(":8080"); err != nil {
		logger.Error("server failed", "error", err)
	}
}
