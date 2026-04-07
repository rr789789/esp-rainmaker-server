package main

import (
	"flag"
	"fmt"
	"log"

	"esp-rainmaker-server/internal/api"
	"esp-rainmaker-server/internal/config"
	"esp-rainmaker-server/internal/store"

	_ "modernc.org/sqlite"
)

func main() {
	configPath := flag.String("config", "", "Path to config file")
	flag.Parse()

	// Load config
	if err := config.Load(*configPath); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Init database
	if err := store.InitDB(config.AppConfig.Database.Path); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer store.DB.Close()

	// Setup router
	r := api.SetupRouter()

	addr := fmt.Sprintf("%s:%d", config.AppConfig.Server.Host, config.AppConfig.Server.Port)
	log.Printf("ESP RainMaker Server starting on %s", addr)
	log.Printf("Admin panel: http://%s/admin/", addr)
	log.Printf("API endpoint: http://%s/v1/", addr)

	if err := r.Run(addr); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
