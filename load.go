package main

import (
	"fmt"
	"gorm.io/gorm"
)

func loadItemsFromDB(db *gorm.DB, resultSlice *[]string, filter string, limit int, offset int) error {
	var items []Item
	result := db.Limit(limit).Offset(offset).Order("published_parsed desc").Where(&Item{FeedAbbr: filter}).Find(&items)
	if result.Error != nil {
		return result.Error
	}
	for _,item := range items {
		*resultSlice = append(*resultSlice, fmt.Sprintf("<div><a href=\"%v\">%v %v-%v</a></div>\n",item.Link,item.PublishedParsed.Format("02 Jan 15:04"),item.FeedAbbr,item.Title))
	}
	return nil
}