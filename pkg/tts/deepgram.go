package tts

import (
	"context"
	"fmt"
	"strconv"

	api "github.com/deepgram/deepgram-go-sdk/pkg/api/speak/v1/rest"
	"github.com/deepgram/deepgram-go-sdk/pkg/client/interfaces"
	client "github.com/deepgram/deepgram-go-sdk/pkg/client/speak"
	"github.com/spf13/viper"
)

func DeepgramTextToSpeech(outputFilePath string, textToSpeech string) error {
	client.InitWithDefault()
	ctx := context.Background()

	options := &interfaces.SpeakOptions{
		Model: "aura-hera-en",
	}

	c := client.NewREST(viper.GetString("DEEPGRAM_API_KEY"), &interfaces.ClientOptions{})
	dg := api.New(c)

	return processLargeText(ctx, outputFilePath, textToSpeech, dg, options)
}

func processLargeText(ctx context.Context, outputFilePath, text string, dg *api.Client, options *interfaces.SpeakOptions) error {
	chunks := chunkText(text, 2000)

	files := make([]string, 0)
	for i, chunk := range chunks {
		file := strconv.Itoa(i) + "_" + outputFilePath
		files = append(files, file)
		_, err := dg.ToSave(ctx, file, chunk, options)
		if err != nil {
			return fmt.Errorf("chunk processing failed: %v", err)
		}
	}

	return JoinMp3Files(files, outputFilePath)
}

func chunkText(text string, chunkSize int) []string {
	if chunkSize <= 0 {
		return nil
	}

	var chunks []string
	runes := []rune(text)
	length := len(runes)
	var chunk []rune
	i := 0

	// Function to check if a rune is a sentence-ending punctuation
	isSentenceEnd := func(r rune) bool {
		return r == '.' || r == '!' || r == '?'
	}

	for i < length {
		// Start a new chunk if the current one is empty
		if len(chunk) == 0 {
			chunk = make([]rune, 0, chunkSize)
		}

		// Find the end of the current sentence
		j := i
		for j < length && !isSentenceEnd(runes[j]) {
			j++
		}
		// Include the sentence-ending punctuation
		if j < length && isSentenceEnd(runes[j]) {
			j++
		}

		// Check if adding this sentence exceeds the chunk size
		if len(chunk)+j-i > chunkSize {
			// If the chunk is empty, this sentence is too long; make it a chunk by itself
			if len(chunk) == 0 {
				chunks = append(chunks, string(runes[i:j]))
				chunk = nil
				i = j
				continue
			}
			// Otherwise, finalize the current chunk up to the previous sentence
			chunks = append(chunks, string(chunk))
			chunk = nil
			continue
		}

		// Add the sentence to the current chunk
		chunk = append(chunk, runes[i:j]...)
		i = j

		// If the chunk is full, add it to chunks and start a new chunk
		if len(chunk) == chunkSize {
			chunks = append(chunks, string(chunk))
			chunk = nil
		}
	}

	// Add any remaining text in the chunk
	if len(chunk) > 0 {
		chunks = append(chunks, string(chunk))
	}

	return chunks
}
