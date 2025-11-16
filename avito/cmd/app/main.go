package main

import (
	"avito/internal/db"
	httphandler "avito/internal/http"
	"context"
	"log"
	"net/http"
	"time"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	database := db.MustConnect(ctx)
	if err := db.ApplyMigrations(ctx, database); err != nil {
		log.Fatalf("apply migrations: %v", err)
	}

	router := httphandler.NewRouter(database)

	addr := ":8080"
	log.Printf("listening on %s\n", addr)
	if err := http.ListenAndServe(addr, router); err != nil {
		log.Fatal(err)
	}
}
