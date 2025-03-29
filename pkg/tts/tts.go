package tts

import (
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/andrejsstepanovs/storygen/pkg/tts/handlers"
	"github.com/spf13/viper"
)

func TextToSpeech(dir, outputFilePath, textToSpeech, inbetweenFile string) error {
	speed := viper.GetFloat64("STORYGEN_SPEECH_SPEED")
	if speed == 0 {
		speed = 0.9
	}
	openaiHandler := &handlers.TTS{
		APIKey:       viper.GetString("OPENAI_API_KEY"),
		Model:        viper.GetString("STORYGEN_OPENAI_TTS_MODEL"),
		Voice:        viper.GetString("STORYGEN_VOICE"),
		Instructions: "Voice Affect: Fun, active, involved and engaged teacher voice reading a bedtime story to group of kids.\n\nTone: Sincere, empathetic, involved, engaged.\n\nPacing: Slow enough for kids to understand but realisticly faster when story picks up action.\n\nEmotion: Adopting to what is happening in the story.\n\nPauses: Big pause right before story chapter starts.",
		Speed:        speed,
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

			err := openaiHandler.Convert(textToSpeech, targetFile)
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
