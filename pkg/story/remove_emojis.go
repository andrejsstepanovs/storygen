package story

import (
	"regexp"
	"strings"
	"unicode"
)

func RemoveEmojis(prompt string) string {
	// Remove emojis and other problematic characters for speech synthesis
	cleaned := prompt

	// Remove emojis (Unicode ranges for common emojis)
	emojiRegex := regexp.MustCompile(`[\x{1F600}-\x{1F64F}]|[\x{1F300}-\x{1F5FF}]|[\x{1F680}-\x{1F6FF}]|[\x{1F1E0}-\x{1F1FF}]|[\x{2600}-\x{26FF}]|[\x{2700}-\x{27BF}]`)
	cleaned = emojiRegex.ReplaceAllString(cleaned, "")

	// Remove additional symbol and pictograph ranges
	symbolRegex := regexp.MustCompile(`[\x{1F900}-\x{1F9FF}]|[\x{1FA70}-\x{1FAFF}]`)
	cleaned = symbolRegex.ReplaceAllString(cleaned, "")

	// Remove dingbats and misc symbols
	dingbatRegex := regexp.MustCompile(`[\x{2728}-\x{274B}]|[\x{1F004}-\x{1F0CF}]`)
	cleaned = dingbatRegex.ReplaceAllString(cleaned, "")

	// Remove variation selectors (used for emoji variations)
	variationSelectorRegex := regexp.MustCompile(`[\x{FE00}-\x{FE0F}]`)
	cleaned = variationSelectorRegex.ReplaceAllString(cleaned, "")

	// Replace zero-width characters with spaces to preserve word boundaries
	cleaned = strings.ReplaceAll(cleaned, "\u200B", " ") // zero width space
	cleaned = strings.ReplaceAll(cleaned, "\u200C", " ") // zero width non-joiner
	cleaned = strings.ReplaceAll(cleaned, "\u200D", " ") // zero width joiner
	cleaned = strings.ReplaceAll(cleaned, "\uFEFF", " ") // zero width no-break space

	// Remove any remaining characters that are likely problematic for TTS
	// This includes combining diacritical marks and other special symbols
	combiningRegex := regexp.MustCompile(`[\x{0300}-\x{036F}]|[\x{20D0}-\x{20FF}]`)
	cleaned = combiningRegex.ReplaceAllString(cleaned, "")

	// Remove excessive whitespace and normalize spacing
	var result strings.Builder
	result.Grow(len(cleaned))

	inWhitespace := false
	for _, r := range cleaned {
		if unicode.IsSpace(r) {
			if !inWhitespace {
				result.WriteRune(' ')
				inWhitespace = true
			}
		} else {
			result.WriteRune(r)
			inWhitespace = false
		}
	}

	// Trim leading/trailing whitespace
	finalText := strings.TrimSpace(result.String())

	return finalText
}
