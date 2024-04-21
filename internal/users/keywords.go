package users

import (
	"log"
	"strings"
	"time"
)

type KeywordMode string
type KeywordList []Keyword

const (
	HighlightMode KeywordMode = "KeywordHighlight"
	SuppressMode  KeywordMode = "KeywordSuppress"
	DoNothingMode KeywordMode = "KeywordIgnore"
)

type Keyword struct {
	ID         uint `gorm:"primaryKey"`
	CreatedAt  time.Time
	UpdatedAt  time.Time
	Mode       KeywordMode // the mode (highlight or suppress)
	Text       string      // the keyword that will trigger
	Annotation string      // an optional note explaining this keyword
	UserID     uint        // reference back to user
}

// Compares a the search string provided to the keyword list and returns a Mode (highlight/suppress/do nothing)
func (kl KeywordList) Match(search string) KeywordMode {
	for _, k := range kl {
		if strings.Contains(strings.ToLower(search), strings.ToLower(k.Text)) {
			return k.Mode
		}
	}
	return DoNothingMode
}

func KeywordsForUser(name string) (KeywordList, error) {
	if Config.DB == nil {
		return nil, ErrNoDBConnection
	}
	var user User
	result := Config.DB.Preload("Keywords").Where(&User{UserName: name}).First(&user)
	if result.Error != nil {
		return nil, result.Error
	}
	return user.Keywords, nil
}

func AddKeywordForUser(keyword Keyword, username string) error {
	if Config.DB == nil {
		return ErrNoDBConnection
	}
	log.Printf("Add keyword %v for user %v", keyword.Text, username)
	var user User
	result := Config.DB.Where(&User{UserName: username}).First(&user)
	if result.Error != nil {
		return result.Error
	}
	user.Keywords = append(user.Keywords, keyword)
	result = Config.DB.Save(user)
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func DeleteKeywordForUser(keywordID uint, username string) error {
	if Config.DB == nil {
		return ErrNoDBConnection
	}
	log.Printf("Delete keyword id %v for user %v", keywordID, username)
	result := Config.DB.Delete(&Keyword{}, keywordID)
	if result.Error != nil {
		return result.Error
	}
	return nil
}
