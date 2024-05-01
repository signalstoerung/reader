package users

import "github.com/signalstoerung/reader/internal/feeds"

func SavedItemsForUser(username string) ([]feeds.Item, error) {
	user, err := UserByName(username)
	if err != nil {
		return nil, err
	}
	var items = make([]feeds.Item, 0, 20)
	err = Config.DB.Order("published_parsed desc").Model(&user).Association("SavedItems").Find(&items)
	if err != nil {
		return nil, err
	}
	return items, nil
}

func AddItemForUser(username string, itemId int) error {
	user, err := UserByName(username)
	if err != nil {
		return err
	}
	var item feeds.Item
	result := Config.DB.First(&item, itemId)
	if result.Error != nil {
		return result.Error
	}
	user.SavedItems = append(user.SavedItems, item)
	result = Config.DB.Save(&user)
	return result.Error
}

func DeleteItemForUser(username string, itemId int) error {
	user, err := UserByName(username)
	if err != nil {
		return err
	}
	var item feeds.Item
	err = Config.DB.Model(&user).Association("SavedItems").Find(&item, itemId)
	if err != nil {
		return err
	}
	err = Config.DB.Model(&user).Association("SavedItems").Delete(&item)
	if err != nil {
		return err
	}
	return nil
}
