package users

import (
	"strings"

	"gorm.io/gorm"
)

type Mode string
type KeywordList []Keyword

const (
	HighlightMode Mode = "KeywordHighlight"
	SuppressMode  Mode = "KeywordSuppress"
	DoNothingMode Mode = "KeywordIgnore"
)

type Keyword struct {
	gorm.Model
	Mode       Mode   // the mode (highlight or suppress)
	Text       string // the keyword that will trigger
	Annotation string // an optional note explaining this keyword
	UserID     uint   // reference back to user
}

// Compares a the search string provided to the keyword list and returns a Mode (highlight/suppress/do nothing)
func (kl KeywordList) Match(search string) Mode {
	for _, k := range kl {
		if strings.Contains(search, k.Text) {
			return k.Mode
		}
	}
	return DoNothingMode
}
