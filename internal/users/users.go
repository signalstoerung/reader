package users

import (
	"errors"

	"github.com/signalstoerung/reader/internal/feeds"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	UserName   string
	Password   string
	Keywords   []Keyword
	SavedItems []feeds.Item `gorm:"many2many:user_saved_items"`
}

type Configuration struct {
	DB     *gorm.DB
	Secret []byte
}

var (
	ErrNotImplemented = errors.New("not implemented")
	ErrNoDBConnection = errors.New("no database connection")
	ErrNotFound       = errors.New("wrong username or password")
	Config            = Configuration{}
)

func (c *Configuration) OpenDatabase(path string) error {
	db, err := gorm.Open(sqlite.Open(path), &gorm.Config{})
	if err != nil {
		return err
	}
	db.AutoMigrate(&Keyword{})
	db.AutoMigrate(&User{})
	c.DB = db
	return nil
}

func CreateUser(username string, password string) error {
	if Config.DB == nil {
		return ErrNoDBConnection
	}
	passwordHash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	user := User{
		UserName: username,
		Password: string(passwordHash),
	}
	result := Config.DB.Create(&user)
	return result.Error
}

// returns NIL on success
func VerifyUser(username string, password string) error {
	if Config.DB == nil {
		return ErrNoDBConnection
	}
	var maybeUser User
	result := Config.DB.Where(&User{UserName: username}).First(&maybeUser)
	if result.Error != nil {
		return result.Error
	}
	return bcrypt.CompareHashAndPassword([]byte(maybeUser.Password), []byte(password))
}

func UserByName(name string) (User, error) {
	var maybeUser User
	result := Config.DB.Where(&User{UserName: name}).First(&maybeUser)
	if result.Error != nil {
		return User{}, result.Error
	}
	return maybeUser, nil
}
