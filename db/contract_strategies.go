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
	Margin                decimal.Decimal
	Side                  int64 // 0: short  1: long
	Params                datatypes.JSONMap
	Enabled               int64  // 0: disabled  1: enabled
	PositionStatus        int64  // 0: closed  1: opened  2: unknown
	Exchange              string // e.g. FTX
	ExchangeOrdersDetails datatypes.JSONMap
	Comment               string
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

// TODO Loop with LIMIT until no more
func (db *DB) GetNonClosedContractStrategiesBySymbol(userUuid string, symbol string, uuid string) ([]ContractStrategy, int64, error) {
	var css []ContractStrategy
	result := db.GormDB.Where("user_uuid = ? AND position_status != 0 AND symbol = ? AND uuid != ?", userUuid, symbol, uuid).Find(&css)
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return css, 0, result.Error
	}
	return css, result.RowsAffected, result.Error
}

// NOTE Struct doesn't support 0 value, use map instead
func (db *DB) UpdateContractStrategy(uuid string, contractStrategy map[string]interface{}) (int64, error) {
	result := db.GormDB.Model(ContractStrategy{}).Where("uuid = ?", uuid).Updates(contractStrategy)
	return result.RowsAffected, result.Error
}

func (db *DB) GetContractStrategyByUuid(uuid string) (*ContractStrategy, error) {
	var s ContractStrategy
	result := db.GormDB.Where("uuid = ?", uuid).First(&s)
	return &s, result.Error
}

// for API
func (db *DB) GetContractStrategiesByUser(userUuid string) ([]ContractStrategy, int64, error) {
	var css []ContractStrategy
	result := db.GormDB.Where("user_uuid = ?", userUuid).Order("position_status DESC, enabled DESC").Find(&css)
	if result.Error != nil && !errors.Is(result.Error, gorm.ErrRecordNotFound) {
		return css, 0, result.Error
	}
	return css, result.RowsAffected, result.Error
}

// for API

func (db *DB) GetContractStrategyByUuidByUser(uuid string, userUuid string) (*ContractStrategy, error) {
	var s ContractStrategy
	result := db.GormDB.Where("uuid = ? AND user_uuid = ?", uuid, userUuid).First(&s)
	return &s, result.Error
}

// for API
func (db *DB) CreateContractStrategy(contractStrategy ContractStrategy) (int64, int64, error) {
	result := db.GormDB.Model(ContractStrategy{}).Create(&contractStrategy)
	return contractStrategy.Id, result.RowsAffected, result.Error
}
