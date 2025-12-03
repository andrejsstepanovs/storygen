# go-litellm

go-litellm is a comprehensive Go client library for interacting with the LiteLLM API, which provides unified access to multiple AI language models. This library abstracts the complexities of working with different AI providers through LiteLLM's proxy service, enabling developers to easily integrate AI capabilities into their Go applications. The library supports a wide range of operations including chat completions, audio transcription, text-to-speech conversion, embeddings generation, token counting, and MCP (Model Context Protocol) tool interactions.

The library is built with type safety, developer ergonomics, and production readiness in mind. It features automatic retry mechanisms with exponential backoff, comprehensive error handling, structured request/response types, and support for advanced features like streaming, JSON schema validation, image analysis, and tool calling. The clean API design allows developers to quickly implement AI functionality while maintaining full control over model parameters, temperature settings, and timeout configurations.

## Client Initialization

Creating a new LiteLLM client with connection configuration and API credentials.

```go
package main

import (
    "context"
    "log"
    "net/url"
    "time"

    "github.com/andrejsstepanovs/go-litellm/client"
    "github.com/andrejsstepanovs/go-litellm/conf/connections/litellm"
)

func main() {
    ctx := context.Background()

    // Parse base URL for LiteLLM service
    baseURL, err := url.Parse("http://localhost:4000")
    if err != nil {
        log.Fatal("Failed to parse URL:", err)
    }

    // Configure connection with different timeout targets
    conn := litellm.Connection{
        URL: *baseURL,
        Targets: litellm.Targets{
            System: litellm.Target{Timeout: time.Second * 30},
            LLM:    litellm.Target{Timeout: time.Minute * 2},
            MCP:    litellm.Target{Timeout: time.Minute * 5},
        },
    }

    // Validate connection configuration
    if err := conn.Validate(); err != nil {
        log.Fatal("Connection validation failed:", err)
    }

    // Create client configuration
    cfg := client.Config{
        APIKey:      "sk-1234",
        Temperature: 0.7,
    }

    // Initialize client
    ai, err := client.New(cfg, conn)
    if err != nil {
        log.Fatal("Failed to create client:", err)
    }

    log.Println("LiteLLM client initialized successfully")
    _ = ai
}
```

## Chat Completion

Basic chat completion request to generate AI responses from text prompts.

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/andrejsstepanovs/go-litellm/client"
    "github.com/andrejsstepanovs/go-litellm/request"
)

func main() {
    ctx := context.Background()
    ai := getClient() // Assume initialized client

    // Get model metadata
    model, err := ai.Model(ctx, "claude-4")
    if err != nil {
        log.Fatal("Failed to get model:", err)
    }

    // Create simple user message
    messages := request.Messages{
        request.UserMessageSimple("What is the capital of France?"),
    }

    // Build completion request
    req := request.NewCompletionRequest(model, messages, nil, nil, 0.7)

    // Execute completion
    resp, err := ai.Completion(ctx, req)
    if err != nil {
        log.Fatal("Completion failed:", err)
    }

    // Access response
    fmt.Println("Response:", resp.String())
    fmt.Printf("Tokens used: %d (prompt: %d, completion: %d)\n",
        resp.Usage.TotalTokens,
        resp.Usage.PromptTokens,
        resp.Usage.CompletionTokens)
    fmt.Println("Finish reason:", resp.Choice().FinishReason)
}

func getClient() *client.Litellm {
    // Implementation omitted for brevity
    return nil
}
```

## Multi-turn Conversation

Building conversational context with multiple messages and roles.

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/andrejsstepanovs/go-litellm/request"
)

func main() {
    ctx := context.Background()
    ai := getClient()

    model, _ := ai.Model(ctx, "gpt-4")

    // Build conversation history
    messages := request.Messages{
        request.SystemMessageSimple("You are a helpful coding assistant."),
        request.UserMessageSimple("How do I reverse a string in Go?"),
        request.AssistantMessageSimple("You can reverse a string by converting it to runes..."),
        request.UserMessageSimple("Can you show me the complete code?"),
    }

    req := request.NewCompletionRequest(model, messages, nil, nil, 0.7)
    resp, err := ai.Completion(ctx, req)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Assistant:", resp.String())

    // Add AI response to conversation history
    messages.AddMessage(request.AIMessage(resp.Message()))

    // Continue conversation
    messages.AddMessage(request.UserMessageSimple("Can you optimize it?"))
    req = request.NewCompletionRequest(model, messages, nil, nil, 0.7)
    resp, err = ai.Completion(ctx, req)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Assistant:", resp.String())
}
```

