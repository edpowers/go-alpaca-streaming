package main

import (
	"go-alpaca-streaming/pkg/websocket_conn"
	"log"
	"os"
	"sync"
)

func main() {

	/*
		err := godotenv.Load("./cmd/.env")
		if err != nil {
			fmt.Println("Error loading .env file:", err)
			os.Exit(1)
		}
	*/

	if os.Getenv("APCA_API_KEY_ID") == "" {
		log.Fatal("APCA_API_KEY_ID is not set")
	}
	if os.Getenv("APCA_API_SECRET_KEY") == "" {
		log.Fatal("APCA_API_SECRET_KEY is not set")
	}

	var wg sync.WaitGroup

	// Increase the counter
	wg.Add(1)

	log.Println("Starting the WebSocket client.")

	go websocket_conn.RunWebSocketClient(&wg)

	// Wait until all goroutines call Done
	wg.Wait()

	log.Println("WebSocket client has stopped.")

}
