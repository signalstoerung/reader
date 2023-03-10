package main

import (
	"errors"
	"log"

	"gorm.io/gorm"
)

// loadItems is called by rootHandler. It retrieves items from the DB (optionally filtered by abbreviation), with limit and/or offset to allow pagination)
// it modifies the slice of headlinesTemplateResult that was provided by the calling function
func loadItems(db *gorm.DB, resultSlice *[]HeadlinesItem, filter string, limit int, offset int) error {
	// had some issues with this function panicking. Bugs seem to be quashed, but leaving this in just in case.
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Recovered from panic in loadItems: %v.", r)
		}
	}()

	var items []Item
	result := db.Limit(limit).Offset(offset).Order("published_parsed desc").Where(&Item{FeedAbbr: filter}).Find(&items)
	if result.Error != nil {
		return result.Error
	}

	// I don't quite understand why RowsAffected is sometimes 1 and sometimes 0, but both return empty result slices, so catch it as an error
	// this was what caused the panic later (calling .Format on a nil result)
	if result.RowsAffected <= 1 {
		return errors.New("No headlines found.")
	}

	// shorten slice to number of items in the results
	*resultSlice = (*resultSlice)[:result.RowsAffected]

	for i, item := range items {
		if i > len(*resultSlice) {
			break
		}

		// at the very end of the result set (last page), DB seems to return an empty item
		// calling .Format on a nil object causes a panic
		if item.PublishedParsed == nil {
			break
		}
		(*resultSlice)[i].Link = item.Link
		(*resultSlice)[i].Title = item.Title
		(*resultSlice)[i].Timestamp = item.PublishedParsed.In(globalConfig.localTZ).Format("02 Jan 15:04")
		(*resultSlice)[i].FeedAbbr = item.FeedAbbr
	}
	return nil
}
