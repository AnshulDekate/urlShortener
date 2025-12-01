package main

import (
	"database/sql"
	"fmt"
	"log"
	"os" 
	"time"
	"github.com/gin-gonic/gin"

	"github.com/pressly/goose/v3"
	_ "github.com/jackc/pgx/v5/stdlib" 

	"github.com/AnshulDekate/urlShortener/repository"
	"github.com/AnshulDekate/urlShortener/service"
	"github.com/AnshulDekate/urlShortener/handler"
	"github.com/AnshulDekate/urlShortener/middleware"
)

func mustGetEnv(key string) string {
	value := os.Getenv(key)
	if value == "" {
		log.Fatalf("Fatal: Required environment variable %s is not set. Application cannot start.", key)
	}
	return value
}

func waitForDB(db *sql.DB, maxAttempts int, delay time.Duration) error {
	for i := 0; i < maxAttempts; i++ {
		if err := db.Ping(); err == nil {
			return nil
		}
		log.Printf("Database not ready, waiting %s... (Attempt %d/%d)", delay, i+1, maxAttempts)
		time.Sleep(delay)
	}
	return fmt.Errorf("database connection timed out")
}

func runMigrations(db *sql.DB) error {
	log.Println("Running database migrations...")

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set Goose dialect: %w", err)
	}

	migrationsPath := mustGetEnv("MIGRATIONS_PATH")
	if err := goose.Up(db, migrationsPath); err != nil {
		return fmt.Errorf("migration failed: %w", err)
	}

	log.Println("Migrations completed successfully.")
	return nil
}

func main() {
	dbHost := mustGetEnv("DB_HOST")
	dbPort := mustGetEnv("DB_PORT")
	dbUser := mustGetEnv("DB_USER")
	dbPass := mustGetEnv("DB_PASS")
	dbName := mustGetEnv("DB_NAME")
	appPort := mustGetEnv("APP_PORT")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPass, dbName)

	db, err := sql.Open("pgx", connStr)
	if err != nil {
		log.Fatalf("Error opening database connection: %v", err)
	}
	defer db.Close()

	if err := waitForDB(db, 10, 1*time.Second); err != nil {
		log.Fatalf("Fatal: Database not available: %v", err)
	}
	if err := runMigrations(db); err != nil {
		log.Fatalf("Fatal: Failed to run migrations: %v", err)
	}

	listenAddr := fmt.Sprintf(":%s", appPort)
	shortURLDomain := fmt.Sprintf("http://localhost%s/", listenAddr)

	repo := &repository.Repository{DB: db}
	svc := &service.Service{Repo: repo}
	h := handler.NewGinHandler(svc, shortURLDomain) 

	log.Println("Setting up HTTP handlers with Gin...")
	
	r := gin.New()
	r.Use(gin.Recovery())      
	r.Use(gin.Logger())       
	r.Use(middleware.RateLimiterMiddleware()) 

	r.POST("/shorten", h.Shorten)
	r.GET("/healthcheck", h.HealthCheck)
	r.GET("/:code", h.Redirect) 
	r.GET("/urls", h.ListURLs)

	log.Printf("Gin server starting on %s...", listenAddr)
	if err := r.Run(listenAddr); err != nil {
		log.Fatalf("Gin server failed: %v", err)
	}
}
