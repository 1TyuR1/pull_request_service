package db

import (
	"context"
	"database/sql"
	"log"
	"os"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func MustConnect(ctx context.Context) *sql.DB {
	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		log.Fatal("DATABASE_URL is not set")
	}

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("failed to open db: %v", err)
	}

	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(time.Hour)

	const maxAttempts = 30
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		pingCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
		err = db.PingContext(pingCtx)
		cancel()

		if err == nil {
			log.Println("database connection established")
			return db
		}

		log.Printf("db not ready yet (attempt %d/%d): %v", attempt, maxAttempts, err)
		time.Sleep(1 * time.Second)
	}

	log.Fatalf("failed to ping db after %d attempts: %v", maxAttempts, err)
	return nil
}
