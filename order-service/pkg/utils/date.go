package utils

import (
	"fmt"
	"time"
)

func ConvertDateTimeToHumanReadableFormat(datetime int64) (string, error) {
	// Parse the input string into a time.Time object
	t := time.Unix(datetime, 0)
	// Format the time in the desired output format location := time.FixedZone("WIB", 7*60*60)
	location := time.FixedZone("WIB", 7*60*60)
	wibTime := t.In(location)
	outputFormat := "02 January 2006, 15:04 WIB"
	formattedDateTime := wibTime.Format(outputFormat)

	return formattedDateTime, nil
}

func ConvertDateTimeWibToUnixTimestamp(wibTime string) (int64, error) {
	// Define the WIB time zone
	wibLocation, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		return 0, fmt.Errorf("error loading WIB time zone: %v", err)
	}

	// Parse the input time string
	t, err := time.ParseInLocation("2006-01-02 15:04:05", wibTime, wibLocation)
	if err != nil {
		return 0, fmt.Errorf("error parsing time: %v", err)
	}

	// Convert to Unix timestamp
	return t.Unix(), nil
}
