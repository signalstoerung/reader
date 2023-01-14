package main

import (
	"fmt"
	"gorm.io/gorm"
)


// loadItemsFromDB is called by rootHandler. It retrieves items from the DB (optionally filtered by abbreviation), with limit and/or offset to allow pagination).
// TODO: Delete, obsolete
func loadItemsFromDB(db *gorm.DB, resultSlice *[]string, filter string, limit int, offset int) error {
	var items []Item
	result := db.Limit(limit).Offset(offset).Order("published_parsed desc").Where(&Item{FeedAbbr: filter}).Find(&items)
	if result.Error != nil {
		return result.Error
	}
	for _,item := range items {
		*resultSlice = append(*resultSlice, fmt.Sprintf("<div><a href=\"%v\" target=\"_blank\">%v %v-%v</a></div>\n",item.Link,item.PublishedParsed.Format("02 Jan 15:04"),item.FeedAbbr,item.Title))
	}
	return nil
}

// loadItems is called by rootHandler. It retrieves items from the DB (optionally filtered by abbreviation), with limit and/or offset to allow pagination)
// it modifies the slice of headlinesTemplateResult that was provided by the calling function
func loadItems(db *gorm.DB, resultSlice *[]HeadlinesItem, filter string, limit int, offset int) error {
	var items []Item
	result := db.Limit(limit).Offset(offset).Order("published_parsed desc").Where(&Item{FeedAbbr: filter}).Find(&items)
	if result.Error != nil {
		return result.Error
	}
	for i, item := range items {
		if i > len(*resultSlice) {
			break
		}
		(*resultSlice)[i].Link = item.Link
		(*resultSlice)[i].Title = item.Title
		(*resultSlice)[i].Timestamp = item.PublishedParsed.Format("02 Jan 15:04")
		(*resultSlice)[i].FeedAbbr = item.FeedAbbr
	}
	return nil
}