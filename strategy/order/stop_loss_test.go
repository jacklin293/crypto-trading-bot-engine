package order

import (
	"crypto-trading-bot-engine/strategy/trigger"
	"reflect"
	"testing"

	"github.com/shopspring/decimal"
)

func TestNewStopLoss(t *testing.T) {
	testcases := []struct {
		title         string
		entryType     string
		data          map[string]interface{}
		expectedError bool
	}{
		{
			title:     "new limit trigger",
			entryType: ENTRY_LIMIT,
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
			title:         "new limit trigger - 'trigger' is missing",
			entryType:     ENTRY_LIMIT,
			data:          map[string]interface{}{},
			expectedError: true,
		},
		{
			title:     "new baseline trigger",
			entryType: ENTRY_BASELINE,
			data: map[string]interface{}{
				"loss_tolerance_percent":        0.005,
				"baseline_readjustment_enabled": true,
			},
			expectedError: false,
		},
		{
			title:     "new baseline trigger - 'loss_tolerance_percent' is missing",
			entryType: ENTRY_BASELINE,
			data: map[string]interface{}{
				"baseline_readjustment_enabled": true,
			},
			expectedError: true,
		},
		{
			title:     "new baseline trigger - 'loss_tolerance_percent' less than 0",
			entryType: ENTRY_BASELINE,
			data: map[string]interface{}{
				"loss_tolerance_percent":        -0.005,
				"baseline_readjustment_enabled": true,
			},
			expectedError: true,
		},
	}

	for _, tc := range testcases {
		_, err := NewStopLoss(tc.entryType, tc.data)
		hasError := (err != nil)
		if tc.expectedError != hasError {
			t.Errorf("TestNewStopLoss case '%s' - expect '%t', but got '%t'", tc.title, tc.expectedError, hasError)
		}
	}
}

func TestStopLossGetSetUnsetTrigger(t *testing.T) {
	expectedTrigger := &trigger.Limit{
		Operator: "<=",
		Price:    decimal.NewFromFloat(47200.23),
	}
	o := &StopLoss{}
	o.SetTrigger(expectedTrigger)
	trigger := o.GetTrigger()

	if !reflect.DeepEqual(expectedTrigger, trigger) {
		t.Errorf("TestStopLossGetSetUnsetTrigger - expect '%v', but got '%v'", expectedTrigger, trigger)
	}

	o.UnsetTrigger()
	if o.Trigger != nil {
		t.Errorf("TestStopLossGetSetUnsetTrigger - expect 'nil', but got '%v'", o.GetTrigger())
	}
}

func TestStopLossIsTriggered(t *testing.T) {
	testcases := getIsTriggeredTestCases()
	for _, tc := range testcases {
		o := StopLoss{
			Trigger: tc.trigger,
		}
		triggered := o.IsTriggered(tc.time, tc.price)

		if tc.expectedTriggered != triggered {
			t.Errorf("TestStopLossIsTriggered case '%s' - expect '%t', but got '%t'", tc.title, tc.expectedTriggered, triggered)
		}
	}
}

func TestStopLossUpdateTriggerByLossPercent(t *testing.T) {
	testcases := []struct {
		title           string
		LossPercent     float64
		side            Side
		baselinePrice   decimal.Decimal
		expectedTrigger trigger.Trigger
	}{
		{
			title:         "test long - positive percent",
			LossPercent:   0.01,
			side:          LONG,
			baselinePrice: decimal.NewFromFloat(100.1),
			expectedTrigger: &trigger.Limit{
				TriggerType: "limit",
				Operator:    "<=",
				Price:       decimal.NewFromFloat(99.099),
			},
		},
		{
			title:         "test short - positive percent",
			LossPercent:   0.01,
			side:          SHORT,
			baselinePrice: decimal.NewFromFloat(100.1),
			expectedTrigger: &trigger.Limit{
				TriggerType: "limit",
				Operator:    ">=",
				Price:       decimal.NewFromFloat(101.101),
			},
		},
	}

	for _, tc := range testcases {
		o := &StopLoss{LossTolerancePercent: tc.LossPercent}
		o.UpdateTriggerByLossPercent(tc.side, tc.baselinePrice)

		if !reflect.DeepEqual(tc.expectedTrigger, o.GetTrigger()) {
			t.Errorf("TestStopLossUpdateTriggerByLossPercent case '%s' - expect '%v', but got '%v'", tc.title, tc.expectedTrigger, o.GetTrigger())
		}
	}
}
