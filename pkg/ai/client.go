package ai

import (
	"log"
	"strings"
	"time"

	"github.com/spf13/viper"
	"github.com/teilomillet/gollm"
	"github.com/teilomillet/gollm/llm"
	"github.com/teilomillet/gollm/providers"
	"github.com/teilomillet/gollm/utils"
)

type AI struct {
	client   llm.LLM
	audience string
}

func NewAI(audience string) (*AI, error) {
	provider := viper.GetString("STORYGEN_PROVIDER")
	model := viper.GetString("STORYGEN_MODEL")

	cfg, err := gollm.LoadConfig()
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	keys := []string{
		"ANTHROPIC_API_KEY",
		"OPENAI_API_KEY",
		"DEEPSEEK_API_KEY",
		"OPENROUTER_API_KEY",
	}
	apiKey := ""
	for _, key := range keys {
		if !strings.Contains(strings.ToLower(key), strings.ToLower(provider)) {
			continue
		}
		apiKey = viper.GetString(key)
		if apiKey != "" {
			log.Printf("Using LLM provider %q with %q for story building\n", provider, model)
			break
		}
	}

	if apiKey == "" {
		log.Fatalf("No API key found for provider %q", provider)
	}

	registry := providers.NewProviderRegistry()
	registry.Register("deepseek", func(apiKey, model string, extraHeaders map[string]string) providers.Provider {
		return NewCustomOpenAIProvider(
			"deepseek",
			"https://api.deepseek.com/chat/completions",
			apiKey,
			model,
			extraHeaders,
		)
	})

	registry.Register("openrouter", func(apiKey, model string, extraHeaders map[string]string) providers.Provider {
		return NewCustomOpenAIProvider(
			"openrouter",
			"https://openrouter.ai/api/v1/chat/completions",
			apiKey,
			model,
			extraHeaders,
		)
	})

	cfg.Provider = provider
	cfg.APIKeys = map[string]string{provider: apiKey}
	cfg.Model = model
	cfg.MaxTokens = 4096
	cfg.MaxRetries = 30
	cfg.RetryDelay = time.Second * 5
	cfg.LogLevel = gollm.LogLevelInfo
	conn, err := llm.NewLLM(cfg, utils.NewLogger(cfg.LogLevel), registry)

	if err != nil {
		log.Fatalf("Failed to create LLM: %v", err)
		return nil, err
	}

	return &AI{
		client:   conn,
		audience: audience,
	}, nil
}