## Image Analysis

Analyzing images by providing image URLs in message content.

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/andrejsstepanovs/go-litellm/request"
)

func main() {
    ctx := context.Background()
    ai := getClient()

    // Use vision-capable model
    model, err := ai.Model(ctx, "gpt-4o-mini")
    if err != nil {
        log.Fatal(err)
    }

    // Create message with image URL
    imageUrl := request.MessageImage("https://example.com/image.jpg")
    messages := request.Messages{
        request.UserMessageImage("Describe this image in detail", imageUrl),
    }

    req := request.NewCompletionRequest(model, messages, nil, nil, 0.7)
    resp, err := ai.Completion(ctx, req)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Image description:", resp.String())

    // Multiple images in one request
    messages = request.Messages{
        request.UserMessageImage(
            "Compare these two images",
            request.MessageImage("https://example.com/image1.jpg"),
        ),
        request.UserMessage(request.MessageContents{
            {Type: "image_url", ImageUrl: &request.ImageUrl{URL: "https://example.com/image2.jpg"}},
        }),
    }

    req = request.NewCompletionRequest(model, messages, nil, nil, 0.7)
    resp, err = ai.Completion(ctx, req)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Println("Comparison:", resp.String())
}
```

## Structured JSON Output

Generate structured JSON responses conforming to a strict schema.

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"

    "github.com/andrejsstepanovs/go-litellm/request"
)

// Define output structure
type City struct {
    CityName        string `json:"city_name"`
    PopulationCount int    `json:"population_count"`
    Country         string `json:"country"`
}

type ListOfCities struct {
    Cities []City `json:"cities"`
}

func main() {
    ctx := context.Background()
    ai := getClient()

    model, _ := ai.Model(ctx, "claude-4")

    // Define JSON schema
    schema := request.JSONSchema{
        Name: "list_of_cities",
        Schema: map[string]interface{}{
            "type": "object",
            "properties": map[string]interface{}{
                "cities": map[string]interface{}{
                    "type": "array",
                    "items": map[string]interface{}{
                        "type": "object",
                        "properties": map[string]interface{}{
                            "city_name":        map[string]interface{}{"type": "string"},
                            "population_count": map[string]interface{}{"type": "integer"},
                            "country":          map[string]interface{}{"type": "string"},
                        },
                        "required": []string{"city_name", "population_count", "country"},
                        "additionalProperties": false,
                    },
                },
            },
            "required": []string{"cities"},
            "additionalProperties": false,
        },
        Strict: true,
    }

    messages := request.Messages{
        request.UserMessageSimple("List the 5 largest cities in Europe with their populations"),
    }

    req := request.NewCompletionRequest(model, messages, nil, nil, 0.2)
    req.SetJSONSchema(schema) // Enable strict JSON mode

    resp, err := ai.Completion(ctx, req)
    if err != nil {
        log.Fatal(err)
    }

    // Unmarshal structured response
    var cities ListOfCities
    if err := json.Unmarshal(resp.Bytes(), &cities); err != nil {
        log.Fatal("Failed to unmarshal JSON:", err)
    }

    // Access structured data
    for _, city := range cities.Cities {
        fmt.Printf("%s, %s - Population: %d\n",
            city.CityName, city.Country, city.PopulationCount)
    }
}
```

## JSON Mode Output

Generate JSON responses without strict schema validation.

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"

    "github.com/andrejsstepanovs/go-litellm/request"
)

