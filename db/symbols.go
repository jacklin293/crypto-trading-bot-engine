package db

import (
	"errors"
	"time"

	"gorm.io/gorm"
)

type Symbol struct {
	Id         int64
	MarketType int64  // 1: spot   0: contract
	Exchange   string // Exchange name e.g. FTX
	Name       string // Symbol name e.g. BTC-PERP
	Enabled    int64  // 1: enabled   0: disabled
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

// TODO Filter by exchange
func (db *DB) GetEnabledSymbols() ([]Symbol, int64, error) {
	var ss []Symbol
	result := db.GormDB.Where("enabled = 1").Order("created_at DESC").Find(&ss)
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return ss, 0, result.Error
	}
	return ss, result.RowsAffected, result.Error
}
