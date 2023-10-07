package websocket_conn

import (
	"context"
	"os"

	// "regexp"
	"strings"
	"sync"

	"encoding/json"
	"log"
	"net/http"
	"net/url"

	// "time"
	"crypto/tls"

	// "strings"
	"fmt"
	author_symbols "go-alpaca-streaming/pkg/symbols"
	"go-alpaca-streaming/pkg/telegraf"
	"go-alpaca-streaming/pkg/utils"

	"github.com/gorilla/websocket"
)

type TradeData struct {
	Symbol string
	Price  float64
	Size   int
	X      string
	C      string
	Time   int64
	I      int
	Z      string
	// ... other fields
}

type GenericMessage struct {
	T   string `json:"T"`
	Msg string `json:"msg"`
}

type WebSocketAuthenticator struct {
	conn *websocket.Conn
}

func (auth *WebSocketAuthenticator) WaitForAuthentication() (bool, error) {
	var authResponse []GenericMessage

	_, authMessage, err := auth.conn.ReadMessage()
	if err != nil {
		return false, fmt.Errorf("Failed to read auth acknowledgment: %v %v", authMessage, err)
	}

	err = json.Unmarshal(authMessage, &authResponse)
	if err != nil {
		return false, fmt.Errorf("Error unmarshalling auth acknowledgment: %v", err)
	}

	for _, msg := range authResponse {
		if msg.T == "success" {
			return true, nil
		}
	}

	return false, fmt.Errorf("Authentication failed")
}

func authenticate(conn *websocket.Conn) bool {
	authenticator := WebSocketAuthenticator{conn: conn}
	authSuccess, err := authenticator.WaitForAuthentication()
	if err != nil || !authSuccess {
		log.Fatalf("Authentication failed: %v", err)
		return false
	}
	return true
}

func establishConnection(dialer websocket.Dialer, u url.URL) (*websocket.Conn, *http.Response, error) {
	headers := http.Header{}

	headers.Add("APCA-API-KEY-ID", os.Getenv("APCA_API_KEY_ID"))
	headers.Add("APCA-API-SECRET-KEY", os.Getenv("APCA_API_SECRET_KEY"))
	return dialer.Dial(u.String(), headers)
}

