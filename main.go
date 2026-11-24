package main

import (
	"log"

	"app/internal/config"
	"app/internal/database"
	"app/internal/features/ai"
	"app/internal/server"

	"github.com/gin-gonic/gin"
)

// Entry point untuk build CI/CD (go build .). Setara dengan cmd/server/main.go.
func main() {
	cfg := config.Load()

	db, err := database.ConnectPostgres(cfg.DSN())
	if err != nil {
		log.Fatalf("gagal terhubung ke PostgreSQL: %v", err)
	}
	if err := database.Migrate(db); err != nil {
		log.Fatalf("gagal migrasi tabel: %v", err)
	}
	if err := database.Seed(db, cfg.AdminUsername, cfg.AdminPassword); err != nil {
		log.Fatalf("gagal seeding admin: %v", err)
	}

	aiSvc := ai.NewService(
		&ai.SearchClient{Endpoint: cfg.SearchEngineEndpoint},
		&ai.ChatClient{Endpoint: cfg.OpenAIEndpoint, Key: cfg.OpenAIKey, Model: cfg.OpenAIModel},
	)

	if cfg.AppEnv == "production" {
		gin.SetMode(gin.ReleaseMode)
	}
	r := server.New(cfg, db, aiSvc)
	if err := r.Run(":" + cfg.AppPort); err != nil {
		log.Fatalf("server berhenti: %v", err)
	}
}
