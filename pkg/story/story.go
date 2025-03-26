package story

import (
	"fmt"
	"strings"
	"unicode"

	"github.com/andrejsstepanovs/storygen/pkg/utils"
)

const TextChapter = "Chapter"
const TextTheEnd = "The End."

type Chapter struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	Text   string `json:"text"`
}

type Chapters []Chapter

type Structure struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type Structures []Structure

type Morales []Morale

type Morale struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type TimePeriods []TimePeriod

type TimePeriod struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}
type Protagonists []Protagonist

type Protagonist struct {
	Type   string `json:"type"`
	Gender string `json:"gender"`
	Size   string `json:"size"`
	Age    string `json:"age"`
}

type Story struct {
	StorySuggestion string       `json:"story_suggestion"`
	Structure       Structure    `json:"structure"`
	TimePeriod      TimePeriod   `json:"time_period"`
	Length          string       `json:"length"`
	Morales         Morales      `json:"morales"`
	Protagonists    Protagonists `json:"protagonists"`
	Villain         string       `json:"villain"`
	Plan            string       `json:"plan"`
	Location        string       `json:"location"`
	Summary         string       `json:"summary"`
	Chapters        Chapters     `json:"chapters"`
	Title           string       `json:"title"`
}

func NewStory() Story {
	return Story{}
}

func (s *Structures) ToJson() string {
	return utils.ToJsonStr(s)
}
func (s *Structure) ToJson() string {
	return utils.ToJsonStr(s)
}
func (m *Morales) ToJson() string {
	return utils.ToJsonStr(m)
}
func (m *Morale) ToJson() string {
	return utils.ToJsonStr(m)
}
func (s *Story) ToJson() string {
	return utils.ToJsonStr(s)
}
func (c *Chapters) ToJson() string {
	return utils.ToJsonStr(c)
}
func (c *Chapter) ToJson() string {
	return utils.ToJsonStr(c)
}
func (t *TimePeriods) ToJson() string {
	return utils.ToJsonStr(t)
}
func (t *TimePeriod) ToJson() string {
	return utils.ToJsonStr(t)
}
func (p *Protagonists) ToJson() string {
	return utils.ToJsonStr(p)
}
func (p *Protagonist) ToJson() string {
	return utils.ToJsonStr(p)
}

// normalizeForComparison converts a string to lowercase, replaces punctuation
// and whitespace sequences with single spaces, and trims leading/trailing space.
func normalizeForComparison(s string) string {
	var sb strings.Builder
	sb.Grow(len(s))
	lastWasSpace := true

	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			sb.WriteRune(unicode.ToLower(r))
			lastWasSpace = false
		} else if !lastWasSpace {
			sb.WriteRune(' ')
			lastWasSpace = true
		}
	}
	return strings.TrimSpace(sb.String())
}

