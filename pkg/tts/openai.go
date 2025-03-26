package tts

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"strings"
	"unicode"

	"github.com/sashabaranov/go-openai"
	"github.com/spf13/viper"
)

func splitByChapters(text string) []string {
	// Regex to find chapter markers for Chapter 2 and higher.
	// (?:[2-9]|[1-9]\d+) matches 2-9 or any number 10 or greater.
	re := regexp.MustCompile(`\n(?:...\n)?\s*Chapter (?:[2-9]|[1-9]\d+)\.`)

	// Find the start indices of all Chapter 2+ markers.
	indices := re.FindAllStringIndex(text, -1)

	// If no markers for Chapter 2+ are found, the entire text is one chapter.
	if len(indices) == 0 {
		trimmedText := strings.TrimSpace(text)
		if trimmedText == "" {
			return []string{}
		}
		return []string{trimmedText}
	}

	chapters := []string{}
	startIdx := 0 // Start of the current chapter slice

	// Iterate through the found indices (which mark the START of Chapter 2, 3, etc.)
	for _, indexPair := range indices {
		// Get the text from the previous start up to the beginning of the current Chapter 2+ marker.
		chapterText := text[startIdx:indexPair[0]]
		trimmedChapter := strings.TrimSpace(chapterText)
		if trimmedChapter != "" {
			chapters = append(chapters, trimmedChapter)
		}
		// Update the start index for the next segment to be the beginning of the current marker.
		startIdx = indexPair[0]
	}

	// Add the final segment (from the start of the last Chapter 2+ marker to the end of the text).
	lastChapterText := text[startIdx:]
	trimmedLastChapter := strings.TrimSpace(lastChapterText)
	if trimmedLastChapter != "" {
		chapters = append(chapters, trimmedLastChapter)
	}

	// Clean up any potential empty strings, although TrimSpace should handle most cases.
	finalChapters := make([]string, 0, len(chapters))
	for _, ch := range chapters {
		if ch != "" {
			finalChapters = append(finalChapters, ch)
		}
	}

	return finalChapters
}
func TextToSpeech(voice string, dir, outputFilePath, textToSpeech, inbetweenFile string) error {
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

		chunks := chunkText(chapterText, 2000) // Adjust chunk size as needed
		for k, chunk := range chunks {
			trimmedChunk := strings.TrimSpace(chunk)
			trimmedChunk = strings.TrimLeft(trimmedChunk, "...")
			trimmedChunk = strings.TrimSpace(trimmedChunk)
			if trimmedChunk == "" {
				continue
			}

			file := fmt.Sprintf("%d_%d_%s", n, k, outputFilePath) // n=segment index, k=chunk index
			targetFile := path.Join(dir, file)

			// fmt.Println("#################")
			// fmt.Println(trimmedChunk)

			err := openaiFile(voice, targetFile, trimmedChunk)
			if err != nil {
				return fmt.Errorf("chunk processing failed for segment %d chunk %d: %w", n, k, err)
			}
			files = append(files, targetFile)
		}
	}

	if len(files) == 0 {
		fmt.Println("No audio files were generated.")
		return fmt.Errorf("no audio files generated, cannot join")
	}

	fmt.Printf("\nJoining %d audio segments...\n", len(files))
	err := JoinMp3Files(files, path.Join(dir, outputFilePath), inbetweenFile)
	if err != nil {
		return fmt.Errorf("failed to join MP3 files: %w", err)
	}

	fmt.Println("\nCleaning up temporary files...")
	err = Remove(files)
	if err != nil {
		fmt.Printf("Warning: Failed to remove temporary files: %v\n", err)
	}

	fmt.Println("\nTextToSpeech process completed successfully.")
	return nil
}

