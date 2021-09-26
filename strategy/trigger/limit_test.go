package trigger

import (
	"reflect"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestNewLimit(t *testing.T) {
	testcases := []struct {
		title         string
		params        map[string]interface{}
		expectedError bool
	}{
		{
			title: "valid params",
			params: map[string]interface{}{
				"operator": ">=",
				"price":    "333",
			},
			expectedError: false,
		},
		{
			title: "missing operator",
			params: map[string]interface{}{
				"price": "333.444",
			},
			expectedError: true,
		},
		{
			title: "missing price",
			params: map[string]interface{}{
				"operator": ">=",
			},
			expectedError: true,
		},
		{
			title: "wrong type of price",
			params: map[string]interface{}{
				"operator": ">=",
				"price":    333,
			},
			expectedError: true,
		},
	}

	for _, tc := range testcases {
		_, err := newLimit(tc.params)
		hasError := (err != nil)
		if tc.expectedError != hasError {
			t.Errorf("TestNewLimit case '%s' - expect '%t', but got '%t'", tc.title, tc.expectedError, hasError)
		}
	}
}

func TestLimitGetPrice(t *testing.T) {
	testcases := []struct {
		title            string
		trigger          Trigger
		expectedOperator string
		expectedPrice    decimal.Decimal
	}{
		{
			title: "trigger_type: limit",
			trigger: &Limit{
				Operator: ">=",
				Price:    decimal.NewFromFloat(30144.542314410480331391),
			},
			expectedPrice: decimal.NewFromFloat(30144.542314410480331391),
		},
	}

	for _, tc := range testcases {
		p := tc.trigger.GetPrice(time.Now()) //time doesn't matter
		if tc.expectedPrice.StringFixed(8) != p.StringFixed(8) {
			t.Errorf("TestLimitGetPrice case '%s' - expect '%s', but got '%s'", tc.title, tc.expectedPrice.String(), p.String())
		}
	}
}

func TestLimitGetOperator(t *testing.T) {
	testcases := []struct {
		title            string
		trigger          Trigger
		expectedOperator string
		expectedPrice    decimal.Decimal
	}{
		{
			title: "trigger_type: limit",
			trigger: &Limit{
				Operator: ">=",
				Price:    decimal.NewFromFloat(30144.542314410480331391),
			},
			expectedOperator: ">=",
		},
		{
			title: "trigger_type: limit",
			trigger: &Limit{
				Operator: "<=",
				Price:    decimal.NewFromFloat(30144.542314410480331391),
			},
			expectedOperator: "<=",
		},
	}

	for _, tc := range testcases {
		o := tc.trigger.GetOperator()
		if o != tc.expectedOperator {
			t.Errorf("TestLimitGetOperator case '%s' - expect '%s', but got '%s'", tc.title, tc.expectedOperator, o)
		}
	}
}

func TestLimitSetOperator(t *testing.T) {
	trigger := &Limit{
		Operator: "<=",
		Price:    decimal.NewFromFloat(38443.27),
	}
	expectedOperator := ">="
	trigger.SetOperator(expectedOperator)
	if expectedOperator != trigger.GetOperator() {
		t.Errorf("TestLimitSetOperator - expect '%s', but got '%s'", expectedOperator, trigger.GetOperator())
	}
}

func TestLimitReadjustPrice(t *testing.T) {
	testcases := []struct {
		title           string
		price           decimal.Decimal
		t               time.Time
		trigger         Limit
		expectedTrigger Limit
	}{
		{
			title: "readjust price",
			price: decimal.NewFromFloat(32213),
			trigger: Limit{
				Operator: "<=",
				Price:    decimal.NewFromFloat(30144.542314410480331391),
			},
			expectedTrigger: Limit{
				Operator: "<=",
				Price:    decimal.NewFromFloat(32213),
			},
		},
	}

	for _, tc := range testcases {
		tc.trigger.ReadjustPrice(tc.price, time.Now()) // time doesn't matter for 'limit'
		if !reflect.DeepEqual(tc.expectedTrigger, tc.trigger) {
			t.Errorf("TestLimitReadjustPrice case '%s' - trigger and expectedTrigger aren't equal", tc.title)
		}
	}
}

func TestLimitUpdatePriceByPercent(t *testing.T) {
	testcases := []struct {
		title           string
		percent         decimal.Decimal
		trigger         Limit
		expectedTrigger Limit
	}{
		{
			title:   "update price by percent",
			percent: decimal.NewFromFloat(1.0001),
			trigger: Limit{
				Operator: "<=",
				Price:    decimal.NewFromFloat(1100),
			},
			expectedTrigger: Limit{
				Operator: "<=",
				Price:    decimal.NewFromFloat(1100.11),
			},
		},
	}

	for _, tc := range testcases {
		tc.trigger.UpdatePriceByPercent(tc.percent)
		if !reflect.DeepEqual(tc.expectedTrigger, tc.trigger) {
			t.Errorf("TestLimitUpdatePriceByPercent case '%s' - trigger and expectedTrigger aren't equal", tc.title)
		}
	}
}

func TestLimitClone(t *testing.T) {
	source := &Limit{
		Operator: "<=",
		Price:    decimal.NewFromInt(150),
	}

	// Clone trigger from source
	clone := source.Clone()
	clone.ReadjustPrice(decimal.NewFromInt(100), time.Now())

	if reflect.DeepEqual(clone, source) {
		t.Error("TestLimitClone - trigger and expectedTrigger are equal")
	}
}
