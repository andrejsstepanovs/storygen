package story

import (
	"fmt"
	"regexp"
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
	Name   string `json:"name"`
	Voice  string `json:"voice"`
	Type   string `json:"type"`
	Gender string `json:"gender"`
	Size   string `json:"size"`
	Age    string `json:"age"`
}

type Story struct {
	StorySuggestion string       `json:"story_prompt"`
	Structure       Structure    `json:"structure"`
	TimePeriod      TimePeriod   `json:"time_period"`
	Length          string       `json:"length"`
	Morales         Morales      `json:"morales"`
	Protagonists    Protagonists `json:"protagonists"`
	Villain         string       `json:"villain"`
	VillainVoice    string       `json:"villain_voice"`
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

func (p *Protagonists) String() string {
	instr := make([]string, 0)
	for _, protagonist := range *p {
		txt := fmt.Sprintf(
			"# Named %q\n- Type: %s\n- Gender: %s\n- Size: %s\n- Age: %s\n- Voice: %s\n",
			protagonist.Name,
			protagonist.Type,
			protagonist.Gender,
			protagonist.Size,
			protagonist.Age,
			protagonist.Voice,
		)
		instr = append(instr, txt)
	}

	return fmt.Sprintf("# Protagonists:\n%s", strings.Join(instr, "\n"))
}

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

func trimChapterTitleFromText(c Chapter) string {
	text := c.Text
	title := c.Title

	if len(text) == 0 || len(title) == 0 {
		return strings.TrimSpace(text)
	}

	normalizedTitle := normalizeForComparison(title)
	if len(normalizedTitle) == 0 {
		return strings.TrimSpace(text)
	}

	var currentNormalizedPrefix strings.Builder
	lastWasSpace := true
	matchEndIndex := -1
	maxLengthToCheck := len(normalizedTitle) + 50
	if maxLengthToCheck > 400 {
		maxLengthToCheck = 400
	}
	if len(text) < maxLengthToCheck {
		maxLengthToCheck = len(text)
	}
	scanText := text[:maxLengthToCheck]

	for i, r := range scanText {
		originalCharLen := len(string(r))
		if unicode.IsLetter(r) || unicode.IsNumber(r) {
			currentNormalizedPrefix.WriteRune(unicode.ToLower(r))
			lastWasSpace = false
		} else if !lastWasSpace {
			currentNormalizedPrefix.WriteRune(' ')
			lastWasSpace = true
		}
		normalizedPrefixStr := strings.TrimSpace(currentNormalizedPrefix.String())

		if normalizedPrefixStr != "" && strings.HasSuffix(normalizedPrefixStr, normalizedTitle) {
			suffixStartIndex := len(normalizedPrefixStr) - len(normalizedTitle)
			if suffixStartIndex == 0 || normalizedPrefixStr[suffixStartIndex-1] == ' ' {
				matchEndIndex = i + originalCharLen
				break
			}
		}
		if len(normalizedPrefixStr) > len(normalizedTitle)+20 {
			if !strings.HasSuffix(normalizedPrefixStr, normalizedTitle) {
				break
			}
		}
	}

	if matchEndIndex != -1 {
		remainingText := text[matchEndIndex:]
		trimmedText := strings.TrimLeftFunc(remainingText, func(r rune) bool {
			return unicode.IsSpace(r) || r == ':' || r == '.' || r == '-' || r == '#' || r == '*'
		})
		return strings.TrimSpace(trimmedText)
	}

	return strings.TrimSpace(text)
}

func removeChars(text string) string {
	text = strings.Replace(text, "*", "", -1)
	text = strings.Replace(text, "#", "", -1)
	return text
}

var newlineNormalizerRegex = regexp.MustCompile(`\n{2,}`)

func (s *Story) BuildContent(chapterLabel, theEnd string) string {
	content := make([]string, 0)

	// --- Process Story Title ---
	storyTitle := removeChars(s.Title)
	storyTitle = strings.TrimPrefix(storyTitle, "Title:")
	storyTitle = strings.TrimSpace(storyTitle)
	if storyTitle != "" {
		content = append(content, storyTitle)
		content = append(content, "...")
	}

	// --- Process Chapters ---
	numChapters := len(s.Chapters)
	for i, c := range s.Chapters {
		// --- Format Chapter Header ---
		chapterTitleClean := strings.TrimSpace(removeChars(c.Title))
		var formattedHeader string
		if chapterTitleClean != "" {
			chapterTitleClean = strings.TrimRight(chapterTitleClean, ".?!:;,") + "."
			// Header now contains Chapter+Num on one line, Title on the next.
			formattedHeader = fmt.Sprintf("%s %d.\n%s", chapterLabel, c.Number, chapterTitleClean)
		} else {
			// Header only has Chapter+Num
			formattedHeader = fmt.Sprintf("%s %d.", chapterLabel, c.Number)
		}
		content = append(content, formattedHeader) // Add the header block

		// --- Process Chapter Text ---
		chapterText := trimChapterTitleFromText(c) // Already trims space
		chapterText = removeChars(chapterText)

		// Normalize consecutive newlines WITHIN the text to standard paragraph breaks (\n\n)
		chapterText = newlineNormalizerRegex.ReplaceAllString(chapterText, "\n")
		// Trim space *again* after normalization, just in case
		chapterText = strings.TrimSpace(chapterText)

		if chapterText != "" {
			content = append(content, chapterText) // Add the processed text block
		}

		// --- Add Separator Between Chapters (if needed) ---
		if i < numChapters-1 { // If NOT the last chapter
			// Add just the ellipsis line as a separator.
			content = append(content, "...")
		}
	}

	// --- Add Ending ---
	// Add the ellipsis separator before the end text.
	content = append(content, "...")
	content = append(content, theEnd)

	// Join all parts with TWO newlines "\n\n".
	// This creates ONE blank line between each element in the content slice.
	return RemoveEmojis(strings.Join(content, "\n\n"))
}