func main() {
    ctx := context.Background()
    ai := getClient()

    model, _ := ai.Model(ctx, "gpt-4")

    messages := request.Messages{
        request.SystemMessageSimple("You are a helpful assistant that outputs JSON."),
        request.UserMessageSimple("Give me information about the Go programming language"),
    }

    req := request.NewCompletionRequest(model, messages, nil, nil, 0.7)
    req.SetJSONMode() // Enable JSON object mode (less strict than schema)

    resp, err := ai.Completion(ctx, req)
    if err != nil {
        log.Fatal(err)
    }

    // Parse generic JSON
    var result map[string]interface{}
    if err := json.Unmarshal(resp.Bytes(), &result); err != nil {
        log.Fatal("Failed to parse JSON:", err)
    }

    fmt.Printf("JSON response: %+v\n", result)
}
```

## Speech to Text (Audio Transcription)

Transcribe audio files to text using speech recognition models.

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/andrejsstepanovs/go-litellm/audio"
)

func main() {
    ctx := context.Background()
    ai := getClient()

    // Get speech recognition model
    model, err := ai.Model(ctx, "whisper-1")
    if err != nil {
        log.Fatal("Failed to get model:", err)
    }

    // Transcribe audio file
    audioFilePath := "/path/to/audio.oga"
    result, err := ai.SpeechToText(ctx, model, audioFilePath)
    if err != nil {
        log.Fatal("Transcription failed:", err)
    }

    fmt.Println("Transcription:", result.Text)

    // Process different audio formats
    audioFormats := []string{
        "/path/to/recording.mp3",
        "/path/to/speech.wav",
        "/path/to/voice.m4a",
    }

    for _, audioFile := range audioFormats {
        res, err := ai.SpeechToText(ctx, model, audioFile)
        if err != nil {
            log.Printf("Failed to transcribe %s: %v\n", audioFile, err)
            continue
        }
        fmt.Printf("File: %s\nText: %s\n\n", audioFile, res.Text)
    }
}
```

## Text to Speech

Convert text to spoken audio files with customizable voice parameters.

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"

    "github.com/andrejsstepanovs/go-litellm/models"
    "github.com/andrejsstepanovs/go-litellm/request"
)

func main() {
    ctx := context.Background()
    ai := getClient()

    // Configure speech request
    speechRequest := request.Speech{
        Model:          models.ModelID("tts-1"),
        Input:          "Hello, this is a test of the text to speech system.",
        Voice:          "alloy", // Options: alloy, echo, fable, onyx, nova, shimmer
        ResponseFormat: "mp3",   // Options: mp3, opus, aac, flac, wav, pcm
        Speed:          1.0,     // Range: 0.25 to 4.0
    }

    // Generate speech
    result, err := ai.TextToSpeech(ctx, speechRequest)
    if err != nil {
        log.Fatal("Text-to-speech failed:", err)
    }

    fmt.Printf("Audio file created: %s\n", result.Full)
    fmt.Printf("File name: %s\n", result.Name)
    fmt.Printf("Directory: %s\n", result.Directory)
    fmt.Printf("Format: %s\n", result.Extension)

    // Read audio file
    audioData, err := os.ReadFile(result.Full)
    if err != nil {
        log.Fatal("Failed to read audio file:", err)
    }

    fmt.Printf("Audio file size: %d bytes\n", len(audioData))

    // Generate with different voices and speeds
    voices := []string{"alloy", "echo", "nova"}
    for _, voice := range voices {
        req := request.Speech{
            Model:          models.ModelID("tts-1-hd"),
            Input:          "Testing different voices",
            Voice:          voice,
            Speed:          1.25,
            ResponseFormat: "opus",
        }

        res, err := ai.TextToSpeech(ctx, req)
        if err != nil {
            log.Printf("Failed with voice %s: %v\n", voice, err)
            continue
        }

        fmt.Printf("Generated audio with voice %s: %s\n", voice, res.Full)
    }
}
```

## Embeddings Generation

Generate vector embeddings for text input to enable semantic search and similarity comparisons.

```go
package main

import (
    "context"
    "fmt"
    "log"
    "math"

    "github.com/andrejsstepanovs/go-litellm/response"
)

