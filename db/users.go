package db

import "time"

type User struct {
	Id             int64
	Uuid           string
	TelegramChatId string
	Username       string
	ApiKey         string
	ApiSecret      string
	Activated      int64
	LastLoginAt    time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
