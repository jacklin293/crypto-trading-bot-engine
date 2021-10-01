package order

import (
	"crypto-trading-bot-engine/strategy/trigger"
	"reflect"
	"testing"

	"github.com/shopspring/decimal"
)

func TestNewTakeProfit(t *testing.T) {
	testcases := []struct {
		title         string
		data          map[string]interface{}
		expectedError bool
	}{
		{
			title: "new limit trigger",
			data: map[string]interface{}{
				"trigger": map[string]interface{}{
					"trigger_type": "limit",
					"operator":     "<=",
					"price":        "47200.23",
				},
			},
			expectedError: false,
		},
		{
			title: "new line trigger",
			data: map[string]interface{}{
				"trigger": map[string]interface{}{
					"trigger_type": "line",
					"operator":     ">=",
					"time_1":       "2021-08-18T18:00:00Z",
					"price_1":      "46000.23",
					"time_2":       "2021-08-19T01:45:00Z",
					"price_2":      "45234.56",
				},
			},
			expectedError: false,
		},
		{
			title:         "'trigger' is missing",
			data:          map[string]interface{}{},
			expectedError: true,
		},
	}

	for _, tc := range testcases {
		_, err := NewTakeProfit(tc.data)
		hasError := (err != nil)
		if tc.expectedError != hasError {
			t.Errorf("TestNewTakeProfit case '%s' - expect '%t', but got '%t'", tc.title, tc.expectedError, hasError)
		}
	}
}

func TestTakeProfitGetSetTrigger(t *testing.T) {
	expectedTrigger := &trigger.Limit{
		Operator: "<=",
		Price:    decimal.NewFromFloat(47200.23),
	}
	o := &TakeProfit{}
	o.SetTrigger(expectedTrigger)
	trigger := o.GetTrigger()

	if !reflect.DeepEqual(expectedTrigger, trigger) {
		t.Errorf("TestTakeProfitGetSetTrigger - expect '%v', but got '%v'", expectedTrigger, trigger)
	}
}

func TestTakeProfitIsTriggered(t *testing.T) {
	testcases := getIsTriggeredTestCases()
	for _, tc := range testcases {
		o := TakeProfit{
			Trigger: tc.trigger,
		}
		triggered := o.IsTriggered(tc.time, tc.price)

		if tc.expectedTriggered != triggered {
			t.Errorf("TestTakeProfitIsTriggered case '%s' - expect '%t', but got '%t'", tc.title, tc.expectedTriggered, triggered)
		}
	}
}