func RunWebSocketClient(wg *sync.WaitGroup) {
	// Decrease the counter when the goroutine completes
	wg.Add(1) // Increment the counter
	defer wg.Done()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Prepare WebSocket URL
	u := url.URL{Scheme: "wss", Host: "stream.data.alpaca.markets", Path: "/v2/sip"}

	// Custom Gorilla Dialer with TLS verification disabled
	dialer := websocket.Dialer{
		TLSClientConfig:   &tls.Config{InsecureSkipVerify: true},
		EnableCompression: true,
	}

	conn, resp, err := establishConnection(dialer, u)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
		return
	}
	defer conn.Close()

	if !authenticate(conn) {
		log.Fatalf("Authentication failed. Exiting. %v", resp)
		return
	}

	/*
		// Prepare and send authentication data
		authData := map[string]interface{}{
			"action": "auth",
			// "key":    os.Getenv("ALPACA_API_KEY"),
			// "secret": os.Getenv("ALPACA_API_SECRET"),
			"key":    "PKQSHIK7LOCNHMK36ZVP",
			"secret": "qSBkZcKND0mJhy0E3ZH0evnUIKJRDdKlPBtwzf7q",
		}
	*/

	authenticator := WebSocketAuthenticator{conn: conn}
	authSuccess, err := authenticator.WaitForAuthentication()
	if err != nil {
		log.Fatalf("%v. Exiting.", err)
		return
	}

	if !authSuccess {
		log.Fatalf("Authentication failed. Exiting.")
		return
	}

	// Retrieve the symbols
	SymbolsToUse, err := author_symbols.GetAuthorSymbols()
	if err != nil {
		log.Println("Error retrieving symbols:", err)

		// Fallback mechanism
		SymbolsToUse = author_symbols.GetLocalSymbols()
		if len(SymbolsToUse) == 0 {
			log.Println("No local symbols available for fallback. Exiting.")
			return
		}
		log.Println("Using local symbols for fallback:", SymbolsToUse)
	}

	// Fixed type mismatch
	subscriptionMessage := map[string]interface{}{
		"action": "subscribe",
		"trades": SymbolsToUse, // Assuming the key is 'trades' and the value is an array of strings
	}

	// Send subscription message
	err = conn.WriteJSON(subscriptionMessage)
	if err != nil {
		log.Fatalf("Failed to subscribe: %v", err)
		return
	}

	go func(ctx context.Context, conn *websocket.Conn) {
		defer wg.Done() // Decrement counter when goroutine completes
		// Initialize a flag to check if "success" message is received
		successReceived := false
		for {
			select {
			case <-ctx.Done():
				log.Println("Context done, stopping.")
				return
			default:
				// log.Println("Reading message.")
				_, message, err := conn.ReadMessage()
				if err != nil {
					log.Printf("Error reading raw message: %v", err)
					continue
				}

				// receivedBytes := []byte(message)
				//receivedString := string(receivedBytes)
				// log.Println("Received message:", receivedString)

				// If "success" message is already received, process as RawTrade
				if successReceived {
					// log.Println("Processing as RawTrade.")
					var trades []utils.RawTrade
					if err := json.Unmarshal(message, &trades); err != nil {
						log.Printf("Error unmarshalling array of trades: %v", err)
						continue
					}

					for _, trade := range trades {
						// For debugging:
						// log.Println("Handling trade: %v %v %v", trade, trades, message)
						handleWebSocket(trade)
					}
					continue
				} else {
					// log.Println("Processing as GenericMessage. %v", message)
					// Try to unmarshal into a GenericMessage to look for "success"
					var messages []GenericMessage
					if err := json.Unmarshal(message, &messages); err != nil {
						log.Printf("Error unmarshalling initial message: %v", err)
						log.Printf("Message: %s", message)
						continue
					}

					for _, msg := range messages {

						if msg.T == "success" || msg.T == "subscription" {
							log.Println("Received success message.")
							// Update the flag
							successReceived = true
						} else {
							log.Printf("Received unknown or unhandled message type: %v", msg.T)
						}
					}
				}
			}
		}
	}(ctx, conn)

	wg.Wait() // Wait for all goroutines to finish
}

func handleWebSocket(raw utils.RawTrade) {
	// Convert Alpaca Trade to TradeData
	convertedData := ConvertToTradeData(raw)
	// Convert TradeData to LineProtocol
	lineProtocol := convertedData.FormatTradeLineProtocol()

	// Validate line protocol
	if !telegraf.IsValidLineProtocol(lineProtocol) {
		log.Println("Invalid line protocol:", lineProtocol)
		return
	}

	// Send to Telegraf
	if err := telegraf.SendToTelegraf([]string{lineProtocol}); err != nil {
		log.Println("Error sending to Telegraf:", err)
		log.Println("Failed trade data:", lineProtocol)
		return
	}
}

// ConvertToTradeData converts Alpaca StreamTrade to your TradeData type
func ConvertToTradeData(raw utils.RawTrade) *TradeData {
	// Assume we have a function to convert data.T to epoch_ns
	return &TradeData{
		Symbol: raw.Symbol,
		Price:  raw.Price,
		Size:   raw.Size,
		X:      raw.X,
		C:      utils.MakeTradeCondition(raw),
		Time:   utils.ParseStrConvertToEpochNs(raw.Time),
		I:      raw.I,
		Z:      raw.Z,
	}
}

func removeExtraSpaces(input string) string {
	insideQuotes := false
	var result strings.Builder

	for _, char := range input {
		if char == '"' {
			insideQuotes = !insideQuotes
		}
		if char != ' ' || insideQuotes {
			result.WriteRune(char)
		}
	}
	return result.String()
}

func (data *TradeData) FormatTradeLineProtocol() string {
	// Measurement
	measurement := "alpaca_equities_streaming_trades"

	// Tags
	tags := fmt.Sprintf(`symbol=%s,conditions_str="%s",exchange=%s`, data.Symbol, data.C, data.X)
	tags = removeExtraSpaces(tags)

	// Fields
	fields := fmt.Sprintf(`price=%f,size=%d,trade_id=%d,tape="%s"`, data.Price, data.Size, data.I, data.Z)
	fields = removeExtraSpaces(fields)

	return fmt.Sprintf("%s,%s %s %d", measurement, tags, fields, data.Time)
}
