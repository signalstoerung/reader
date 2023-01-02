package main

import (
	"fmt"
	"gorm.io/gorm"
)

func loadItemsFromDB(db *gorm.DB, limit int, offset int) error {
	var items []Item
	result := db.Limit(limit).Offset(offset).Order("published_parsed desc").Find(&items)
	if result.Error != nil {
		return result.Error
	}
	for _,item := range items {
		fmt.Printf("<div><a href=\"%v\">%v -- %v</a></div>\n",item.Link,item.PublishedParsed.Format("02 Jan 15:04"),item.Title)
	}
	return nil
}