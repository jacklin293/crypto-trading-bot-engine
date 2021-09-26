package db

import (
	"time"

	"gorm.io/datatypes"
)

type User struct {
	Id              int64
	Uuid            string
	TelegramChatId  int64
	Username        string
	ExchangeApiInfo datatypes.JSONMap // e.g. {"api_key":"...","api_secret":"...","subaccount_name":"..."}
	Activated       int64
	LastLoginAt     time.Time
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

// TODO
func (db *DB) GetUserByUuid(uuid string) (*User, error) {
	var u User
	result := db.GormDB.Where("uuid = ?", uuid).First(&u)
	return &u, result.Error
}
