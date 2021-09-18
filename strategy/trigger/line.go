package trigger

import (
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// trigger_type: 'line'
type Line struct {
	TriggerType string          `json:"trigger_type"`
	Operator    string          `json:"operator"` // '>=' or '<='
	Time1       time.Time       `json:"time_1"`
	Price1      decimal.Decimal `json:"price_1"`
	Time2       time.Time       `json:"time_2"`
	Price2      decimal.Decimal `json:"price_2"`
}

// New line trigger
func newLine(data map[string]interface{}) (l *Line, err error) {
	operator, ok := data["operator"].(string)
	if !ok {
		err = errors.New("'operator' is missing")
		return
	}
	if err = validateOperator(operator); err != nil {
		return
	}

	// price 1
	p1, ok := data["price_1"].(string)
	if !ok {
		err = errors.New("'price_1' is missing or not string")
		return
	}
	price1, err := decimal.NewFromString(p1)
	if err != nil {
		err = errors.New("'price_1' isn't a stringified number")
		return
	}

	// price 2
	p2, ok := data["price_2"].(string)
	if !ok {
		err = errors.New("'price_2' is missing or not string")
		return
	}
	price2, err := decimal.NewFromString(p2)
	if err != nil {
		err = errors.New("'price_2' isn't a stringified number")
		return
	}

	// time 1
	t1, ok := data["time_1"].(string)
	if !ok {
		err = errors.New("'time_1' is missing")
		return
	}
	time1, err := time.Parse(time.RFC3339, t1)
	if err != nil {
		err = fmt.Errorf("failed to parse 'time_1', err: %v", err)
		return
	}

	// time 2
	t2, ok := data["time_2"].(string)
	if !ok {
		err = errors.New("'time_2' is missing")
		return
	}
	time2, err := time.Parse(time.RFC3339, t2)
	if err != nil {
		err = fmt.Errorf("failed to parse 'time_2', err: %v", err)
		return
	}

	// time2 should be later than time1
	if time2.Before(time1) {
		err = fmt.Errorf("time_1 should be eariler than time_2")
		return
	}

	return &Line{
		TriggerType: "line",
		Operator:    operator,
		Time1:       time1,
		Price1:      price1,
		Time2:       time2,
		Price2:      price2,
	}, nil
}

// get price
func (l *Line) GetPrice(t time.Time) decimal.Decimal {
	if t.Equal(l.Time1) {
		return l.Price1
	}

	if t.Equal(l.Time2) {
		return l.Price2
	}

	lineTimeLength := decimal.NewFromFloat(l.Time2.Sub(l.Time1).Seconds())
	linePriceLength := l.Price2.Sub(l.Price1)

	// time point is before time 1
	if t.Before(l.Time1) {
		timeLength := decimal.NewFromFloat(l.Time1.Sub(t).Seconds())
		timeLengthPercent := timeLength.Div(lineTimeLength)
		return l.Price1.Sub(linePriceLength.Mul(timeLengthPercent))
	}

	// time point is during time period
	if t.After(l.Time1) && t.Before(l.Time2) {
		timeLength := decimal.NewFromFloat(t.Sub(l.Time1).Seconds())
		timeLengthPercent := timeLength.Div(lineTimeLength)
		return l.Price1.Add(linePriceLength.Mul(timeLengthPercent))
	}

	// time point is after time 2
	timeLength := decimal.NewFromFloat(t.Sub(l.Time2).Seconds())
	timeLengthPercent := timeLength.Div(lineTimeLength)
	return l.Price2.Add(linePriceLength.Mul(timeLengthPercent))
}

// Get operator
func (l *Line) GetOperator() string {
	return l.Operator
}

// Set operator
func (l *Line) SetOperator(operator string) {
	l.Operator = operator
}

// Readjust price
func (l *Line) ReadjustPrice(p2 decimal.Decimal, t2 time.Time) {
	l.Time2 = t2
	l.Price2 = p2
}

// Update price by percent
func (l *Line) UpdatePriceByPercent(percent decimal.Decimal) {
	l.Price1 = l.Price1.Mul(percent)
	l.Price2 = l.Price2.Mul(percent)
}

// Copy a new clone of trigger instead of passing pointer
func (l *Line) Clone() Trigger {
	c := *l
	return &c
}
