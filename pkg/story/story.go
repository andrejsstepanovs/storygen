package story

import (
	"fmt"
	"strings"

	"github.com/andrejsstepanovs/storygen/pkg/utils"
)

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

func trimChapterTitleFromText(c Chapter) string {
	text := c.Text
	var searchArea string
	if len(text) >= 100 {
		searchArea = text[:100]
	} else {
		searchArea = text
	}

	// Check if the title is in the search area
	if strings.Contains(searchArea, c.Title) {
		// Find the position right after the title
		titlePos := strings.Index(text, c.Title)
		if titlePos != -1 {
			endTitlePos := titlePos + len(c.Title)
			// Trim leading whitespace after the title
			trimmedText := strings.TrimSpace(text[endTitlePos:])
			return trimmedText
		}
	}
	return text
}

func removeChars(text string) string {
	text = strings.Replace(text, "*", "", -1)
	text = strings.Replace(text, "#", "", -1)
	return text
}

func (s *Story) BuildContent(chapter string) string {
	content := make([]string, 0)

	title := removeChars(s.Title)
	title = strings.TrimLeft(title, "Title: ")
	content = append(content, title)
	content = append(content, "")
	for _, c := range s.Chapters {
		content = append(content, fmt.Sprintf("%s %d. %s", chapter, c.Number, strings.TrimRight(removeChars(c.Title), ".")+"."))
		content = append(content, removeChars(trimChapterTitleFromText(c)))
		content = append(content, "\n\n\n")
	}

	return strings.Join(content, "\n")
}
