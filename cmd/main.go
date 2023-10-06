package main

import (
    "log"
    "sync"
    "go-alpaca-streaming/pkg/websocket_conn"

    )

func main() {

    var wg sync.WaitGroup

    // Increase the counter
	wg.Add(1)

    log.Println("Starting the WebSocket client.")

    go websocket_conn.RunWebSocketClient(&wg)

    // Wait until all goroutines call Done
	wg.Wait()

    log.Println("WebSocket client has stopped.")

}