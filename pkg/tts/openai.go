package tts

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/sashabaranov/go-openai"
	"github.com/spf13/viper"
)

func OpenAITextToSpeech(voice openai.SpeechVoice, outputFilePath, textToSpeech string) error {
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

	return JoinMp3Files(files, outputFilePath)
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
