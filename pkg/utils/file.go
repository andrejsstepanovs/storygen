package utils

import (
	"fmt"
	"log"
	"os"
	"regexp"
	"path"
	"strings"
)

func LoadTextFromFile(filename string) []byte {
	data, err := os.ReadFile(filename)
	if err != nil {
		log.Fatalln(err)
	}
	return data
}

// SaveTextToFile saves the given text content to a file with the specified filename.
// If the file does not exist, it will be created. If it does exist, it will be overwritten.
func SaveTextToFile(dir, filename, extension, text string) (string, error) {
	filename = fmt.Sprintf("%s.%s", SanitizeFilename(filename), extension)

	// Convert the text to a byte slice
	data := []byte(text)

    targetDir := path.Join(dir, filename)
	// Write the data to the file with 0644 permissions (read/write for owner, read for others)
	err := os.WriteFile(targetDir, data, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write to file: %v", err)
	}

	return filename, nil
}

func SanitizeFilename(filename string) string {
	filename = strings.Replace(filename, "\"", "", -1)
	filename = strings.Replace(filename, ".", "_", -1)

	// Define a regex pattern for allowed characters (alphanumeric, underscores, hyphens, and dots)
	allowedPattern := regexp.MustCompile(`[^a-zA-Z0-9_\-\.]`)

	// Replace spaces with underscores
	filename = strings.ReplaceAll(filename, " ", "_")

	// Remove all characters not allowed in the pattern
	filename = allowedPattern.ReplaceAllString(filename, "")

	if len(filename) > 150 {
		filename = filename[:150]
	}
	return filename
}
