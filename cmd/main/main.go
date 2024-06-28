package main

//swagger
//validator DONE
//errors
//status codes DONE
//docker
//pgx DONE

import (
	"context"
	"log"
	"os"

	"github.com/ursuldaniel/bank-api/internal/server"
	"github.com/ursuldaniel/bank-api/internal/storage"
)

func main() {
	// err := godotenv.Load(".env")
	// if err != nil {
	// 	log.Fatal(err)
	// }

	listenAddr := os.Getenv("LISTEN_ADDR")
	if listenAddr == "" {
		log.Fatal("missed server address")
	}

	log.Println(listenAddr)

	storage, err := storage.NewPostgresStorage(context.TODO(), os.Getenv("CONN_STR"))
	if err != nil {
		log.Fatal(err)
	}

	server := server.NewServer(listenAddr, storage)
	log.Fatal(server.Run())
}
