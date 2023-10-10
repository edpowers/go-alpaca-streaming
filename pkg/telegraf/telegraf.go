package telegraf

import (
	"errors"
	"fmt"
	"log"
	"net"
	"strings"
)

var telegrafHost string = "telegraf"
var telegrafPort string = "8094"

var sharedConn net.Conn

func SetupTelegrafConnection() {
	var err error
	sharedConn, err = net.Dial("tcp", fmt.Sprintf("%s:%s", telegrafHost, telegrafPort))
	if err != nil {
		log.Fatalf("Failed to establish connection to Telegraf: %v", err)
	}
}

func SendToTelegraf(processedData []string) error {
	if sharedConn == nil {
		log.Println("Telegraf connection is not established.")
		return errors.New("Telegraf connection is not established")
	}

	for _, lineData := range processedData {
		_, err := fmt.Fprintf(sharedConn, "%s\n", lineData)
		if err != nil {
			log.Printf("Failed to write data to Telegraf: %s, Error: %v", lineData, err)
			return err
		}
	}

	return nil
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

func _SendToTelegraf(processedData []string) error {
	conn, err := net.Dial("tcp", fmt.Sprintf("%s:%s", telegrafHost, telegrafPort))
	if err != nil {
		log.Println("Failed to establish connection to Telegraf:", err)
		return err
	}
	defer conn.Close()

	for _, lineData := range processedData {
		_, err := fmt.Fprintf(conn, "%s\n", lineData)
		if err != nil {
			log.Printf("Failed to write data to Telegraf: %s, Error: %v", lineData, err)
			return err
		}
	}

	return nil
}
