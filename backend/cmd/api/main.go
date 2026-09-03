// @title           Converge API
// @version         1.0
// @description     API for teacher availability management and bookings
// @BasePath        /api

// @securityDefinitions.apikey  BearerAuth
// @in                          header
// @name                        Authorization
// @description                 Type "Bearer" followed by a space and the JWT from /login.

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
	"github.com/RinZ5/converge/backend/internal/adapter/token"
	"github.com/RinZ5/converge/backend/internal/adapter/web"
	"github.com/RinZ5/converge/backend/internal/branch"
	"github.com/RinZ5/converge/backend/internal/commute"
	"github.com/RinZ5/converge/backend/internal/scheduling"
	"github.com/RinZ5/converge/backend/internal/shared"
	"github.com/RinZ5/converge/backend/internal/teacher"
	"github.com/RinZ5/converge/backend/internal/user"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
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
	defaultJWTTTL     = 24 * time.Hour
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
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		AllowCredentials: false,
	}
}

func jwtTTL() time.Duration {
	raw := os.Getenv("JWT_TTL")
	if raw == "" {
		return defaultJWTTTL
	}
	d, err := time.ParseDuration(raw)
	if err != nil || d <= 0 {
		return defaultJWTTTL
	}
	return d
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

	// Load .env before reading any configuration, so env values in the file are
	// visible to the checks below (InitDB also loads it, but that runs later).
	_ = godotenv.Load()

	jwtSecret := os.Getenv("JWT_SECRET")
	if jwtSecret == "" {
		logger.Error("JWT_SECRET is required (set it in .env or the environment)")
		os.Exit(1)
	}
	jwtAdapter := token.NewJWT([]byte(jwtSecret), jwtTTL())

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
	userSvc := user.NewService(userRepo, jwtAdapter, logger)
	teacherRoster := adapter.NewTeacherRosterAdapter(teacherSvc)
	commuteConfigRepo := db.NewCommuteConfigRepository(database)
	commuteSvc := commute.NewService(commuteConfigRepo, logger)
	commuteAdapter := adapter.NewCommuteAdapter(commuteSvc)
	branchCapacityAdapter := adapter.NewBranchCapacityAdapter(branchSvc)

	scorer := scheduling.NewWeightedScorer()
	clpEngine := scheduling.NewCLPEngine(bookingRepo, teacherRoster, scorer, commuteAdapter, branchCapacityAdapter, logger)

	schedulingSvc := scheduling.NewSchedulingService(bookingRepo, availRepo, clpEngine)

	availHandler := web.NewAvailabilityHandler(teacherSvc, logger)
	bookingHandler := web.NewBookingHandler(schedulingSvc, userSvc, logger)
	branchHandler := web.NewBranchHandler(branchSvc, logger)
	commuteHandler := web.NewCommuteHandler(commuteSvc, logger)
	userHandler := web.NewUserHandler(userSvc, logger)

	r := gin.Default()
	r.Use(requestIDMiddleware())
	r.Use(timeoutMiddleware(requestTimeout))
	r.Use(cors.New(corsConfig()))
	authRequired := web.AuthMiddleware(jwtAdapter)

	api := r.Group("/api")

	// Public (guest): login + browsing subjects and teacher availability.
	api.POST("/login", userHandler.Login)
	api.GET("/subjects", availHandler.GetSubjects)
	api.GET("/teachers", availHandler.GetTeachers)
	api.GET("/availability", availHandler.GetAllAvailability)

	api.POST("/register", authRequired, web.RequireRole(shared.RoleAdmin), userHandler.Register)
	api.POST("/availability", authRequired, web.RequireRole(shared.RoleAdmin), availHandler.SubmitWeeklyAvailability)

	// Bookings list is scoped by role inside the handler.
	api.GET("/bookings", authRequired, web.RequireRole(shared.RoleAdmin, shared.RoleStudent, shared.RoleParent), bookingHandler.ListBookings)

	// Everything else is admin-only.
	admin := api.Group("", authRequired, web.RequireRole(shared.RoleAdmin))
	admin.GET("/students", userHandler.ListStudents)
	admin.GET("/parents", userHandler.ListParents)
	admin.GET("/parents/:id/students", userHandler.GetParentStudents)
	admin.POST("/parents/:id/students", userHandler.AddParentStudent)
	admin.DELETE("/parents/:id/students/:studentId", userHandler.RemoveParentStudent)
	admin.POST("/teachers", availHandler.CreateTeacher)
	admin.PATCH("/teachers/:id/status", availHandler.UpdateTeacherStatus)
	admin.PATCH("/teachers/:id/gender", availHandler.UpdateTeacherGender)
	admin.GET("/branches", branchHandler.GetBranches)
	admin.POST("/branches", branchHandler.CreateBranch)
	admin.PATCH("/branches/:id/capacity", branchHandler.UpdateBranchCapacity)
	admin.GET("/commute", commuteHandler.GetCommute)
	admin.PATCH("/commute", commuteHandler.UpdateCommute)
	admin.POST("/bookings", bookingHandler.CreateBooking)
	admin.POST("/bookings/confirm", bookingHandler.ConfirmBooking)
	admin.DELETE("/bookings/:id", bookingHandler.CancelBooking)

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
