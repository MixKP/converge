package main

import (
	"log"

	"github.com/RinZ5/converge/backend/internal/adapter/db"
)

func main() {
	database, err := db.InitDB()
	if err != nil {
		log.Fatal(err)
	}
	defer database.Close()
	if err := db.AutoMigrate(database); err != nil {
		log.Fatal(err)
	}
	if err := db.BootstrapAdmin(database); err != nil {
		log.Fatal(err)
	}
}
