package utils

import (
	"log"
	"time"
)

// SleepForSeconds pauses the current goroutine for the given number of seconds.
func SleepForSeconds(seconds time.Duration) {
	time.Sleep(time.Second * seconds)
}



// ParseStrConvertToEpochNs converts a time string in RFC3339 format to epoch time in nanoseconds.
func ParseStrConvertToEpochNs(timeStr string) int64 {
	t, err := time.Parse(time.RFC3339, timeStr)

	if err != nil {
		log.Printf("Error parsing time: %v", err)
		log.Printf("Time string: %v", timeStr)
		return 0
	}

	return t.UnixNano()
}