func main() {
    ctx := context.Background()
    ai := getClient()

    // Get embedding model
    model, err := ai.Model(ctx, "text-embedding-ada-002")
    if err != nil {
        log.Fatal(err)
    }

    // Generate embedding for single text
    inputText := "The quick brown fox jumps over the lazy dog"
    embeddingResp, err := ai.Embeddings(ctx, model, inputText)
    if err != nil {
        log.Fatal("Embeddings failed:", err)
    }

    // Access embedding vector
    embedding := embeddingResp.Data[0].Embedding
    fmt.Printf("Embedding dimension: %d\n", len(embedding))
    fmt.Printf("First 5 values: %v\n", embedding[:5])
    fmt.Printf("Tokens used: %d\n", embeddingResp.Usage.TotalTokens)

    // Convert to float32 if needed
    float32Embedding := embedding.Float32()
    fmt.Printf("Float32 embedding length: %d\n", len(float32Embedding))

    // Generate multiple embeddings for comparison
    texts := []string{
        "I love programming in Go",
        "Go is a great programming language",
        "The weather is nice today",
    }

    embeddings := make([]response.Embedding, len(texts))
    for i, text := range texts {
        resp, err := ai.Embeddings(ctx, model, text)
        if err != nil {
            log.Fatal(err)
        }
        embeddings[i] = resp.Data[0].Embedding
    }

    // Calculate cosine similarity between first two texts
    similarity := cosineSimilarity(embeddings[0], embeddings[1])
    fmt.Printf("Similarity between text 1 and 2: %.4f\n", similarity)

    similarity = cosineSimilarity(embeddings[0], embeddings[2])
    fmt.Printf("Similarity between text 1 and 3: %.4f\n", similarity)
}

func cosineSimilarity(a, b response.Embedding) float64 {
    var dotProduct, normA, normB float64
    for i := range a {
        dotProduct += a[i] * b[i]
        normA += a[i] * a[i]
        normB += b[i] * b[i]
    }
    return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
```

## Token Counting

Calculate token usage for messages before sending requests to manage costs and limits.

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/andrejsstepanovs/go-litellm/request"
)

func main() {
    ctx := context.Background()
    ai := getClient()

    model, _ := ai.Model(ctx, "claude-4")

    // Count tokens for simple message
    messages := request.Messages{
        request.UserMessageSimple("Hello, how are you?"),
    }

    tokenReq := &request.TokenCounterRequest{
        Model:    model.ModelId,
        Messages: messages,
    }

    count, err := ai.TokenCounter(ctx, tokenReq)
    if err != nil {
        log.Fatal("Token counting failed:", err)
    }

    fmt.Printf("Total tokens: %.0f\n", count.TotalTokens)
    fmt.Printf("Model used: %s\n", count.ModelUsed)
    fmt.Printf("Tokenizer type: %s\n", count.TokenizerType)

    // Count tokens for complex conversation
    complexMessages := request.Messages{
        request.SystemMessageSimple("You are a helpful assistant specialized in Go programming."),
        request.UserMessageSimple("Can you explain how goroutines work?"),
        request.AssistantMessageSimple("Goroutines are lightweight threads managed by the Go runtime..."),
        request.UserMessageSimple("How do I use channels with goroutines?"),
    }

    complexReq := &request.TokenCounterRequest{
        Model:    model.ModelId,
        Messages: complexMessages,
    }

    complexCount, err := ai.TokenCounter(ctx, complexReq)
    if err != nil {
        log.Fatal(err)
    }

    fmt.Printf("\nComplex conversation tokens: %.0f\n", complexCount.TotalTokens)

    // Estimate cost before API call
    const costPerToken = 0.00001 // Example cost
    estimatedCost := complexCount.TotalTokens * costPerToken
    fmt.Printf("Estimated cost: $%.6f\n", estimatedCost)
}
```

## List Available Models

Retrieve all available AI models and their metadata from the LiteLLM service.

```go
package main

import (
    "context"
    "fmt"
    "log"
)

func main() {
    ctx := context.Background()
    ai := getClient()

    // List all available models
    models, err := ai.Models(ctx)
    if err != nil {
        log.Fatal("Failed to list models:", err)
    }

    fmt.Printf("Total models available: %d\n\n", len(models))

    // Display model information
    for _, model := range models {
        fmt.Printf("Model ID: %s\n", model.ID)
        fmt.Printf("  Owned by: %s\n", model.OwnedBy)
        fmt.Printf("  Created: %d\n", model.Created)
        fmt.Printf("  Object: %s\n\n", model.Object)
    }

    // Get model info mapping (model name -> litellm key)
    modelMap, err := ai.ModelInfoMap(ctx)
    if err != nil {
        log.Fatal("Failed to get model info map:", err)
    }

    fmt.Println("Model name mappings:")
    for name, key := range modelMap {
        fmt.Printf("  %s -> %s\n", name, key)
    }

    // Get specific model details
    specificModel, err := ai.Model(ctx, "claude-4")
    if err != nil {
        log.Fatal("Failed to get model details:", err)
    }

    fmt.Printf("\nClaude-4 details:\n")
    fmt.Printf("  Model ID: %s\n", specificModel.ModelId)
    fmt.Printf("  Max tokens: %d\n", specificModel.MaxTokens)
    fmt.Printf("  Supported params: %v\n", specificModel.SupportedOpenAIParams)
}
```

## List MCP Tools

Discover available MCP (Model Context Protocol) tools for function calling.

```go
package main

import (
    "context"
    "encoding/json"
    "fmt"
    "log"
)

func main() {
    ctx := context.Background()
    ai := getClient()

    // List all available tools
    tools, err := ai.Tools(ctx)
    if err != nil {
        log.Fatal("Failed to list tools:", err)
    }

    fmt.Printf("Total tools available: %d\n\n", len(tools))

    // Display tool information
    for i, tool := range tools {
        fmt.Printf("%d. %s\n", i+1, tool.Name)
        fmt.Printf("   Description: %s\n", tool.Description)
        fmt.Printf("   Type: %s\n", tool.Type)

        if tool.McpInfo.ServerName != "" {
            fmt.Printf("   MCP Server: %s\n", tool.McpInfo.ServerName)
        }

        // Show input schema
        if len(tool.InputSchema.Properties) > 0 {
            fmt.Println("   Parameters:")
            for propName, prop := range tool.InputSchema.Properties {
                required := ""
                for _, req := range tool.InputSchema.Required {
                    if req == propName {
                        required = " (required)"
                        break
                    }
                }
                fmt.Printf("     - %s: %s%s - %s\n",
                    propName, prop.Type, required, prop.Description)
            }
        }
        fmt.Println()
    }

    // Convert tools for LLM requests
    llmTools := tools.ToLLMCallTools()
    fmt.Printf("Converted to LLM format: %d tools\n", len(llmTools))

    // Inspect specific tool
    if len(tools) > 0 {
        toolJSON, _ := json.MarshalIndent(tools[0], "", "  ")
        fmt.Printf("\nExample tool JSON:\n%s\n", string(toolJSON))
    }
}
```

## MCP Tool Calling

Execute MCP tools with parameters and handle their responses.

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/andrejsstepanovs/go-litellm/common"
)

