package main

import (
	"fmt"
	"log"
	"net/http"

	"github.com/AggroSec/dm-ai-backend/internal/config"
	"github.com/AggroSec/dm-ai-backend/internal/server"
)

func main() {
	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	srv := server.New(cfg)
	handler := srv.RegisterRoutes()

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Starting AI DM SERVER on %s (env: %s)", addr, cfg.AppEnv)
	log.Fatal(http.ListenAndServe(addr, handler))
}
