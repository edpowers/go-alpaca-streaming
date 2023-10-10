package main

import (
	"errors"
	"go-alpaca-streaming/pkg/websocket_conn"
	"log"
	"os"
	"sync"
	"time"

	"github.com/getsentry/sentry-go"
)

func main() {
	/*
		err := godotenv.Load("./cmd/.env")
		if err != nil {
			fmt.Println("Error loading .env file:", err)
			os.Exit(1)
		}
	*/
	// SENTRY DSN - ATTACHES TO GLITCHTIP
	err := sentry.Init(sentry.ClientOptions{
		Dsn: os.Getenv("GO_ALPACA_STREAMING_SENTRY_DSN"),
	})
	if err != nil {
		log.Fatalf("sentry.Init: %s", err)
	}
	sentry.CaptureException(errors.New("my error"))
	// Since sentry emits events in the background we need to make sure
	// they are sent before we shut down
	sentry.Flush(time.Second * 5)

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