func main() {
    ctx := context.Background()
    ai := getClient()

    // Call a simple tool
    toolCall := common.ToolCallFunction{
        Name:      "current_time",
        Arguments: map[string]string{"timezone": "Europe/Riga"},
    }

    result, err := ai.ToolCall(ctx, toolCall)
    if err != nil {
        log.Fatal("Tool call failed:", err)
    }

    // Access tool response
    for _, res := range result {
        fmt.Printf("Tool response type: %s\n", res.Type)
        fmt.Printf("Result: %s\n", res.Text)
        if res.Annotations != nil {
            fmt.Printf("Annotations: %+v\n", res.Annotations)
        }
    }

    // Call tool with complex parameters
    searchTool := common.ToolCallFunction{
        Name: "web_search",
        Arguments: map[string]string{
            "query":   "latest Go programming news",
            "max_results": "5",
        },
    }

    searchResult, err := ai.ToolCall(ctx, searchTool)
    if err != nil {
        log.Fatal("Search tool failed:", err)
    }

    fmt.Printf("\nSearch results:\n%s\n", searchResult.String())

    // Call calculation tool
    calcTool := common.ToolCallFunction{
        Name: "calculator",
        Arguments: map[string]string{
            "operation": "multiply",
            "a":         "15",
            "b":         "23",
        },
    }

    calcResult, err := ai.ToolCall(ctx, calcTool)
    if err != nil {
        log.Fatal("Calculator tool failed:", err)
    }

    fmt.Printf("Calculation result: %s\n", calcResult[0].Text)
}
```

## Tool-Aware Conversation

Build intelligent conversations where the AI can call tools and use their results.

```go
package main

import (
    "context"
    "fmt"
    "log"
    "net/url"
    "time"

    "github.com/andrejsstepanovs/go-litellm/client"
    "github.com/andrejsstepanovs/go-litellm/conf/connections/litellm"
    "github.com/andrejsstepanovs/go-litellm/mcp"
    "github.com/andrejsstepanovs/go-litellm/models"
    "github.com/andrejsstepanovs/go-litellm/request"
    "github.com/andrejsstepanovs/go-litellm/response"
)

