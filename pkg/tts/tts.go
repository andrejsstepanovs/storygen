package tts

import (
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/andrejsstepanovs/storygen/pkg/story"
	"github.com/andrejsstepanovs/storygen/pkg/tts/handlers"
)

func TextToSpeech(dir, outputFilePath, textToSpeech, inbetweenFile string, voice story.Voice) error {
	openaiHandler := &handlers.TTS{
		APIKey:       voice.Provider.APIKey,
		Model:        voice.Provider.Model,
		Voice:        voice.Provider.Voice,
		Instructions: voice.Instruction.String(),
		Speed:        voice.Provider.Speed,
		Client:       &http.Client{},
	}

	files := make([]string, 0)

	chapterTexts := splitByChapters(textToSpeech)

	if len(chapterTexts) == 0 {
		fmt.Println("Input text resulted in zero chapters after splitting.")
		return nil
	}

	splitLen := 1100
	for n, chapterText := range chapterTexts {
		if chapterText == "" {
			continue
		}

		chunks := chunkText(chapterText, splitLen)
		for k, chunk := range chunks {
			trimmedChunk := strings.TrimSpace(chunk)
			trimmedChunk = strings.TrimLeft(trimmedChunk, "...")
			trimmedChunk = strings.TrimSpace(trimmedChunk)
			if trimmedChunk == "" {
				continue
			}

			lines := strings.Split(trimmedChunk, "\n")
			cleanLines := make([]string, 0)
			for _, line := range lines {
				cleanLines = append(cleanLines, strings.TrimSpace(line))
			}
			cleanContent := strings.Join(cleanLines, "\n")

			file := fmt.Sprintf("%d_%d_%s", n, k, outputFilePath) // n=segment index, k=chunk index
			targetFile := path.Join(dir, file)

			fmt.Printf(">>> %s\n%s\n<<<\n", targetFile, cleanContent)

			err := openaiHandler.Convert(cleanContent, targetFile)
			if err != nil {
				return err
			}

			files = append(files, targetFile)
		}
	}

	if len(files) == 0 {
		fmt.Println("No audio files were generated.")
		return fmt.Errorf("no audio files generated, cannot join")
	}

	fmt.Printf("\nJoining %d audio segments...\n", len(files))
	finalFile := path.Join(dir, outputFilePath)
	err := JoinMp3Files(files, finalFile, inbetweenFile)
	if err != nil {
		return fmt.Errorf("failed to join MP3 files: %w", err)
	}

	fmt.Println("\nCleaning up temporary files...")
	err = Remove(files)
	if err != nil {
		fmt.Printf("Warning: Failed to remove temporary files: %v\n", err)
	}

	fmt.Println("\nTextToSpeech process completed successfully.")

	// todo. if we want to build bigger mp3 file chunks it comes with annoying silence pauses coming from openai.
	// this command is removing the silences with post processing
	// alternative is to keep chunks (splitLen) short
	//cmd := fmt.Sprintf("ffmpeg -i %s -af silenceremove=stop_periods=-1:stop_duration=2:stop_threshold=-50dB %s", finalFile, "clean_"+finalFile)

	return nil
}
