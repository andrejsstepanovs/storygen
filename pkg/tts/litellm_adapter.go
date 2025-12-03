package tts

import (
	"fmt"
	"log"
	"time"
)

// LiteLLMAdapter adapts the AI client to the TTSConverter interface
type LiteLLMAdapter struct {
	TextToSpeechFunc func(text, voice, instructions string, speed float64) (string, error)
	MaxRetries       int
	RetryDelay       time.Duration
	RetryMultiplier  float64
}

// Convert implements the TTSConverter interface with retry logic
func (a *LiteLLMAdapter) Convert(text, voice, instructions string, speed float64) (string, error) {
	var lastErr error
	retries := a.MaxRetries

	delay := a.RetryDelay
	for attempt := 0; attempt <= retries; attempt++ {
		if attempt > 0 {
			log.Printf("Retry attempt %d/%d after %v", attempt, retries, delay)
			time.Sleep(delay)
			delay = time.Duration(float64(delay) * a.RetryMultiplier)
		}

		filePath, err := a.TextToSpeechFunc(text, voice, instructions, speed)
		if err == nil {
			return filePath, nil // Success
		}

		lastErr = err
		log.Printf("Request failed (attempt %d/%d): %v", attempt+1, retries, err)
	}

	return "", fmt.Errorf("all %d conversion attempts failed, last error: %w", retries+1, lastErr)
}
