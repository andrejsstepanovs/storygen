package story

import (
	"github.com/andrejsstepanovs/storygen/pkg/utils"
)

type Problem struct {
	Chapter     int      `json:"chapter_number_int"`
	ChapterName string   `json:"chapter_name"`
	Issues      []string `json:"issues_array_string"`
}

type Problems []Problem

func (p *Problem) ToJson() string {
	return utils.ToJsonStr(p)
}

func (p *Problems) ToJson() string {
	return utils.ToJsonStr(p)
}
