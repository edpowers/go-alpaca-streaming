package telegraf

import (
	"errors"
	"fmt"
	"log"
	"math"
	"net"
	"strings"
	"time"
)

var telegrafHost string = "telegraf"
var telegrafPort string = "8094"
var maxRetries int = 5
var initialBackoff time.Duration = 100 * time.Millisecond

var sharedConn net.Conn

func SetupTelegrafConnection() {
	var err error

	for attempt := 0; attempt < maxRetries; attempt++ {
		sharedConn, err = net.Dial("tcp", fmt.Sprintf("%s:%s", telegrafHost, telegrafPort))
		if err == nil {
			return // Connection successful
		}

		backoff := time.Duration(math.Pow(2, float64(attempt))) * initialBackoff
		log.Printf("Failed to establish connection to Telegraf (attempt %d/%d): %v. Retrying in %v...",
			attempt+1, maxRetries, err, backoff)
		time.Sleep(backoff)
	}

	// If we get here, all retries failed
	log.Fatalf("Failed to establish connection to Telegraf after %d attempts: %v", maxRetries, err)
}

func SendToTelegraf(processedData []string) error {
	if sharedConn == nil {
		log.Println("Telegraf connection is not established.")
		return errors.New("Telegraf connection is not established")
	}

	for _, lineData := range processedData {
		var err error
		for attempt := 0; attempt < maxRetries; attempt++ {
			_, err = fmt.Fprintf(sharedConn, "%s\n", lineData)
			if err == nil {
				break // Write successful
			}

			// If we get a network error, the connection might be broken
			if netErr, ok := err.(net.Error); ok {
				log.Printf("Network error writing to Telegraf (attempt %d/%d): %v",
					attempt+1, maxRetries, netErr)

				// Try to re-establish connection
				closeErr := sharedConn.Close()
				if closeErr != nil {
					log.Printf("Error closing broken connection: %v", closeErr)
				}

				// Exponential backoff before reconnecting
				backoff := time.Duration(math.Pow(2, float64(attempt))) * initialBackoff
				time.Sleep(backoff)

				reconnectErr := reconnectTelegraf()
				if reconnectErr != nil {
					// If reconnection fails, return the original error
					return err
				}
			} else {
				// Not a network error, just retry the write
				backoff := time.Duration(math.Pow(2, float64(attempt))) * initialBackoff
				log.Printf("Failed to write data to Telegraf (attempt %d/%d): %v. Retrying in %v...",
					attempt+1, maxRetries, err, backoff)
				time.Sleep(backoff)
			}
		}

		if err != nil {
			log.Printf("Failed to write data to Telegraf after %d attempts: %s, Error: %v",
				maxRetries, lineData, err)
			return err
		}
	}

	return nil
}

// reconnectTelegraf attempts to re-establish the connection to Telegraf
func reconnectTelegraf() error {
	var err error

	for attempt := 0; attempt < maxRetries; attempt++ {
		sharedConn, err = net.Dial("tcp", fmt.Sprintf("%s:%s", telegrafHost, telegrafPort))
		if err == nil {
			log.Println("Successfully reconnected to Telegraf")
			return nil
		}

		backoff := time.Duration(math.Pow(2, float64(attempt))) * initialBackoff
		log.Printf("Failed to reconnect to Telegraf (attempt %d/%d): %v. Retrying in %v...",
			attempt+1, maxRetries, err, backoff)
		time.Sleep(backoff)
	}

	return fmt.Errorf("failed to reconnect to Telegraf after %d attempts", maxRetries)
}

func CloseTelegrafConnection() {
	if sharedConn != nil {
		sharedConn.Close()
	}
}

// SendToTelegraf sends processed data to the Telegraf instance

// IsValidLineProtocol validates if the given string conforms to InfluxDB Line Protocol.
func IsValidLineProtocol(line string) bool {
	// A very basic check to see if the line contains essential components:
	// measurement, field-key, field-value, and timestamp.
	// For more rigorous validation, you might want to use regex or more advanced parsing.
	if strings.Count(line, ",") > 0 && strings.Count(line, "=") > 0 && strings.Count(line, " ") > 0 {
		return true
	}
	return false
}
