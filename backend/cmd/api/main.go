// @title           Converge API
// @version         1.0
// @description     API for teacher availability management and bookings
// @BasePath        /api

package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	_ "github.com/RinZ5/converge/backend/docs"
	"github.com/RinZ5/converge/backend/internal/adapter"
	"github.com/RinZ5/converge/backend/internal/adapter/db"
	"github.com/RinZ5/converge/backend/internal/adapter/web"
	"github.com/RinZ5/converge/backend/internal/branch"
	"github.com/RinZ5/converge/backend/internal/commute"
	"github.com/RinZ5/converge/backend/internal/scheduling"
	"github.com/RinZ5/converge/backend/internal/shared"
	"github.com/RinZ5/converge/backend/internal/teacher"
	"github.com/RinZ5/converge/backend/internal/user"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

const (
	defaultCORSOrigin = "http://localhost:5173"
	serverAddr        = ":8080"
	shutdownTimeout   = 10 * time.Second
	requestTimeout    = 30 * time.Second
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 15 * time.Second
	idleTimeout       = 60 * time.Second
)

func corsConfig() cors.Config {
	allowedOriginsStr := os.Getenv("ALLOWED_ORIGINS")
	allowedOrigins := []string{defaultCORSOrigin}
	if allowedOriginsStr != "" {
		allowedOrigins = allowedOrigins[:0]
		for _, item := range strings.Split(allowedOriginsStr, ",") {
			if trimmed := strings.TrimSpace(item); trimmed != "" {
				allowedOrigins = append(allowedOrigins, trimmed)
			}
		}
	}
	return cors.Config{
		AllowOrigins:     allowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept"},
		AllowCredentials: false,
	}
}

func requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, _ := shared.ContextWithRequestID(c.Request.Context())
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func timeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), timeout)
		defer cancel()
		c.Request = c.Request.WithContext(ctx)
		c.Next()
	}
}

func main() {
	logger := shared.NewLogger()

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
		os.Exit(1)
	}

	availRepo := db.NewPostgresRepo(database)
	bookingRepo := db.NewBookingRepository(database)
	branchRepo := db.NewBranchRepository(database)
	userRepo := db.NewUserRepository(database)
	teacherSvc := teacher.NewService(availRepo, availRepo, availRepo, logger)
	branchSvc := branch.NewService(branchRepo, logger)
	userSvc := user.NewService(userRepo, logger)
	teacherRoster := adapter.NewTeacherRosterAdapter(teacherSvc)
	commuteConfigRepo := db.NewCommuteConfigRepository(database)
	commuteSvc := commute.NewService(commuteConfigRepo, logger)
	commuteAdapter := adapter.NewCommuteAdapter(commuteSvc)
	branchCapacityAdapter := adapter.NewBranchCapacityAdapter(branchSvc)

	scorer := scheduling.NewWeightedScorer()
	clpEngine := scheduling.NewCLPEngine(bookingRepo, teacherRoster, scorer, commuteAdapter, branchCapacityAdapter, logger)

	schedulingSvc := scheduling.NewSchedulingService(bookingRepo, availRepo, clpEngine)

	availHandler := web.NewAvailabilityHandler(teacherSvc, logger)
	bookingHandler := web.NewBookingHandler(schedulingSvc, logger)
	branchHandler := web.NewBranchHandler(branchSvc, logger)
	commuteHandler := web.NewCommuteHandler(commuteSvc, logger)
	userHandler := web.NewUserHandler(userSvc, logger)

	r := gin.Default()
	r.Use(requestIDMiddleware())
	r.Use(timeoutMiddleware(requestTimeout))
	r.Use(cors.New(corsConfig()))
	api := r.Group("/api")
	api.POST("/register", userHandler.Register)
	api.POST("/login", userHandler.Login)
	api.GET("/teachers", availHandler.GetTeachers)
	api.POST("/teachers", availHandler.CreateTeacher)
	api.PATCH("/teachers/:id/status", availHandler.UpdateTeacherStatus)
	api.PATCH("/teachers/:id/gender", availHandler.UpdateTeacherGender)
	api.GET("/availability", availHandler.GetAllAvailability)
	api.POST("/availability", availHandler.SubmitWeeklyAvailability)
	api.GET("/branches", branchHandler.GetBranches)
	api.PATCH("/branches/:id/capacity", branchHandler.UpdateBranchCapacity)
	api.GET("/commute", commuteHandler.GetCommute)
	api.PATCH("/commute", commuteHandler.UpdateCommute)
	api.GET("/subjects", availHandler.GetSubjects)
	api.POST("/bookings", bookingHandler.CreateBooking)
	api.GET("/bookings", bookingHandler.ListBookings)
	api.POST("/bookings/confirm", bookingHandler.ConfirmBooking)
	api.DELETE("/bookings/:id", bookingHandler.CancelBooking)

	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	r.GET("/health", func(c *gin.Context) {
		if err := database.PingContext(c.Request.Context()); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unhealthy"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "healthy"})
	})

	r.NoRoute(func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
	})

	srv := &http.Server{
		Addr:              serverAddr,
		Handler:           r,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	go func() {
		logger.Info("server starting", "addr", serverAddr, "tz", shared.LoadLocation().String())
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Error("server failed", "error", err)
			os.Exit(1)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")
	ctx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		logger.Error("server forced to shutdown", "error", err)
	}
	logger.Info("server stopped")
}
