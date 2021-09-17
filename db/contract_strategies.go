package db

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

type ContractStrategy struct {
	Id                    int64
	Uuid                  string
	UserUuid              string
	Symbol                string // e.g. BTC-PERP
	Cost                  decimal.Decimal
	ContractDirection     int64 // 1: long   0: short
	ContractParams        datatypes.JSONMap
	Enabled               int64  // 1: enabled   0: disabled
	PositionStatus        int64  // 2: unknown   1: opened   0: closed
	Exchange              string // e.g. FTX
	ExchangeOrdersDetails datatypes.JSONMap
	LastPositionAt        time.Time
	CreatedAt             time.Time
	UpdatedAt             time.Time
}

// TODO Loop with LIMIT until no more
func (db *DB) GetEnabledContractStrategies() ([]ContractStrategy, int64, error) {
	var css []ContractStrategy
	result := db.GormDB.Where("enabled = 1").Find(&css)
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return css, 0, result.Error
	}
	return css, result.RowsAffected, result.Error
}
