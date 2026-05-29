package main

import (
	"log"

	"github.com/RinZ5/converge/backend/internal/adapter/db"
)

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
		log.Fatalf("Migration failed: %v", err)
	}

	if err := db.Seed(database); err != nil {
		log.Fatalf("Seed failed: %v", err)
	}

	log.Println("Database seeded successfully")
}
