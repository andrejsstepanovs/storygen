package story

import "github.com/andrejsstepanovs/storygen/pkg/utils"

type Problem struct {
	Chapter     int      `json:"chapter_number_int"`
	ChapterName string   `json:"chapter_name"`
	Issues      []string `json:"issues_array_string"`
}

type Problems []Problem

type Suggestions []Suggestion

type Suggestion struct {
	Chapter     int      `json:"chapter_number_int"`
	ChapterName string   `json:"chapter_name"`
	Suggestions []string `json:"suggestions_array_string"`
}

func (p *Problem) ToJson() string {
	return utils.ToJsonStr(p)
}

func (p *Problems) ToJson() string {
	return utils.ToJsonStr(p)
}

func (p *Suggestions) ToJson() string {
	return utils.ToJsonStr(p)
}

func (p *Suggestion) ToJson() string {
	return utils.ToJsonStr(p)
}

func (p *Suggestions) Count() int {
	count := 0
	for _, suggestion := range *p {
		count += len(suggestion.Suggestions)
	}
	return count
}
