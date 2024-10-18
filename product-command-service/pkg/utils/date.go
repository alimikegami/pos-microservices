package utils

import (
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
