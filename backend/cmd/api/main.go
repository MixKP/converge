package main

import (
	"log"

	"github.com/RinZ5/converge/backend/internal/adapter/db"
	"github.com/RinZ5/converge/backend/internal/adapter/web"

	"github.com/gin-gonic/gin"
)

func main() {
	if err := db.InitDB(); err != nil {
		log.Fatalf("Database init failed: %v", err)
	}
	defer db.CloseDB()

	if err := db.AutoMigrate(); err != nil {
		log.Fatalf("Migration failed: %v", err)
	}

	repo := &db.PostgresRepo{}

	handler := web.NewAvailabilityHandler(repo)

	r := gin.Default()
	api := r.Group("/api")
	{
		api.GET("/teachers", handler.GetTeachers)
		api.POST("/availability", handler.SubmitWeeklyAvailability)
	}

	log.Println("Server starting on :8080")
	if err := r.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