func main() {
    ctx := context.Background()

    // Initialize client
    baseURL, _ := url.Parse("http://localhost:4000")
    conn := litellm.Connection{
        URL: *baseURL,
        Targets: litellm.Targets{
            System: litellm.Target{Timeout: time.Second * 30},
            LLM:    litellm.Target{Timeout: time.Minute * 2},
            MCP:    litellm.Target{Timeout: time.Minute * 5},
        },
    }

    cfg := client.Config{
        APIKey:      "sk-1234",
        Temperature: 0.7,
    }

    ai, err := client.New(cfg, conn)
    if err != nil {
        log.Fatal(err)
    }

    // Get model and available tools
    model, _ := ai.Model(ctx, "claude-4")
    tools, _ := ai.Tools(ctx)

    // Start conversation with tool awareness
    messages := request.Messages{
        request.SystemMessageSimple("You are a helpful assistant with access to tools."),
        request.UserMessageSimple("What's the current time in Tokyo and what's the weather like there?"),
    }

    // Run conversation loop
    finalResp := runToolConversation(ctx, ai, model, tools, messages, 10)
    fmt.Println("\nFinal Answer:", finalResp.String())
}

func runToolConversation(
    ctx context.Context,
    ai *client.Litellm,
    model models.ModelMeta,
    tools mcp.AvailableTools,
    messages request.Messages,
    maxIterations int,
) response.Response {
    if maxIterations <= 0 {
        log.Println("Max iterations reached")
        return response.Response{}
    }

    // Create request with available tools
    req := request.NewCompletionRequest(
        model,
        messages,
        tools.ToLLMCallTools(),
        nil,
        0.7,
    )

    // Get AI response
    resp, err := ai.Completion(ctx, req)
    if err != nil {
        log.Fatal("Completion error:", err)
    }

    // Add AI response to conversation
    messages.AddMessage(request.AIMessage(resp.Message()))

    // Check if AI wants to call tools
    if resp.Choice().FinishReason == response.FINISH_REASON_TOOL {
        fmt.Printf("\nAI requested %d tool call(s)\n", len(resp.Choice().Message.ToolCalls))

        // Execute all requested tool calls
        for _, toolCall := range resp.Choice().Message.ToolCalls.SortASC() {
            fmt.Printf("Calling tool: %s with args: %v\n",
                toolCall.Function.Name,
                toolCall.Function.Arguments)

            toolResp, err := ai.ToolCall(ctx, toolCall.Function)
            if err != nil {
                log.Printf("Tool call error: %v\n", err)
                continue
            }

            // Add tool results to conversation
            for _, tr := range toolResp {
                fmt.Printf("Tool result: %s\n", tr.Text)
                messages = append(messages, request.ToolCallMessage(toolCall, tr))
            }
        }

        // Continue conversation with tool results
        return runToolConversation(ctx, ai, model, tools, messages, maxIterations-1)
    }

    // Conversation complete
    return resp
}
```

## Advanced Request Configuration

Fine-tune requests with custom temperature, tool choices, and response formats.

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/andrejsstepanovs/go-litellm/request"
)

func main() {
    ctx := context.Background()
    ai := getClient()

    model, _ := ai.Model(ctx, "gpt-4")

    // Create base request
    req := request.NewRequest(model)

    // Configure messages
    messages := request.Messages{
        request.SystemMessageSimple("You are a creative writing assistant."),
        request.UserMessageSimple("Write a short story about a robot."),
    }
    req.SetMessages(messages)

    // Set custom temperature (lower = more deterministic)
    customTemp := float32(0.9)
    req.SetTemperature(customTemp, model.SupportedOpenAIParams)

    // Execute request
    resp, err := ai.Completion(ctx, req)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("Creative story:", resp.String())

    // Request with JSON mode for structured output
    req2 := request.NewRequest(model)
    req2.SetMessages(request.Messages{
        request.UserMessageSimple("List 3 programming languages with their use cases"),
    })
    req2.SetJSONMode()

    resp2, err := ai.Completion(ctx, req2)
    if err != nil {
        log.Fatal(err)
    }
    fmt.Println("\nJSON response:", resp2.String())

    // Request with tools but force specific tool usage
    tools, _ := ai.Tools(ctx)
    req3 := request.NewRequest(model)
    req3.SetMessages(request.Messages{
        request.UserMessageSimple("What's the weather?"),
    })
    req3.SetAvailableTools(tools.ToLLMCallTools())
    req3.ToolChoice = "auto" // Options: auto, none, or specific tool name

    resp3, err := ai.Completion(ctx, req3)
    if err != nil {
        log.Fatal(err)
    }

    if resp3.Choice().FinishReason == response.FINISH_REASON_TOOL {
        fmt.Println("\nAI chose to use tools:")
        for _, tc := range resp3.Choice().Message.ToolCalls {
            fmt.Printf("  - %s\n", tc.Function.Name)
        }
    }
}
```

