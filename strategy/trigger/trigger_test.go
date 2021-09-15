package trigger

import (
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestValidateOperator(t *testing.T) {
	testcases := []struct {
		title         string
		operator      string
		expectedError bool
	}{
		{
			title:         "validate operator '>='",
			operator:      ">=",
			expectedError: false,
		},
		{
			title:         "validate operator '<='",
			operator:      "<=",
			expectedError: false,
		},
		{
			title:         "validate operator '>'",
			operator:      ">",
			expectedError: true,
		},
		{
			title:         "validate operator '='",
			operator:      "=",
			expectedError: true,
		},
		{
			title:         "validate operator '<'",
			operator:      "<",
			expectedError: true,
		},
	}

	for _, tc := range testcases {
		err := validateOperator(tc.operator)
		hasError := (err != nil)
		if tc.expectedError != hasError {
			t.Errorf("TestValidateOperator case '%s' - expect '%t', but got '%t'", tc.title, tc.expectedError, hasError)
		}
	}
}

func TestNewTrigger(t *testing.T) {
	testcases := []struct {
		title         string
		params        map[string]interface{}
		expectedError bool
	}{
		{
			title: "new limit trigger",
			params: map[string]interface{}{
				"trigger_type": "limit",
				"operator":     ">=",
				"price":        float64(30144.542314410480331391),
			},
			expectedError: false,
		},
		{
			title: "new line trigger",
			params: map[string]interface{}{
				"trigger_type": "line",
				"operator":     ">=",
				"time_1":       "2021-07-25 14:30:00",
				"price_1":      float64(33874.98),
				"time_2":       "2021-07-30 09:00:00",
				"price_2":      float64(38443.27),
			},
			expectedError: false,
		},
	}

	for _, tc := range testcases {
		_, err := NewTrigger(tc.params)
		hasError := (err != nil)
		if tc.expectedError != hasError {
			t.Errorf("TestNewTrigger case '%s' - expect no error, but got an error '%v'", tc.title, err)
		}
	}
}

func TestNewTriggers(t *testing.T) {
	testcases := []struct {
		title                string
		params               []interface{}
		expectedError        bool
		expectedTriggerCount int
	}{
		{
			title: "new 1 trigger",
			params: []interface{}{
				map[string]interface{}{
					"trigger_type": "limit",
					"operator":     ">=",
					"price":        float64(30144.542314410480331391),
				},
			},
			expectedTriggerCount: 1,
		},
		{
			title: "new 2 triggers",
			params: []interface{}{
				map[string]interface{}{
					"trigger_type": "limit",
					"operator":     ">=",
					"price":        float64(30144.542314410480331391),
				},
				map[string]interface{}{
					"trigger_type": "line",
					"operator":     ">=",
					"time_1":       "2021-07-25 14:30:00",
					"price_1":      float64(33874.98),
					"time_2":       "2021-07-30 09:00:00",
					"price_2":      float64(38443.27),
				},
			},
			expectedTriggerCount: 2,
		},
		{
			title: "new 3 triggers",
			params: []interface{}{
				map[string]interface{}{
					"trigger_type": "limit",
					"operator":     ">=",
					"price":        float64(30144.542314410480331391),
				},
				map[string]interface{}{
					"trigger_type": "limit",
					"operator":     "<=",
					"price":        float64(60000),
				},
				map[string]interface{}{
					"trigger_type": "line",
					"operator":     ">=",
					"time_1":       "2021-07-25 14:30:00",
					"price_1":      float64(33874.98),
					"time_2":       "2021-07-30 09:00:00",
					"price_2":      float64(38443.27),
				},
			},
			expectedTriggerCount: 3,
		},
	}

	for _, tc := range testcases {
		triggers, err := NewTriggers(tc.params)
		hasError := (err != nil)
		if tc.expectedError != hasError {
			t.Errorf("TestNewTriggers case '%s' - expect '%t', but got '%t'", tc.title, tc.expectedError, hasError)
		}
		if tc.expectedTriggerCount != len(triggers) {
			t.Errorf("TestNewTriggers case '%s' - expect '%d', but got '%d'", tc.title, tc.expectedTriggerCount, len(triggers))
		}
	}
}

func TestIsTriggeredBySingleTrigger(t *testing.T) {
	testcases := []struct {
		title       string
		trigger     Trigger
		time        time.Time
		marketPrice decimal.Decimal
		isTriggered bool
	}{
		{
			title: "trigger_type: limit (>=)",
			trigger: &Limit{
				Operator: ">=",
				Price:    decimal.NewFromInt(10),
			},
			time:        time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC), // time doesn't matter
			marketPrice: decimal.NewFromInt(11),
			isTriggered: true,
		},
		{
			title: "trigger_type: limit (>=)",
			trigger: &Limit{
				Operator: ">=",
				Price:    decimal.NewFromInt(10),
			},
			time:        time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC), // time doesn't matter
			marketPrice: decimal.NewFromInt(10),
			isTriggered: true,
		},
		{
			title: "trigger_type: limit (>=)",
			trigger: &Limit{
				Operator: ">=",
				Price:    decimal.NewFromFloat(10.00000001),
			},
			time:        time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC), // time doesn't matter
			marketPrice: decimal.NewFromFloat(10.00000001),
			isTriggered: true,
		},
		{
			title: "trigger_type: limit (>=)",
			trigger: &Limit{
				Operator: ">=",
				Price:    decimal.NewFromInt(10),
			},
			time:        time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC), // time doesn't matter
			marketPrice: decimal.NewFromInt(9),
			isTriggered: false,
		},
		{
			title: "trigger_type: line (>=, uptrend), before time_1",
			trigger: &Line{
				Operator: ">=",
				Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(33874.98),
				Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(38443.27),
			},
			time:        time.Date(2021, 7, 21, 17, 00, 0, 0, time.UTC),
			marketPrice: decimal.NewFromInt(30145),
			isTriggered: true,
		},
		{
			title: "trigger_type: line (>=, uptrend), before time_1",
			trigger: &Line{
				Operator: ">=",
				Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(33874.98),
				Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(38443.27),
			},
			time:        time.Date(2021, 7, 21, 17, 00, 0, 0, time.UTC),
			marketPrice: decimal.NewFromInt(30144),
			isTriggered: false,
		},
		{
			title: "trigger_type: line (>=, uptrend), equal to time_1",
			trigger: &Line{
				Operator: ">=",
				Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(33874.98),
				Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(38443.27),
			},
			time:        time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
			marketPrice: decimal.NewFromFloat(33874.98),
			isTriggered: true,
		},
		{
			title: "trigger_type: line (>=, uptrend), during time period",
			trigger: &Line{
				Operator: ">=",
				Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(33874.98),
				Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(38443.27),
			},
			time:        time.Date(2021, 7, 27, 19, 30, 0, 0, time.UTC),
			marketPrice: decimal.NewFromInt(35990),
			isTriggered: true,
		},
		{
			title: "trigger_type: line (>=, uptrend), during time period",
			trigger: &Line{
				Operator: ">=",
				Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(33874.98),
				Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(38443.27),
			},
			time:        time.Date(2021, 7, 27, 19, 30, 0, 0, time.UTC),
			marketPrice: decimal.NewFromInt(35989),
			isTriggered: false,
		},
		{
			title: "trigger_type: line (>=, uptrend), equal to time_2",
			trigger: &Line{
				Operator: ">=",
				Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(33874.98),
				Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(38443.27),
			},
			time:        time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
			marketPrice: decimal.NewFromFloat(38443.27),
			isTriggered: true,
		},
		{
			title: "trigger_type: line (>=, uptrend), after time_2",
			trigger: &Line{
				Operator: ">=",
				Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(33874.98),
				Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(38443.27),
			},
			time:        time.Date(2021, 8, 1, 20, 30, 0, 0, time.UTC),
			marketPrice: decimal.NewFromInt(40818),
			isTriggered: true,
		},
		{
			title: "trigger_type: line (>=, uptrend), after time_2",
			trigger: &Line{
				Operator: ">=",
				Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(33874.98),
				Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(38443.27),
			},
			time:        time.Date(2021, 8, 1, 20, 30, 0, 0, time.UTC),
			marketPrice: decimal.NewFromInt(40817),
			isTriggered: false,
		},
		{
			title: "trigger_type: line (>=, downtrend), before time_1",
			trigger: &Line{
				Operator: ">=",
				Time1:    time.Date(2021, 6, 30, 19, 0, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(34835.08),
				Time2:    time.Date(2021, 7, 10, 20, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(33004.78),
			},
			time:        time.Date(2021, 6, 21, 1, 0, 0, 0, time.UTC),
			marketPrice: decimal.NewFromInt(36612),
			isTriggered: false,
		},
		{
			title: "trigger_type: line (>=, downtrend), before time_1",
			trigger: &Line{
				Operator: ">=",
				Time1:    time.Date(2021, 6, 30, 19, 0, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(34835.08),
				Time2:    time.Date(2021, 7, 10, 20, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(33004.78),
			},
			time:        time.Date(2021, 6, 21, 1, 0, 0, 0, time.UTC),
			marketPrice: decimal.NewFromInt(36613),
			isTriggered: true,
		},
		{
			title: "trigger_type: line (>=, downtrend), equal to time_1",
			trigger: &Line{
				Operator: ">=",
				Time1:    time.Date(2021, 6, 30, 19, 0, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(34835.08),
				Time2:    time.Date(2021, 7, 10, 20, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(33004.78),
			},
			time:        time.Date(2021, 6, 30, 19, 0, 0, 0, time.UTC),
			marketPrice: decimal.NewFromFloat(34835.08),
			isTriggered: true,
		},
		{
			title: "trigger_type: limit (<=)",
			trigger: &Limit{
				Operator: "<=",
				Price:    decimal.NewFromInt(10),
			},
			time:        time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC), // time doesn't matter
			marketPrice: decimal.NewFromInt(11),
			isTriggered: false,
		},
		{
			title: "trigger_type: limit (<=)",
			trigger: &Limit{
				Operator: "<=",
				Price:    decimal.NewFromInt(10),
			},
			time:        time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC), // time doesn't matter
			marketPrice: decimal.NewFromInt(10),
			isTriggered: true,
		},
		{
			title: "trigger_type: limit (<=)",
			trigger: &Limit{
				Operator: "<=",
				Price:    decimal.NewFromFloat(10.00000001),
			},
			time:        time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC), // time doesn't matter
			marketPrice: decimal.NewFromFloat(10.00000001),
			isTriggered: true,
		},
		{
			title: "trigger_type: limit (<=)",
			trigger: &Limit{
				Operator: "<=",
				Price:    decimal.NewFromInt(10),
			},
			time:        time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC), // time doesn't matter
			marketPrice: decimal.NewFromInt(9),
			isTriggered: true,
		},
		{
			title: "trigger_type: line (<=, uptrend), before time_1",
			trigger: &Line{
				Operator: "<=",
				Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(33874.98),
				Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(38443.27),
			},
			time:        time.Date(2021, 7, 21, 17, 00, 0, 0, time.UTC),
			marketPrice: decimal.NewFromInt(30145),
			isTriggered: false,
		},
		{
			title: "trigger_type: line (<=, uptrend), before time_1",
			trigger: &Line{
				Operator: "<=",
				Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(33874.98),
				Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(38443.27),
			},
			time:        time.Date(2021, 7, 21, 17, 00, 0, 0, time.UTC),
			marketPrice: decimal.NewFromInt(30144),
			isTriggered: true,
		},
		{
			title: "trigger_type: line (<=, uptrend), equal to time_1",
			trigger: &Line{
				Operator: "<=",
				Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(33874.98),
				Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(38443.27),
			},
			time:        time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
			marketPrice: decimal.NewFromFloat(33874.98),
			isTriggered: true,
		},
		{
			title: "trigger_type: line (<=, uptrend), during time period",
			trigger: &Line{
				Operator: "<=",
				Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(33874.98),
				Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(38443.27),
			},
			time:        time.Date(2021, 7, 27, 19, 30, 0, 0, time.UTC),
			marketPrice: decimal.NewFromInt(35990),
			isTriggered: false,
		},
		{
			title: "trigger_type: line (<=, uptrend), during time period",
			trigger: &Line{
				Operator: "<=",
				Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(33874.98),
				Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(38443.27),
			},
			time:        time.Date(2021, 7, 27, 19, 30, 0, 0, time.UTC),
			marketPrice: decimal.NewFromInt(35989),
			isTriggered: true,
		},
		{
			title: "trigger_type: line (<=, uptrend), equal to time_2",
			trigger: &Line{
				Operator: "<=",
				Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(33874.98),
				Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(38443.27),
			},
			time:        time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
			marketPrice: decimal.NewFromFloat(38443.27),
			isTriggered: true,
		},
		{
			title: "trigger_type: line (<=, uptrend), after time_2",
			trigger: &Line{
				Operator: "<=",
				Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(33874.98),
				Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(38443.27),
			},
			time:        time.Date(2021, 8, 1, 20, 30, 0, 0, time.UTC),
			marketPrice: decimal.NewFromInt(40818),
			isTriggered: false,
		},
		{
			title: "trigger_type: line (<=, uptrend), after time_2",
			trigger: &Line{
				Operator: "<=",
				Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(33874.98),
				Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(38443.27),
			},
			time:        time.Date(2021, 8, 1, 20, 30, 0, 0, time.UTC),
			marketPrice: decimal.NewFromInt(40817),
			isTriggered: true,
		},
		{
			title: "trigger_type: line (<=, downtrend), before time_1",
			trigger: &Line{
				Operator: "<=",
				Time1:    time.Date(2021, 6, 30, 19, 0, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(34835.08),
				Time2:    time.Date(2021, 7, 10, 20, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(33004.78),
			},
			time:        time.Date(2021, 6, 21, 1, 0, 0, 0, time.UTC),
			marketPrice: decimal.NewFromInt(36612),
			isTriggered: true,
		},
		{
			title: "trigger_type: line (<=, downtrend), before time_1",
			trigger: &Line{
				Operator: "<=",
				Time1:    time.Date(2021, 6, 30, 19, 0, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(34835.08),
				Time2:    time.Date(2021, 7, 10, 20, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(33004.78),
			},
			time:        time.Date(2021, 6, 21, 1, 0, 0, 0, time.UTC),
			marketPrice: decimal.NewFromInt(36613),
			isTriggered: false,
		},
		{
			title: "trigger_type: line (<=, downtrend), equal to time_1",
			trigger: &Line{
				Operator: "<=",
				Time1:    time.Date(2021, 6, 30, 19, 0, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(34835.08),
				Time2:    time.Date(2021, 7, 10, 20, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(33004.78),
			},
			time:        time.Date(2021, 6, 30, 19, 0, 0, 0, time.UTC),
			marketPrice: decimal.NewFromFloat(34835.08),
			isTriggered: true,
		},
	}
	for _, tc := range testcases {
		result := IsTriggeredBySingleTrigger(tc.trigger, tc.time, tc.marketPrice)
		if result != tc.isTriggered {
			t.Errorf("TestIsTriggeredBySingleTrigger case '%s' - expect '%t', but got '%t'", tc.title, result, tc.isTriggered)
		}
	}
}

func TestIsTriggeredByMultipleTriggers(t *testing.T) {
	// mixed triggers - AND
	testcases := []struct {
		title       string
		operator    string
		triggers    []Trigger
		time        time.Time
		marketPrice decimal.Decimal
		isTriggered bool
	}{
		{
			title:    "mixed triggers ('AND', uptrend line), both >= market price",
			operator: "AND",
			triggers: []Trigger{
				&Limit{
					Operator: ">=",
					Price:    decimal.NewFromInt(36000),
				},
				&Line{
					Operator: ">=",
					Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
					Price1:   decimal.NewFromFloat(33874.98),
					Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
					Price2:   decimal.NewFromFloat(38443.27),
				},
			},
			time:        time.Date(2021, 7, 27, 19, 30, 0, 0, time.UTC), // 35989
			marketPrice: decimal.NewFromInt(36001),
			isTriggered: true,
		},
		{
			title:    "mixed triggers ('OR', uptrend line), both >= market price",
			operator: "OR",
			triggers: []Trigger{
				&Limit{
					Operator: ">=",
					Price:    decimal.NewFromInt(36000),
				},
				&Line{
					Operator: ">=",
					Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
					Price1:   decimal.NewFromFloat(33874.98),
					Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
					Price2:   decimal.NewFromFloat(38443.27),
				},
			},
			time:        time.Date(2021, 7, 27, 19, 30, 0, 0, time.UTC), // 35989
			marketPrice: decimal.NewFromInt(36001),
			isTriggered: true,
		},
		{
			title:    "mixed triggers ('AND', uptrend line), non of them >= market price",
			operator: "AND",
			triggers: []Trigger{
				&Limit{
					Operator: ">=",
					Price:    decimal.NewFromInt(36000),
				},
				&Line{
					Operator: ">=",
					Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
					Price1:   decimal.NewFromFloat(33874.98),
					Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
					Price2:   decimal.NewFromFloat(38443.27),
				},
			},
			time:        time.Date(2021, 7, 27, 19, 30, 0, 0, time.UTC), // 35989
			marketPrice: decimal.NewFromInt(35988),
			isTriggered: false,
		},
		{
			title:    "mixed triggers ('AND', uptrend line), both <= market price",
			operator: "AND",
			triggers: []Trigger{
				&Limit{
					Operator: "<=",
					Price:    decimal.NewFromInt(36000),
				},
				&Line{
					Operator: "<=",
					Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
					Price1:   decimal.NewFromFloat(33874.98),
					Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
					Price2:   decimal.NewFromFloat(38443.27),
				},
			},
			time:        time.Date(2021, 7, 27, 19, 30, 0, 0, time.UTC), // 35989
			marketPrice: decimal.NewFromInt(35988),
			isTriggered: true,
		},
		{
			title:    "mixed triggers ('AND', uptrend line), only limit trigger >= market price",
			operator: "AND",
			triggers: []Trigger{
				&Limit{
					Operator: ">=",
					Price:    decimal.NewFromInt(36000),
				},
				&Line{
					Operator: ">=",
					Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
					Price1:   decimal.NewFromFloat(33874.98),
					Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
					Price2:   decimal.NewFromFloat(38443.27),
				},
			},
			time:        time.Date(2021, 7, 27, 19, 30, 0, 0, time.UTC), // 35989
			marketPrice: decimal.NewFromInt(35990),
			isTriggered: false,
		},
		{
			title:    "mixed triggers ('OR', uptrend line), only limit trigger >= market price",
			operator: "OR",
			triggers: []Trigger{
				&Limit{
					Operator: ">=",
					Price:    decimal.NewFromInt(36000),
				},
				&Line{
					Operator: ">=",
					Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
					Price1:   decimal.NewFromFloat(33874.98),
					Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
					Price2:   decimal.NewFromFloat(38443.27),
				},
			},
			time:        time.Date(2021, 7, 27, 19, 30, 0, 0, time.UTC), // 35989
			marketPrice: decimal.NewFromInt(35990),
			isTriggered: true,
		},
		{
			title:    "mixed triggers ('AND', uptrend line), only line trigger >= market price",
			operator: "AND",
			triggers: []Trigger{
				&Limit{
					Operator: ">=",
					Price:    decimal.NewFromInt(35987),
				},
				&Line{
					Operator: ">=",
					Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
					Price1:   decimal.NewFromFloat(33874.98),
					Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
					Price2:   decimal.NewFromFloat(38443.27),
				},
			},
			time:        time.Date(2021, 7, 27, 19, 30, 0, 0, time.UTC), // 35989
			marketPrice: decimal.NewFromInt(35988),
			isTriggered: false,
		},
		{
			title:    "mixed triggers ('OR', uptrend line), only line trigger >= market price",
			operator: "OR",
			triggers: []Trigger{
				&Limit{
					Operator: ">=",
					Price:    decimal.NewFromInt(35987),
				},
				&Line{
					Operator: ">=",
					Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
					Price1:   decimal.NewFromFloat(33874.98),
					Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
					Price2:   decimal.NewFromFloat(38443.27),
				},
			},
			time:        time.Date(2021, 7, 27, 19, 30, 0, 0, time.UTC), // 35989
			marketPrice: decimal.NewFromInt(35988),
			isTriggered: true,
		},
		{
			title:    "mixed triggers ('AND', uptrend line), market price is in between 2 triggers",
			operator: "AND",
			triggers: []Trigger{
				&Limit{
					Operator: "<=",
					Price:    decimal.NewFromInt(36000),
				},
				&Line{
					Operator: ">=",
					Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
					Price1:   decimal.NewFromFloat(33874.98),
					Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
					Price2:   decimal.NewFromFloat(38443.27),
				},
			},
			time:        time.Date(2021, 7, 27, 19, 30, 0, 0, time.UTC), // 35989
			marketPrice: decimal.NewFromInt(35995),
			isTriggered: true,
		},
		{
			title:    "mixed triggers ('OR', uptrend line), market price is in between 2 triggers",
			operator: "OR",
			triggers: []Trigger{
				&Limit{
					Operator: "<=",
					Price:    decimal.NewFromInt(36000),
				},
				&Line{
					Operator: ">=",
					Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
					Price1:   decimal.NewFromFloat(33874.98),
					Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
					Price2:   decimal.NewFromFloat(38443.27),
				},
			},
			time:        time.Date(2021, 7, 27, 19, 30, 0, 0, time.UTC), // 35989
			marketPrice: decimal.NewFromInt(35995),
			isTriggered: true,
		},
		{
			title:    "mixed triggers ('AND', uptrend line), market price is in between 2 different triggers",
			operator: "AND",
			triggers: []Trigger{
				&Limit{
					Operator: ">=",
					Price:    decimal.NewFromInt(35987),
				},
				&Line{
					Operator: "<=",
					Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
					Price1:   decimal.NewFromFloat(33874.98),
					Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
					Price2:   decimal.NewFromFloat(38443.27),
				},
			},
			time:        time.Date(2021, 7, 27, 19, 30, 0, 0, time.UTC), // 35989
			marketPrice: decimal.NewFromInt(35988),
			isTriggered: true,
		},
		{
			title:    "mixed triggers ('OR', uptrend line), market price is in between 2 different triggers",
			operator: "OR",
			triggers: []Trigger{
				&Limit{
					Operator: ">=",
					Price:    decimal.NewFromInt(35987),
				},
				&Line{
					Operator: "<=",
					Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
					Price1:   decimal.NewFromFloat(33874.98),
					Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
					Price2:   decimal.NewFromFloat(38443.27),
				},
			},
			time:        time.Date(2021, 7, 27, 19, 30, 0, 0, time.UTC), // 35989
			marketPrice: decimal.NewFromInt(35988),
			isTriggered: true,
		},
		{
			title:    "mixed triggers ('AND', uptrend line), market price is in between 2 lines",
			operator: "AND",
			triggers: []Trigger{
				&Line{
					Operator: "<=",
					Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
					Price1:   decimal.NewFromFloat(33879.98), // +5
					Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
					Price2:   decimal.NewFromFloat(38448.27), // +5
				},
				&Line{
					Operator: ">=",
					Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
					Price1:   decimal.NewFromFloat(33874.98),
					Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
					Price2:   decimal.NewFromFloat(38443.27),
				},
			},
			time:        time.Date(2021, 7, 27, 19, 30, 0, 0, time.UTC), // 35989
			marketPrice: decimal.NewFromInt(35992),                      // 35590, 35591, 35592, 35593, 35594 can pas
			isTriggered: true,
		},
		{
			title:    "mixed triggers ('OR', uptrend line), market price is in between 2 lines",
			operator: "OR",
			triggers: []Trigger{
				&Line{
					Operator: "<=",
					Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
					Price1:   decimal.NewFromFloat(33879.98), // +5
					Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
					Price2:   decimal.NewFromFloat(38448.27), // +5
				},
				&Line{
					Operator: ">=",
					Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
					Price1:   decimal.NewFromFloat(33874.98),
					Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
					Price2:   decimal.NewFromFloat(38443.27),
				},
			},
			time:        time.Date(2021, 7, 27, 19, 30, 0, 0, time.UTC), // 35989
			marketPrice: decimal.NewFromInt(35992),                      // 35590, 35591, 35592, 35593, 35594 can pas
			isTriggered: true,
		},
		{
			title:    "mixed triggers ('AND', downtrend line), both >= market price",
			operator: "AND",
			triggers: []Trigger{
				&Limit{
					Operator: ">=",
					Price:    decimal.NewFromInt(36613),
				},
				&Line{
					Operator: ">=",
					Time1:    time.Date(2021, 6, 30, 19, 0, 0, 0, time.UTC),
					Price1:   decimal.NewFromFloat(34835.08),
					Time2:    time.Date(2021, 7, 10, 20, 0, 0, 0, time.UTC),
					Price2:   decimal.NewFromFloat(33004.78),
				},
			},
			time:        time.Date(2021, 6, 21, 1, 0, 0, 0, time.UTC), // 36612.21
			marketPrice: decimal.NewFromInt(36614),
			isTriggered: true,
		},
		{
			title:    "mixed triggers ('OR', downtrend line), both >= market price",
			operator: "OR",
			triggers: []Trigger{
				&Limit{
					Operator: ">=",
					Price:    decimal.NewFromInt(36613),
				},
				&Line{
					Operator: ">=",
					Time1:    time.Date(2021, 6, 30, 19, 0, 0, 0, time.UTC),
					Price1:   decimal.NewFromFloat(34835.08),
					Time2:    time.Date(2021, 7, 10, 20, 0, 0, 0, time.UTC),
					Price2:   decimal.NewFromFloat(33004.78),
				},
			},
			time:        time.Date(2021, 6, 21, 1, 0, 0, 0, time.UTC), // 36612.21
			marketPrice: decimal.NewFromInt(36614),
			isTriggered: true,
		},
		{
			title:    "mixed triggers ('AND', downtrend line), non of them <= market price",
			operator: "AND",
			triggers: []Trigger{
				&Limit{
					Operator: ">=",
					Price:    decimal.NewFromInt(36613),
				},
				&Line{
					Operator: ">=",
					Time1:    time.Date(2021, 6, 30, 19, 0, 0, 0, time.UTC),
					Price1:   decimal.NewFromFloat(34835.08),
					Time2:    time.Date(2021, 7, 10, 20, 0, 0, 0, time.UTC),
					Price2:   decimal.NewFromFloat(33004.78),
				},
			},
			time:        time.Date(2021, 6, 21, 1, 0, 0, 0, time.UTC), // 36612.21
			marketPrice: decimal.NewFromInt(36611),
			isTriggered: false,
		},
		{
			title:    "mixed triggers ('AND', downtrend line), only limit trigger >= market price",
			operator: "AND",
			triggers: []Trigger{
				&Limit{
					Operator: ">=",
					Price:    decimal.NewFromInt(36614),
				},
				&Line{
					Operator: ">=",
					Time1:    time.Date(2021, 6, 30, 19, 0, 0, 0, time.UTC),
					Price1:   decimal.NewFromFloat(34835.08),
					Time2:    time.Date(2021, 7, 10, 20, 0, 0, 0, time.UTC),
					Price2:   decimal.NewFromFloat(33004.78),
				},
			},
			time:        time.Date(2021, 6, 21, 1, 0, 0, 0, time.UTC), // 36612.21
			marketPrice: decimal.NewFromInt(36613),
			isTriggered: false,
		},
		{
			title:    "mixed triggers ('OR', downtrend line), only limit trigger >= market price",
			operator: "OR",
			triggers: []Trigger{
				&Limit{
					Operator: ">=",
					Price:    decimal.NewFromInt(36614),
				},
				&Line{
					Operator: ">=",
					Time1:    time.Date(2021, 6, 30, 19, 0, 0, 0, time.UTC),
					Price1:   decimal.NewFromFloat(34835.08),
					Time2:    time.Date(2021, 7, 10, 20, 0, 0, 0, time.UTC),
					Price2:   decimal.NewFromFloat(33004.78),
				},
			},
			time:        time.Date(2021, 6, 21, 1, 0, 0, 0, time.UTC), // 36612.21
			marketPrice: decimal.NewFromInt(36613),
			isTriggered: true,
		},
		{
			title:    "mixed triggers ('AND', downtrend line), only line trigger >= market price",
			operator: "AND",
			triggers: []Trigger{
				&Limit{
					Operator: ">=",
					Price:    decimal.NewFromInt(36610),
				},
				&Line{
					Operator: ">=",
					Time1:    time.Date(2021, 6, 30, 19, 0, 0, 0, time.UTC),
					Price1:   decimal.NewFromFloat(34835.08),
					Time2:    time.Date(2021, 7, 10, 20, 0, 0, 0, time.UTC),
					Price2:   decimal.NewFromFloat(33004.78),
				},
			},
			time:        time.Date(2021, 6, 21, 1, 0, 0, 0, time.UTC), // 36612.21
			marketPrice: decimal.NewFromInt(36611),
			isTriggered: false,
		},
		{
			title:    "mixed triggers ('OR', downtrend line), only line trigger >= market price",
			operator: "OR",
			triggers: []Trigger{
				&Limit{
					Operator: ">=",
					Price:    decimal.NewFromInt(36610),
				},
				&Line{
					Operator: ">=",
					Time1:    time.Date(2021, 6, 30, 19, 0, 0, 0, time.UTC),
					Price1:   decimal.NewFromFloat(34835.08),
					Time2:    time.Date(2021, 7, 10, 20, 0, 0, 0, time.UTC),
					Price2:   decimal.NewFromFloat(33004.78),
				},
			},
			time:        time.Date(2021, 6, 21, 1, 0, 0, 0, time.UTC), // 36612.21
			marketPrice: decimal.NewFromInt(36611),
			isTriggered: true,
		},
		{
			title:    "mixed triggers ('AND', downtrend line), market price is in between 2 triggers",
			operator: "AND",
			triggers: []Trigger{
				&Limit{
					Operator: "<=",
					Price:    decimal.NewFromInt(36614),
				},
				&Line{
					Operator: ">=",
					Time1:    time.Date(2021, 6, 30, 19, 0, 0, 0, time.UTC),
					Price1:   decimal.NewFromFloat(34835.08),
					Time2:    time.Date(2021, 7, 10, 20, 0, 0, 0, time.UTC),
					Price2:   decimal.NewFromFloat(33004.78),
				},
			},
			time:        time.Date(2021, 6, 21, 1, 0, 0, 0, time.UTC), // 36612.21
			marketPrice: decimal.NewFromInt(36613),
			isTriggered: true,
		},
		{
			title:    "mixed triggers ('OR', downtrend line), market price is in between 2 triggers",
			operator: "OR",
			triggers: []Trigger{
				&Limit{
					Operator: "<=",
					Price:    decimal.NewFromInt(36614),
				},
				&Line{
					Operator: ">=",
					Time1:    time.Date(2021, 6, 30, 19, 0, 0, 0, time.UTC),
					Price1:   decimal.NewFromFloat(34835.08),
					Time2:    time.Date(2021, 7, 10, 20, 0, 0, 0, time.UTC),
					Price2:   decimal.NewFromFloat(33004.78),
				},
			},
			time:        time.Date(2021, 6, 21, 1, 0, 0, 0, time.UTC), // 36612.21
			marketPrice: decimal.NewFromInt(36613),
			isTriggered: true,
		},
		{
			title:    "mixed triggers ('AND', downtrend line), market price is in between 2 different triggers",
			operator: "AND",
			triggers: []Trigger{
				&Limit{
					Operator: ">=",
					Price:    decimal.NewFromInt(36610),
				},
				&Line{
					Operator: "<=",
					Time1:    time.Date(2021, 6, 30, 19, 0, 0, 0, time.UTC),
					Price1:   decimal.NewFromFloat(34835.08),
					Time2:    time.Date(2021, 7, 10, 20, 0, 0, 0, time.UTC),
					Price2:   decimal.NewFromFloat(33004.78),
				},
			},
			time:        time.Date(2021, 6, 21, 1, 0, 0, 0, time.UTC), // 36612.21
			marketPrice: decimal.NewFromInt(36611),
			isTriggered: true,
		},
		{
			title:    "mixed triggers ('OR', downtrend line), market price is in between 2 different triggers",
			operator: "OR",
			triggers: []Trigger{
				&Limit{
					Operator: ">=",
					Price:    decimal.NewFromInt(36610),
				},
				&Line{
					Operator: "<=",
					Time1:    time.Date(2021, 6, 30, 19, 0, 0, 0, time.UTC),
					Price1:   decimal.NewFromFloat(34835.08),
					Time2:    time.Date(2021, 7, 10, 20, 0, 0, 0, time.UTC),
					Price2:   decimal.NewFromFloat(33004.78),
				},
			},
			time:        time.Date(2021, 6, 21, 1, 0, 0, 0, time.UTC), // 36612.21
			marketPrice: decimal.NewFromInt(36611),
			isTriggered: true,
		},
		{
			title:    "mixed triggers ('AND', downtrend line), market price is in between 2 lines",
			operator: "AND",
			triggers: []Trigger{
				&Line{
					Operator: "<=",
					Time1:    time.Date(2021, 6, 30, 19, 0, 0, 0, time.UTC),
					Price1:   decimal.NewFromFloat(34840.08), // +5
					Time2:    time.Date(2021, 7, 10, 20, 0, 0, 0, time.UTC),
					Price2:   decimal.NewFromFloat(33009.78), // +5
				},
				&Line{
					Operator: ">=",
					Time1:    time.Date(2021, 6, 30, 19, 0, 0, 0, time.UTC),
					Price1:   decimal.NewFromFloat(34835.08),
					Time2:    time.Date(2021, 7, 10, 20, 0, 0, 0, time.UTC),
					Price2:   decimal.NewFromFloat(33004.78),
				},
			},
			time:        time.Date(2021, 6, 21, 1, 0, 0, 0, time.UTC), // 36612.21
			marketPrice: decimal.NewFromInt(36615),                    // 36613, 36614, 36615, 36616, 36617 can pass
			isTriggered: true,
		},
		{
			title:    "mixed triggers ('OR', downtrend line), market price is in between 2 lines",
			operator: "OR",
			triggers: []Trigger{
				&Line{
					Operator: "<=",
					Time1:    time.Date(2021, 6, 30, 19, 0, 0, 0, time.UTC),
					Price1:   decimal.NewFromFloat(34840.08), // +5
					Time2:    time.Date(2021, 7, 10, 20, 0, 0, 0, time.UTC),
					Price2:   decimal.NewFromFloat(33009.78), // +5
				},
				&Line{
					Operator: ">=",
					Time1:    time.Date(2021, 6, 30, 19, 0, 0, 0, time.UTC),
					Price1:   decimal.NewFromFloat(34835.08),
					Time2:    time.Date(2021, 7, 10, 20, 0, 0, 0, time.UTC),
					Price2:   decimal.NewFromFloat(33004.78),
				},
			},
			time:        time.Date(2021, 6, 21, 1, 0, 0, 0, time.UTC), // 36612.21
			marketPrice: decimal.NewFromInt(36615),                    // 36613, 36614, 36615, 36616, 36617 can pass
			isTriggered: true,
		},
	}
	for _, tc := range testcases {
		result := IsTriggeredByMultipleTriggers(tc.operator, tc.triggers, tc.time, tc.marketPrice)
		if result != tc.isTriggered {
			t.Errorf("TestIsTriggeredByMultipleTriggers case '%s' - expect '%t', but got '%t'", tc.title, result, tc.isTriggered)
		}
	}
}
