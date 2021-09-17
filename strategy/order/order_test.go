package order

import (
	"crypto-trading-bot-main/strategy/trigger"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

type isTriggeredTeseCase struct {
	title             string
	price             decimal.Decimal
	time              time.Time
	trigger           trigger.Trigger
	expectedTriggered bool
}

func TestNewOrder(t *testing.T) {
	testcases := []struct {
		title         string
		orderType     string
		data          map[string]interface{}
		expectedError bool
	}{
		{
			title:     "new entry",
			orderType: "entry",
			data: map[string]interface{}{
				"trigger": map[string]interface{}{
					"trigger_type": "limit",
					"operator":     "<=",
					"price":        47200.23,
				},
			},
			expectedError: false,
		},
		{
			title:     "new take-profit",
			orderType: "take_profit",
			data: map[string]interface{}{
				"trigger": map[string]interface{}{
					"trigger_type": "limit",
					"operator":     "<=",
					"price":        47200.23,
				},
			},
			expectedError: false,
		},
		{
			title:     "new stop-loss",
			orderType: "stop_loss",
			data: map[string]interface{}{
				"trigger": map[string]interface{}{
					"trigger_type": "limit",
					"operator":     "<=",
					"price":        47200.23,
				},
			},
			expectedError: false,
		},
		{
			title:     "new nonexistent type",
			orderType: "nonexistent",
			data: map[string]interface{}{
				"trigger": map[string]interface{}{
					"trigger_type": "limit",
					"operator":     "<=",
					"price":        47200.23,
				},
			},
			expectedError: true,
		},
	}

	for _, tc := range testcases {
		_, err := NewOrder(LONG, "limit", tc.orderType, tc.data)
		hasError := (err != nil)
		if tc.expectedError != hasError {
			t.Errorf("TestNewOrder case '%s' - expect '%t', but got '%t'", tc.title, tc.expectedError, hasError)
		}
	}
}

func getIsTriggeredTestCases() []isTriggeredTeseCase {
	return []isTriggeredTeseCase{
		{
			title: "test limit is triggered",
			price: decimal.NewFromFloat(47200.22),
			time:  time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
			trigger: &trigger.Limit{
				Operator: "<=",
				Price:    decimal.NewFromFloat(47200.23),
			},
			expectedTriggered: true,
		},
		{
			title: "test limit isn't triggered",
			price: decimal.NewFromFloat(47200.22),
			time:  time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
			trigger: &trigger.Limit{
				Operator: ">=",
				Price:    decimal.NewFromFloat(47200.23),
			},
			expectedTriggered: false,
		},
		{
			title: "test line is triggered",
			price: decimal.NewFromFloat(35989.56),
			time:  time.Date(2021, 7, 27, 19, 30, 0, 0, time.UTC), // 35989.55965065502185401
			trigger: &trigger.Line{
				Operator: ">=",
				Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(33874.98),
				Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(38443.27),
			},
			expectedTriggered: true,
		},
		{
			title: "test line isn't triggered",
			price: decimal.NewFromFloat(35989.56),
			time:  time.Date(2021, 7, 27, 19, 30, 0, 0, time.UTC), // 35989.55965065502185401
			trigger: &trigger.Line{
				Operator: "<=",
				Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(33874.98),
				Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(38443.27),
			},
			expectedTriggered: false,
		},
	}
}