// trimChapterTitleFromText attempts to remove the chapter title if it appears
// (with potential variations) at the beginning of the chapter text.
// It now checks if the normalized start of the text *ends with* the normalized title.
func trimChapterTitleFromText(c Chapter) string {
	text := c.Text
	title := c.Title

	// --- Basic Checks ---
	if len(text) == 0 || len(title) == 0 {
		return text
	}

	// --- Normalization ---
	normalizedTitle := normalizeForComparison(title)
	if len(normalizedTitle) == 0 {
		return text // Title consists only of punctuation/space
	}

	// --- Iterative Comparison ---
	var currentNormalizedPrefix strings.Builder
	lastWasSpace := true
	matchEndIndex := -1 // Index in *original* text where the match ends

	// Limit iteration length (adjust if titles can be very long duplicates)
	// Increased slightly to ensure prefix + title fits.
	maxLengthToCheck := len(normalizedTitle) + 50
	if maxLengthToCheck > 400 { // Absolute cap
		maxLengthToCheck = 400
	}
	if len(text) < maxLengthToCheck {
		maxLengthToCheck = len(text)
	}
	scanText := text[:maxLengthToCheck]

	for i, r := range scanText {
		// Build the normalized version of the text prefix incrementally
		originalCharLen := len(string(r)) // Handle multi-byte runes correctly for index advancement

		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			currentNormalizedPrefix.WriteRune(unicode.ToLower(r))
			lastWasSpace = false
		} else if !lastWasSpace {
			currentNormalizedPrefix.WriteRune(' ')
			lastWasSpace = true
		}

		// Get the current normalized prefix string (trimmed for comparison stability)
		normalizedPrefixStr := strings.TrimSpace(currentNormalizedPrefix.String())

		// --- MODIFIED COMPARISON ---
		// Check if the normalized text prefix *ends with* the normalized title.
		// This allows for prefixes like "Chapter N:" in the text that aren't in c.Title.
		if normalizedPrefixStr != "" && strings.HasSuffix(normalizedPrefixStr, normalizedTitle) {
			// Ensure it's a whole word/phrase match at the end, not a substring mid-word.
			// Check if it's the *entire* prefix string OR if the char before the suffix start is a space.
			suffixStartIndex := len(normalizedPrefixStr) - len(normalizedTitle)
			if suffixStartIndex == 0 || normalizedPrefixStr[suffixStartIndex-1] == ' ' {
				// Found a potential match ending at index 'i' in the original text.
				matchEndIndex = i + originalCharLen // Point *after* the current rune
				// Don't break immediately. Find the *shortest* valid prefix
				// that ends with the title. Example: Title "Plan". Text "Plan: The Plan..."
				// We want to remove "Plan:", not just "Plan".
				// However, for "Chapter 1: Title", Title="Title", we *do* want the first match.
				// Let's stick with breaking on first match for simplicity now, it covers the main case.
				break
			}
		}
		// --- END MODIFIED COMPARISON ---

		// Optimization: If the normalized prefix is significantly longer than the title
		// and we haven't found a match ending in the title, stop.
		// The check needs len(normalizedPrefixStr) because prefixes can add length.
		if len(normalizedPrefixStr) > len(normalizedTitle)+20 { // Allow ~ "Chapter XX: " prefix len
			// If the prefix *doesn't* end with the title by now, it's unlikely to.
			if !strings.HasSuffix(normalizedPrefixStr, normalizedTitle) {
				break
			}
		}
	}

	// --- Result ---
	if matchEndIndex != -1 {
		// Found a match ending at matchEndIndex in the original text.
		remainingText := text[matchEndIndex:]
		trimmedText := strings.TrimLeftFunc(remainingText, func(r rune) bool {
			// Trim leading whitespace AND common punctuation separators
			return unicode.IsSpace(r) || r == ':' || r == '.' || r == '-' || r == '#' || r == '*'
		})
		return trimmedText
	}

	// No match found
	return text
}

// removeChars - unchanged
func removeChars(text string) string {
	text = strings.Replace(text, "*", "", -1)
	text = strings.Replace(text, "#", "", -1)
	return text
}

// BuildContent - Minor cleanup for consistency, logic unchanged
func (s *Story) BuildContent(chapterLabel, theEnd string) string {
	content := make([]string, 0)

	// Process story title
	storyTitle := removeChars(s.Title)
	storyTitle = strings.TrimPrefix(storyTitle, "Title:")
	storyTitle = strings.TrimSpace(storyTitle)
	if storyTitle != "" { // Avoid adding empty title line
		content = append(content, storyTitle)
		content = append(content, "\n\n...\n\n")
	}

	// Process chapters
	for _, c := range s.Chapters {
		// Format chapter header
		chapterTitleClean := strings.TrimSpace(removeChars(c.Title))
		// Handle potential empty title after cleaning
		if chapterTitleClean != "" {
			chapterTitleClean = strings.TrimRight(chapterTitleClean, ".?!:;,") + "."
			content = append(content, fmt.Sprintf("%s %d.\n%s\n...\n", chapterLabel, c.Number, chapterTitleClean))
		} else {
			// If title is empty, just use the chapter number
			content = append(content, fmt.Sprintf("%s %d.\n...\n", chapterLabel, c.Number))
		}

		// Get chapter text, attempting to trim duplicate title
		chapterText := trimChapterTitleFromText(c)
		chapterText = removeChars(chapterText)       // Apply general cleanup
		chapterText = strings.TrimSpace(chapterText) // Trim overall chapter text too

		if chapterText != "" { // Avoid adding empty text blocks
			content = append(content, chapterText)
			content = append(content, "\n\n...\n\n")
		}
	}

	// Add ending
	content = append(content, theEnd)

	return strings.Join(content, "\n")
}