## Error Handling

Comprehensive error handling for various failure scenarios.

```go
package main

import (
    "context"
    "errors"
    "fmt"
    "log"
    "net/url"

    "github.com/andrejsstepanovs/go-litellm/client"
    "github.com/andrejsstepanovs/go-litellm/conf/connections/litellm"
    "github.com/andrejsstepanovs/go-litellm/request"
)

func main() {
    ctx := context.Background()

    // Handle client initialization errors
    baseURL, err := url.Parse("http://invalid-url")
    if err != nil {
        log.Printf("URL parse error: %v\n", err)
        return
    }

    conn := litellm.Connection{URL: *baseURL}

    // Validate connection before use
    if err := conn.Validate(); err != nil {
        log.Printf("Connection validation failed: %v\n", err)
        return
    }

    cfg := client.Config{
        APIKey:      "sk-test",
        Temperature: 0.7,
    }

    // Validate config
    if err := cfg.Validate(); err != nil {
        log.Printf("Config validation failed: %v\n", err)
        return
    }

    ai, err := client.New(cfg, conn)
    if err != nil {
        log.Printf("Client initialization failed: %v\n", err)
        return
    }

    // Handle model retrieval errors
    model, err := ai.Model(ctx, "non-existent-model")
    if err != nil {
        log.Printf("Model not found: %v\n", err)
        // Fallback to default model
        model, err = ai.Model(ctx, "gpt-3.5-turbo")
        if err != nil {
            log.Fatal("Failed to get fallback model:", err)
        }
    }

    // Handle empty messages error
    emptyMessages := request.Messages{}
    req := request.NewCompletionRequest(model, emptyMessages, nil, nil, 0.7)
    _, err = ai.Completion(ctx, req)
    if err != nil {
        log.Printf("Completion with empty messages failed: %v\n", err)
    }

    // Handle API errors with proper messages
    messages := request.Messages{
        request.UserMessageSimple("Test message"),
    }
    req = request.NewCompletionRequest(model, messages, nil, nil, 0.7)

    resp, err := ai.Completion(ctx, req)
    if err != nil {
        // Check for specific error types
        if errors.Is(err, context.DeadlineExceeded) {
            log.Println("Request timed out")
        } else if errors.Is(err, context.Canceled) {
            log.Println("Request was canceled")
        } else {
            log.Printf("Completion failed: %v\n", err)
        }
        return
    }

    // Validate response before use
    if resp.Message().IsEmpty() {
        log.Println("Received empty response from API")
        return
    }

    fmt.Println("Success:", resp.String())

    // Handle tool call errors
    toolCall := common.ToolCallFunction{
        Name:      "invalid_tool",
        Arguments: map[string]string{},
    }

    _, err = ai.ToolCall(ctx, toolCall)
    if err != nil {
        log.Printf("Tool call failed: %v\n", err)
    }

    // Handle file errors for audio
    _, err = ai.SpeechToText(ctx, model, "/non/existent/file.mp3")
    if err != nil {
        log.Printf("Audio transcription failed: %v\n", err)
    }
}
```

## Summary

go-litellm provides a production-ready, type-safe interface for integrating AI capabilities into Go applications through the LiteLLM proxy service. The library supports all major AI operations including text completions, vision analysis, audio processing, embeddings generation, and tool calling through the Model Context Protocol. Its clean API design makes it straightforward to implement complex AI workflows while maintaining code clarity and reliability.

The library is ideal for building AI-powered applications, chatbots, content generation systems, semantic search engines, voice-enabled interfaces, and agentic systems that can interact with external tools. It abstracts away the complexity of managing different AI providers, handles retries and timeouts automatically, and provides comprehensive error handling. Whether you're building a simple chatbot or a sophisticated multi-agent system with tool calling capabilities, go-litellm offers the flexibility and robustness needed for production deployments.
