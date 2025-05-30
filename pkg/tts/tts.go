package tts

import (
	"bytes"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/andrejsstepanovs/storygen/pkg/story"
	"github.com/andrejsstepanovs/storygen/pkg/tts/handlers"
)

func TextToSpeech(dir, outputFilePath, textToSpeech string, voice story.Voice, splitLen int, postProcess bool) (string, error) {
	openaiHandler := &handlers.TTS{
		APIKey:          voice.Provider.APIKey,
		Model:           voice.Provider.Model,
		Voice:           voice.Provider.Voice,
		Instructions:    voice.Instruction.String(),
		Speed:           voice.Provider.Speed,
		Client:          &http.Client{},
		MaxRetries:      3,               // Retry up to 3 times
		RetryDelay:      2 * time.Second, // Start with 2 seconds delay
		RetryMultiplier: 1.5,             // Increase delay by 50% each retry
	}

	files := make([]string, 0)

	chapterTexts := splitByChapters(textToSpeech)

	if len(chapterTexts) == 0 {
		fmt.Println("Input text resulted in zero chapters after splitting.")
		return "", nil
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
				return "", err
			}
			time.Sleep(time.Second * 1) // desperate try to fix openai broken responses. looks like its working!

			files = append(files, targetFile)
		}
	}

	if len(files) == 0 {
		fmt.Println("No audio files were generated.")
		return "", fmt.Errorf("no audio files generated, cannot join")
	}

	fmt.Printf("\nJoining %d audio segments...\n", len(files))
	finalFile := path.Join(dir, outputFilePath)
	err := JoinMp3Files(files, finalFile, "")
	if err != nil {
		return "", fmt.Errorf("failed to join MP3 files: %w", err)
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
		unnoisedFile := path.Join(dir, "unnoised_"+outputFilePath)
		err = postProcessNoiseRemoval(finalFile, unnoisedFile)
		if err != nil {
			return "", fmt.Errorf("failed to post-process noise removal: %w", err)
		}

		cleanFile := path.Join(dir, "clean_"+outputFilePath)
		err = postProcessSilenceRemoval(unnoisedFile, cleanFile)
		if err != nil {
			return "", fmt.Errorf("failed to post-process silence removal: %w", err)
		}
		fmt.Printf("Cleaned file saved as: %s\n", cleanFile)
		os.Remove(finalFile)
		os.Remove(unnoisedFile)

		return cleanFile, nil
	}

	return finalFile, nil
}

func postProcessNoiseRemoval(inputFile, outputFile string) error {
	cmd := exec.Command(
		"ffmpeg",
		"-i", inputFile,
		"-af", "compand=attacks=0:decays=0.7:points=-80/-80|-6/-6|-2/-80",
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
