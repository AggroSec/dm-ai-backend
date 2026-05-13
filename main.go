package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/AggroSec/dm-ai-backend/internal/config"
	"github.com/AggroSec/dm-ai-backend/internal/database"
	"github.com/AggroSec/dm-ai-backend/internal/server"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}
	dbcon, err := database.ConnectDB(cfg.DBURL)
	if err != nil {
		log.Printf("connection to database failed, check dburl: %v\n", err)
	}

	db := database.New(dbcon)

	srv := server.New(cfg, db)
	handler := srv.RegisterRoutes()

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Starting AI DM SERVER on %s (env: %s)", addr, cfg.AppEnv)
	log.Fatal(http.ListenAndServe(addr, handler))
}
