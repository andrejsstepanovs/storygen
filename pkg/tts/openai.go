package tts

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/sashabaranov/go-openai"
	"github.com/spf13/viper"
)

func TextToSpeech(voice openai.SpeechVoice, outputFilePath, textToSpeech, inbetweenFile string) error {
	files := make([]string, 0)
	chunks := chunkText(textToSpeech, 2000)
	for i, chunk := range chunks {
		file := fmt.Sprintf("%d_%s", i, outputFilePath)
		files = append(files, file)
		err := openaiFile(voice, file, chunk)
		if err != nil {
			return fmt.Errorf("chunk processing failed: %v", err)
		}
	}

	return JoinMp3Files(files, outputFilePath, inbetweenFile)
}

func openaiFile(voice openai.SpeechVoice, outputFilePath, textToSpeech string) error {
	request := openai.CreateSpeechRequest{
		Model: openai.TTSModel1HD,
		Voice: voice,
		Input: textToSpeech,
		Speed: 0.9,
	}

	c := openai.NewClient(viper.GetString("OPENAI_API_KEY"))
	resp, err := c.CreateSpeech(context.Background(), request)
	if err != nil {
		fmt.Printf("Speech generation error: %v\n", err)
		return err
	}
	defer resp.Close()

	buf, err := io.ReadAll(resp)

	err = os.WriteFile(outputFilePath, buf, 0644)

	return err
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
