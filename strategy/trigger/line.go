package trigger

import (
	"errors"
	"fmt"
	"time"

	"github.com/shopspring/decimal"
)

// trigger_type: 'line'
type Line struct {
	Operator string // '>=' or '<='
	Time1    time.Time
	Price1   decimal.Decimal
	Time2    time.Time
	Price2   decimal.Decimal
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
	p1, ok := data["price_1"].(float64)
	if !ok {
		err = errors.New("'price_1' is missing or not a float")
		return
	}
	price1 := decimal.NewFromFloat(p1)

	// price 2
	p2, ok := data["price_2"].(float64)
	if !ok {
		err = errors.New("'price_2' is missing or not a float")
		return
	}
	price2 := decimal.NewFromFloat(p2)

	// time 1
	t1, ok := data["time_1"].(string)
	if !ok {
		err = errors.New("'time_1' is missing")
		return
	}
	time1, err := time.Parse("2006-01-02 15:04:05", t1)
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
	time2, err := time.Parse("2006-01-02 15:04:05", t2)
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
		Operator: operator,
		Time1:    time1,
		Price1:   price1,
		Time2:    time2,
		Price2:   price2,
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
