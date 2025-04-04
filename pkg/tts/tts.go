package tts

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"

	"github.com/andrejsstepanovs/storygen/pkg/story"
	"github.com/andrejsstepanovs/storygen/pkg/tts/handlers"
)

func TextToSpeech(dir, outputFilePath, textToSpeech string, voice story.Voice, splitLen int, postProcess bool) error {
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
	err := JoinMp3Files(files, finalFile, "")
	if err != nil {
		return fmt.Errorf("failed to join MP3 files: %w", err)
	}

	fmt.Println("\nCleaning up temporary files...")
	err = Remove(files)
	if err != nil {
		fmt.Printf("Warning: Failed to remove temporary files: %v\n", err)
	}

	fmt.Println("\nTextToSpeech process completed successfully.")

	// OpenAI creates big pauses and silences in files.
	// Tried everything to remove them, but no luck.
	// So, we are using ffmpeg to remove them.
	if postProcess {
		cleanFile := path.Join(dir, "clean_"+outputFilePath)
		err = postProcessSilenceRemoval(finalFile, cleanFile)
		if err != nil {
			return fmt.Errorf("failed to post-process silence removal: %w", err)
		}
		fmt.Printf("Cleaned file saved as: %s\n", cleanFile)
		os.Remove(finalFile)
	}

	return nil
}

func postProcessSilenceRemoval(inputFile, outputFile string) error {
	// Create the command with proper argument separation
	cmd := exec.Command(
		"ffmpeg",
		"-i", inputFile,
		"-af", "silenceremove=stop_periods=-1:stop_duration=2:stop_threshold=-60dB",
		"-c:a", "libmp3lame", "-q:a", "0",
		outputFile,
	)

	// Capture both stdout and stderr
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	// Run the command
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("failed to execute FFmpeg command: %v\nOutput: %s", err, stderr.String())
	}

	return nil
}