func openaiFile(voice string, outputFilePath, textToSpeech string) error {
	speed := viper.GetFloat64("STORYGEN_SPEECH_SPEED")
	if speed == 0 {
		speed = 0.9
	}
	//instructions := "Voice Affect: Fun, active, involved and engaged teacher voice reading a bedtime story to group of kids.\n\nTone: Sincere, empathetic, involved, engaged.\n\nPacing: Slow enough for kids to understand but realisticly faster when story picks up action.\n\nEmotion: Adopting to what is happening in the story.\n\nPauses: Big pause right before story chapter starts."
	request := openai.CreateSpeechRequest{
		Model:          "gpt-4o-mini-tts",
		ResponseFormat: openai.SpeechResponseFormatMp3,
		Voice:          openai.SpeechVoice(voice),
		Input:          textToSpeech,
		//Instructions:   instructions,
		Speed: speed,
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

// isQuote checks if a rune is a quotation mark (supporting various types).
// We focus on closing quotes for boundary extension, but define generally.
func isQuote(r rune) bool {
	// Add more quote types if needed (e.g., « »)
	return r == '"' || r == '\'' || r == '”' || r == '’'
}

// isSentenceEnd checks if a rune is a common sentence-ending punctuation mark.
func isSentenceEnd(r rune) bool {
	return r == '.' || r == '!' || r == '?'
}

// isClosingQuote specifically checks for common closing quotation marks.
// Add more quote types if needed (e.g., « »).
func isClosingQuote(r rune) bool {
	// Adjust based on the actual quotes used in your text.
	// Right single/double quotes are common closers. Neutral quotes can be openers/closers.
	return r == '"' || r == '\'' || r == '”' || r == '’'
}

// chunkText splits text into chunks of a maximum size, trying to respect sentence
// boundaries and quotation marks followed by whitespace.
func chunkText(text string, chunkSize int) []string {
	if chunkSize <= 0 {
		return nil // Or []string{} based on desired behavior
	}
	if text == "" {
		return []string{}
	}

	var chunks []string
	runes := []rune(text)
	length := len(runes)
	startIndex := 0 // Start index of the current chunk

	for startIndex < length {
		// Determine the potential maximum end index for this chunk
		// Use built-in min if Go 1.21+, otherwise use the helper above
		endIndex := min(startIndex+chunkSize, length) // Use built-in min (Go 1.21+)

		// If the potential chunk extends to the end of the text, take it all
		if endIndex == length {
			chunkRunes := runes[startIndex:length]
			trimmedChunk := strings.TrimSpace(string(chunkRunes))
			if len(trimmedChunk) > 0 {
				chunks = append(chunks, trimmedChunk)
			}
			break // Finished
		}

		// Not at the end, so look for the best split point <= endIndex
		// Default split point is the hard limit endIndex
		splitPoint := endIndex
		foundGoodSplit := false

		// Scan backwards from endIndex - 1 to find the best boundary
		for k := endIndex - 1; k >= startIndex; k-- {
			// Priority 1: Sentence Boundary
			if isSentenceEnd(runes[k]) {
				potentialBoundary := k + 1 // Point *after* punctuation

				// Scan forward from boundary to include trailing quotes and whitespace
				scanIdx := potentialBoundary
				// Skip closing quotes
				for scanIdx < length && isClosingQuote(runes[scanIdx]) {
					scanIdx++
				}
				// Skip whitespace (includes spaces, newlines, etc.)
				for scanIdx < length && unicode.IsSpace(runes[scanIdx]) {
					scanIdx++
				}

				// If this natural boundary point (start of next meaningful content)
				// is within our allowed chunk size (<= endIndex), use it.
				if scanIdx <= endIndex {
					splitPoint = scanIdx // This is where the *next* chunk should start
					foundGoodSplit = true
					break // Found the best possible split (sentence end)
				}
				// If scanIdx > endIndex, this sentence boundary and its followers are too long.
				// Continue scanning backwards for an earlier potential split point.
			}

			// Priority 2: Whitespace Boundary (if no suitable sentence end found yet)
			// We only consider this if we haven't already found a good sentence split.
			// We look for the *last* whitespace character before endIndex.
			if !foundGoodSplit && unicode.IsSpace(runes[k]) {
				// This is the *last* whitespace char encountered scanning backwards.
				potentialBoundary := k + 1 // Point *after* this whitespace

				// Scan forward to skip any subsequent whitespace
				scanIdx := potentialBoundary
				for scanIdx < length && unicode.IsSpace(runes[scanIdx]) {
					scanIdx++
				}

				// If this whitespace boundary fits within the limit, use it as a fallback.
				// We only need the *last* suitable whitespace, so store the first one we find
				// scanning backwards that fits.
				if scanIdx <= endIndex {
					splitPoint = scanIdx // Use the point after all spaces
					// Don't break yet. Keep scanning backwards in case there's a
					// *sentence boundary* earlier that also fits. Sentence boundary takes precedence.
					// However, since we store the *last* fitting whitespace found,
					// this will be our fallback if no sentence boundary is found later.
					// Let's refine: If we find a whitespace, record it and continue.
					// If we later find a fitting sentence boundary, it will overwrite this.
					// So, we just need to ensure we take the *first* fitting whitespace found *backwards*.

					// Let's correct the logic: record the whitespace split and keep going.
					// If a sentence split is found later (earlier in text), it will take precedence.
					if splitPoint == endIndex { // Only update if we haven't found a better (whitespace) split yet
						splitPoint = scanIdx
					}
				} else if potentialBoundary <= endIndex && splitPoint == endIndex {
					// Fallback: if skipping spaces went too far, but the point right after
					// the *first* space fits, use that as a last resort whitespace split.
					splitPoint = potentialBoundary
				}
			}
		} // End backward scan

		// If after scanning, the splitPoint hasn't moved forward from startIndex,
		// and we are not at the end, force split at endIndex to ensure progress.
		// This handles cases like very long words exceeding chunkSize.
		if splitPoint <= startIndex && startIndex < length {
			splitPoint = endIndex
			// Additional safeguard: ensure splitPoint actually advances
			if splitPoint <= startIndex {
				splitPoint = min(startIndex+1, length) // Force advance by at least 1 if possible
				if splitPoint <= startIndex {          // If still stuck (e.g. length 0 or error)
					break // Avoid infinite loop
				}
			}
		}

		// Extract the chunk up to the determined splitPoint
		actualEndIndex := splitPoint
		if actualEndIndex <= startIndex {
			// Should not happen with safeguards above, but break defensively
			break
		}
		chunkRunes := runes[startIndex:actualEndIndex]

		// Trim whitespace from the *extracted* chunk for clean output
		trimmedChunk := strings.TrimSpace(string(chunkRunes))
		if len(trimmedChunk) > 0 {
			chunks = append(chunks, trimmedChunk)
		}

		// Update startIndex for the next iteration. It should be the exact point
		// where the current chunk ended (before trimming).
		startIndex = actualEndIndex
	}

	return chunks
}
