package trigger

import (
	"errors"
	"time"

	"github.com/shopspring/decimal"
)

// trigger_type: 'limit'
type Limit struct {
	Operator string // '>=' or '<=
	Price    decimal.Decimal
}

// New limit trigger
func newLimit(data map[string]interface{}) (m *Limit, err error) {
	operator, ok := data["operator"].(string)
	if !ok {
		err = errors.New("'operator' is missing")
		return
	}
	if err = validateOperator(operator); err != nil {
		return
	}
	p, ok := data["price"].(float64)
	if !ok {
		err = errors.New("'price' is missing or not a float")
		return
	}
	price := decimal.NewFromFloat(p)

	return &Limit{
		Operator: operator,
		Price:    price,
	}, nil
}

// Get price
func (l *Limit) GetPrice(_ time.Time) decimal.Decimal {
	return l.Price
}

// Get operator
func (l *Limit) GetOperator() string {
	return l.Operator
}

// Set operator
func (l *Limit) SetOperator(operator string) {
	l.Operator = operator
}

// Readjust price
func (l *Limit) ReadjustPrice(price decimal.Decimal, _ time.Time) {
	l.Price = price
}

// Update price by percent
func (l *Limit) UpdatePriceByPercent(percent decimal.Decimal) {
	l.Price = l.Price.Mul(percent)
}

// Copy a new clone of trigger instead of passing pointer
func (l *Limit) Clone() Trigger {
	c := *l
	return &c
}
