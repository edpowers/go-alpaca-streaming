package utils

import (
	"strings"
)



type RawTrade struct {
	Type string    `json:"T"`  // Message type
	I int          `json:"i"`  // Trade id
	Symbol string  `json:"S"`  // Symbol
	X string       `json:"x"`  // Exchange code where trade occured
	Price float64  `json:"p"`  // Trade Price
	Size int       `json:"s"`  // Trade Size
	Time string    `json:"t"`  // Timestamp
	C []string     `json:"c"`  // Conditions
	Z string       `json:"z"`  // Tape
}

var raw RawTrade

// MakeTradeCondition formats the trade condition component of a RawTrade object

func MakeTradeCondition(raw RawTrade) string {
	// Join the trade condition slice into a string, separated by commas
	tradeCondition := strings.Join(raw.C, ",")

	// Remove prefix comma, if any, and replace other commas
	tradeCondition = strings.TrimPrefix(tradeCondition, ",")
	tradeCondition = strings.ReplaceAll(tradeCondition, ",", "")

	// Check if the resulting string is empty
	if tradeCondition == "" {
		tradeCondition = "N"
	}

	return tradeCondition
}


