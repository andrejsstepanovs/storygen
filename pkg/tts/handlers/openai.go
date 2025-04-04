package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type TTS struct {
	Model           string
	Voice           string
	Instructions    string
	Speed           float64
	APIKey          string
	Client          *http.Client
	MaxRetries      int
	RetryDelay      time.Duration
	RetryMultiplier float64
}

func (o *TTS) Convert(text, fileName string) error {
	var lastErr error
	retries := o.MaxRetries

	delay := o.RetryDelay
	for attempt := 0; attempt <= retries; attempt++ {
		if attempt > 0 {
			log.Printf("Retry attempt %d/%d after %v", attempt, retries, delay)
			time.Sleep(delay)
			delay = time.Duration(float64(delay) * o.RetryMultiplier)
		}

		err := o.doConvert(text, fileName)
		if err == nil {
			return nil // Success
		}

		lastErr = err

		// Only retry on server errors (5xx)
		if !isServerError(err) {
			return err // Don't retry client errors or other issues
		}

		log.Printf("Request failed (attempt %d/%d): %v", attempt+1, retries, err)
	}

	return fmt.Errorf("all %d conversion attempts failed, last error: %w", retries+1, lastErr)
}

// isServerError checks if the error is due to a server error (5xx)
func isServerError(err error) bool {
	// Check if the error is a statusCodeError or wraps a statusCodeError
	var statusErr *statusCodeError
	if errors.As(err, &statusErr) {
		// Server errors are 5xx status codes
		return statusErr.StatusCode >= 500 && statusErr.StatusCode < 600
	}

	// Fallback: check error message for status codes
	// This is less reliable but helps with errors that weren't properly typed
	errStr := err.Error()
	for _, code := range []string{"500", "502", "503", "504"} {
		if len(errStr) >= len(code) && (errStr[:len(code)] == code ||
			(len(errStr) > len(code)+2 && errStr[len(errStr)-len(code)-2:len(errStr)-2] == code)) {
			return true
		}
	}

	return false
}

type statusCodeError struct {
	StatusCode int
	Message    string
}

func (e *statusCodeError) Error() string {
	return fmt.Sprintf("API request failed with status %d: %s", e.StatusCode, e.Message)
}

func (o *TTS) doConvert(text, fileName string) error {
	reqBodyMap := map[string]interface{}{
		"model": o.Model,
		"input": text,
		"voice": o.Voice,
	}
	if o.Instructions != "" {
		reqBodyMap["instructions"] = o.Instructions
	}
	if o.Speed > 0 {
		reqBodyMap["speed"] = o.Speed
	}

	reqBodyBytes, err := json.Marshal(reqBodyMap)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequest("POST", "https://api.openai.com/v1/audio/speech", bytes.NewBuffer(reqBodyBytes))
	if err != nil {
		return fmt.Errorf("failed to create HTTP request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+o.APIKey)

	resp, err := o.Client.Do(req)
	if err != nil {
		return fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		log.Printf("API request failed with status: %s (%d)\n", resp.Status, resp.StatusCode)

		errorBodyBytes, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			log.Printf("Failed to read error response body: %v\n", readErr)
			return fmt.Errorf("API request failed with status %d, and could not read error response body: %w",
				resp.StatusCode, readErr)
		}

		log.Printf("API error response body: %s\n", string(errorBodyBytes))

		// Return a status code error that can be properly checked
		return &statusCodeError{
			StatusCode: resp.StatusCode,
			Message:    string(errorBodyBytes),
		}
	}

	// Create a new file for each attempt to avoid partial writes
	audioFile, err := os.Create(fileName)
	if err != nil {
		return fmt.Errorf("failed to create audio file %q: %w", fileName, err)
	}

	var closeFileErr error
	defer func() {
		err := audioFile.Close()
		if closeFileErr == nil && err != nil {
			closeFileErr = fmt.Errorf("failed to close audio file %q: %w", fileName, err)
		}
	}()
	log.Println("Created file")

	bytesCopied, err := io.Copy(audioFile, resp.Body)
	if err != nil {
		_ = os.Remove(fileName)
		return fmt.Errorf("failed to write audio data to file %q: %w", fileName, err)
	}
	log.Printf("Copied %d bytes\n", bytesCopied)

	if closeFileErr != nil {
		_ = os.Remove(fileName)
		return closeFileErr
	}

	return nil
}
