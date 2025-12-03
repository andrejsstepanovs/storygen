package ai

import (
	"context"
	"log"
	"net/url"
	"strings"
	"time"

	"github.com/andrejsstepanovs/go-litellm/client"
	"github.com/andrejsstepanovs/go-litellm/conf/connections/litellm"
	"github.com/andrejsstepanovs/go-litellm/models"
	"github.com/andrejsstepanovs/go-litellm/request"
	"github.com/spf13/viper"
)

type AI struct {
	client   *client.Litellm
	ctx      context.Context
	audience string
	model    string
}

func NewAI(audience string) (*AI, error) {
	model := viper.GetString("STORYGEN_MODEL")
	if model == "" {
		model = "claude-3-7-sonnet-latest"
	}

	litellmHost := viper.GetString("LITELLM_HOST")
	if litellmHost == "" {
		litellmHost = "http://localhost:4000"
	}

	apiKey := viper.GetString("LITELLM_API_KEY")
	if apiKey == "" {
		apiKey = "sk-1234"
	}

	// Parse base URL for LiteLLM service
	baseURL, err := url.Parse(litellmHost)
	if err != nil {
		log.Fatalf("Failed to parse LiteLLM URL: %v", err)
		return nil, err
	}

	// Configure connection with different timeout targets
	conn := litellm.Connection{
		URL: *baseURL,
		Targets: litellm.Targets{
			System: litellm.Target{
				Timeout:          time.Second * 30,
				RetryInterval:    time.Second * 2,
				RetryMaxAttempts: 3,
				RetryBackoffRate: 1.5,
				MaxRetry:         3,
			},
			LLM: litellm.Target{
				Timeout:          time.Minute * 5,
				RetryInterval:    time.Second * 2,
				RetryMaxAttempts: 3,
				RetryBackoffRate: 1.5,
				MaxRetry:         3,
			},
			MCP: litellm.Target{
				Timeout:          time.Minute * 5,
				RetryInterval:    time.Second * 2,
				RetryMaxAttempts: 3,
				RetryBackoffRate: 1.5,
				MaxRetry:         3,
			},
		},
	}

	// Validate connection configuration
	if err := conn.Validate(); err != nil {
		log.Fatalf("Connection validation failed: %v", err)
		return nil, err
	}

	// Create client configuration
	cfg := client.Config{
		APIKey:      apiKey,
		Temperature: 0.7,
	}

	// Initialize client
	litellmClient, err := client.New(cfg, conn)
	if err != nil {
		log.Fatalf("Failed to create LiteLLM client: %v", err)
		return nil, err
	}

	log.Printf("Using LiteLLM with model %q\n", model)

	return &AI{
		client:   litellmClient,
		ctx:      context.Background(),
		audience: audience,
		model:    model,
	}, nil
}

// TextToSpeech converts text to speech using the configured TTS model
// Returns the path to the generated audio file
func (a *AI) TextToSpeech(text, voice, instructions string, speed float64) (string, error) {
	ttsModel := viper.GetString("STORYGEN_TTS_MODEL")
	if ttsModel == "" {
		ttsModel = "tts-openai"
	}

	speechRequest := request.Speech{
		Model: models.ModelID(ttsModel),
		Input: text,
		Voice: voice,
	}

	if strings.Contains(ttsModel, "openai") {
		speechRequest.Instructions = instructions
		speechRequest.Speed = speed
		speechRequest.ResponseFormat = "mp3"
	}

	resp, err := a.client.TextToSpeech(a.ctx, speechRequest)
	if err != nil {
		return "", err
	}

	return resp.Full, nil
}
