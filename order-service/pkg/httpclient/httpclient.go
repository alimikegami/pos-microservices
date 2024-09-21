package httpclient

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"time"
)

// HttpRequest is a struct to hold request parameters
type HttpRequest struct {
	URL     string
	Method  string
	Body    []byte
	Headers map[string]string
}

// SendRequest sends an HTTP request based on the given HttpRequest struct
func SendRequest(req HttpRequest) (int, []byte, error) {
	// Create the HTTP request
	request, err := http.NewRequest(req.Method, req.URL, bytes.NewBuffer(req.Body))
	if err != nil {
		return 0, nil, fmt.Errorf("failed to create request: %v", err)
	}

	// Add headers to the request
	for key, value := range req.Headers {
		request.Header.Set(key, value)
	}

	// Create an HTTP client with a timeout
	client := &http.Client{Timeout: 10 * time.Second}

	// Send the request
	response, err := client.Do(request)
	if err != nil {
		return 0, nil, fmt.Errorf("request failed: %v", err)
	}
	defer response.Body.Close()

	// Read the response body
	body, err := io.ReadAll(response.Body)
	if err != nil {
		return response.StatusCode, nil, fmt.Errorf("failed to read response body: %v", err)
	}

	return response.StatusCode, body, nil
}
