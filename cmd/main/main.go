package main

//swagger
//validator
//errors
//status codes
//docker
//pgx

import (
	"context"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/ursuldaniel/bank-api/internal/server"
	"github.com/ursuldaniel/bank-api/internal/storage"
)

func main() {
	err := godotenv.Load(".env")
	if err != nil {
		log.Fatal(err)
	}

	listenAddr := os.Getenv("listenAddr")
	if listenAddr == "" {
		log.Fatal("missed server address")
	}

	storage, err := storage.NewPostgresStorage(context.TODO(), os.Getenv("connStr"))
	if err != nil {
		log.Fatal(err)
	}

	server := server.NewServer(listenAddr, storage)
	log.Fatal(server.Run())
}
