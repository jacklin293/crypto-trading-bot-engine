package db

import (
	"time"
)

type User struct {
	Id             int64
	Uuid           string
	TelegramChatId int64
	Username       string
	ExchangeApiKey string // {"FTX":{"api_key":"", "api_secret": "", "subaccount": ""}}
	Activated      int64
	Role           int64 // 0: general user, 99: admin
	LastLoginAt    time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (db *DB) GetUserByUuid(uuid string) (*User, error) {
	var u User
	result := db.GormDB.Where("uuid = ?", uuid).First(&u)
	return &u, result.Error
}

// for API
func (db *DB) GetUserByUsername(username string) (*User, error) {
	var u User
	result := db.GormDB.Where("username = ?", username).First(&u)
	return &u, result.Error
}

// for API
func (db *DB) GetUserByUsernameByPassword(username string, password string) (*User, error) {
	var u User
	result := db.GormDB.Where("username = ? AND password = ? AND password != '' AND password IS NOT NULL AND password_expired_at > ?", username, password, time.Now()).First(&u)
	if result.Error != nil {
		return &u, result.Error
	}
	return &u, nil
}

// for API
func (db *DB) UpdateUser(uuid string, user map[string]interface{}) (int64, error) {
	result := db.GormDB.Model(User{}).Where("uuid = ?", uuid).Updates(user)
	return result.RowsAffected, result.Error
}
