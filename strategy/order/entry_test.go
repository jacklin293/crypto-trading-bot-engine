package order

import (
	"crypto-trading-bot-main/strategy/trigger"
	"reflect"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestNewEntry(t *testing.T) {
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
					"price":        47200.23,
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
				"baseline_trigger": map[string]interface{}{
					"trigger_type": "line",
					"operator":     ">=",
					"time_1":       "2021-08-18 18:00:00",
					"price_1":      46000.23,
					"time_2":       "2021-08-19 01:45:00",
					"price_2":      45234.56,
				},
				"baseline_offset_percent": 0.005,
			},
			expectedError: false,
		},
		{
			title:     "new baseline trigger - 'baseline_trigger' is missing",
			entryType: ENTRY_BASELINE,
			data: map[string]interface{}{
				"baseline_offset_percent": 0.005,
			},
			expectedError: true,
		},
		{
			title:     "new baseline trigger - 'baseline_offset_percent' is missing",
			entryType: ENTRY_BASELINE,
			data: map[string]interface{}{
				"baseline_trigger": map[string]interface{}{
					"trigger_type": "line",
					"operator":     ">=",
					"time_1":       "2021-08-18 18:00:00",
					"price_1":      46000.23,
					"time_2":       "2021-08-19 01:45:00",
					"price_2":      45234.56,
				},
			},
			expectedError: true,
		},
	}

	for _, tc := range testcases {
		_, err := NewEntry(LONG, tc.entryType, tc.data)
		hasError := (err != nil)
		if tc.expectedError != hasError {
			t.Errorf("TestNewEntry case '%s' - expect '%t', but got '%t'", tc.title, tc.expectedError, hasError)
		}
	}
}

func TestEntryGetSetTrigger(t *testing.T) {
	expectedTrigger := &trigger.Limit{
		Operator: "<=",
		Price:    decimal.NewFromFloat(47200.23),
	}
	o := &Entry{}
	o.SetTrigger(expectedTrigger)
	trigger := o.GetTrigger()

	if !reflect.DeepEqual(expectedTrigger, trigger) {
		t.Errorf("TestEntryGetSetTrigger - expect '%v', but got '%v'", expectedTrigger, trigger)
	}
}

func TestEntryIsTriggered(t *testing.T) {
	testcases := getIsTriggeredTestCases()
	for _, tc := range testcases {
		o := Entry{
			Trigger: tc.trigger,
		}
		triggered := o.IsTriggered(tc.time, tc.price)

		if tc.expectedTriggered != triggered {
			t.Errorf("TestEntryIsTriggered case '%s' - expect '%t', but got '%t'", tc.title, tc.expectedTriggered, triggered)
		}
	}
}

func TestEntryUpdateBaselineTrigger(t *testing.T) {
	testcases := []struct {
		title                   string
		contractDirection       ContractDirection
		baselineTrigger         trigger.Trigger
		price2                  decimal.Decimal
		time2                   time.Time
		expectedBaselineTrigger trigger.Trigger
	}{
		{
			title:             "long - price1 > price2",
			contractDirection: LONG,
			baselineTrigger: &trigger.Line{
				Operator: ">=",
				Time1:    time.Date(2021, 8, 29, 1, 15, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(49632.27),
				Time2:    time.Date(2021, 8, 30, 20, 15, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(48696.87),
			},
			price2: decimal.NewFromFloat(48900.87),
			time2:  time.Date(2021, 9, 1, 9, 30, 0, 0, time.UTC),
			expectedBaselineTrigger: &trigger.Line{
				Operator: ">=",
				Time1:    time.Date(2021, 8, 29, 1, 15, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(49632.27),
				Time2:    time.Date(2021, 9, 1, 9, 30, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(48900.87),
			},
		},
		{
			title:             "long - price1 < price2",
			contractDirection: LONG,
			baselineTrigger: &trigger.Line{
				Operator: ">=",
				Time1:    time.Date(2021, 8, 29, 1, 15, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(49632.27),
				Time2:    time.Date(2021, 8, 30, 20, 15, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(48696.87),
			},
			price2: decimal.NewFromFloat(49700.26),
			time2:  time.Date(2021, 9, 1, 9, 30, 0, 0, time.UTC),
			expectedBaselineTrigger: &trigger.Line{
				Operator: ">=",
				Time1:    time.Date(2021, 8, 29, 1, 15, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(49632.27),
				Time2:    time.Date(2021, 9, 1, 9, 30, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(49632.27),
			},
		},
		{
			title:             "short - price1 < price2",
			contractDirection: SHORT,
			baselineTrigger: &trigger.Line{
				Operator: "<=",
				Time1:    time.Date(2021, 8, 27, 0, 15, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(46348.44),
				Time2:    time.Date(2021, 8, 29, 4, 00, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(47762.54),
			},
			price2: decimal.NewFromFloat(46500.37),
			time2:  time.Date(2021, 9, 1, 9, 30, 0, 0, time.UTC),
			expectedBaselineTrigger: &trigger.Line{
				Operator: "<=",
				Time1:    time.Date(2021, 8, 27, 0, 15, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(46348.44),
				Time2:    time.Date(2021, 9, 1, 9, 30, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(46500.37),
			},
		},
		{
			title:             "short - price1 > price2",
			contractDirection: SHORT,
			baselineTrigger: &trigger.Line{
				Operator: "<=",
				Time1:    time.Date(2021, 8, 27, 0, 15, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(46348.44),
				Time2:    time.Date(2021, 8, 29, 4, 00, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(47762.54),
			},
			price2: decimal.NewFromFloat(46100.37),
			time2:  time.Date(2021, 9, 1, 9, 30, 0, 0, time.UTC),
			expectedBaselineTrigger: &trigger.Line{
				Operator: "<=",
				Time1:    time.Date(2021, 8, 27, 0, 15, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(46348.44),
				Time2:    time.Date(2021, 9, 1, 9, 30, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(46348.44),
			},
		},
	}

	for _, tc := range testcases {
		o := Entry{
			BaselineTrigger: tc.baselineTrigger,
		}
		o.UpdateBaselineTrigger(tc.contractDirection, tc.price2, tc.time2)

		if !reflect.DeepEqual(tc.expectedBaselineTrigger, o.BaselineTrigger) {
			t.Errorf("TestEntryUpdateBaselineTrigger case '%s' - expect '%v', but got '%v'", tc.title, tc.expectedBaselineTrigger, o.BaselineTrigger)
		}
	}
}

func TestEntryUpdateTriggerByBaselineAndOffset(t *testing.T) {
	testcases := []struct {
		title             string
		contractDirection ContractDirection
		percent           float64
		expectedTrigger   trigger.Trigger
	}{
		{
			title:             "long - positive percent",
			contractDirection: LONG,
			percent:           0.01,
			expectedTrigger: &trigger.Line{
				Operator: "<=",
				Time1:    time.Date(2021, 8, 27, 0, 15, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(46811.9244),
				Time2:    time.Date(2021, 8, 29, 4, 00, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(48240.1654),
			},
		},
		{
			title:             "long - negative percent",
			contractDirection: LONG,
			percent:           -0.01,
			expectedTrigger: &trigger.Line{
				Operator: "<=",
				Time1:    time.Date(2021, 8, 27, 0, 15, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(45884.9556),
				Time2:    time.Date(2021, 8, 29, 4, 00, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(47284.9146),
			},
		},
		{
			title:             "short - positive percent",
			contractDirection: SHORT,
			percent:           0.01,
			expectedTrigger: &trigger.Line{
				Operator: "<=",
				Time1:    time.Date(2021, 8, 27, 0, 15, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(45884.9556),
				Time2:    time.Date(2021, 8, 29, 4, 00, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(47284.9146),
			},
		},
		{
			title:             "short - negative percent",
			contractDirection: SHORT,
			percent:           -0.01,
			expectedTrigger: &trigger.Line{
				Operator: "<=",
				Time1:    time.Date(2021, 8, 27, 0, 15, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(46811.9244),
				Time2:    time.Date(2021, 8, 29, 4, 00, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(48240.1654),
			},
		},
	}

	for _, tc := range testcases {
		o := Entry{
			BaselineTrigger: &trigger.Line{
				Operator: "<=",
				Time1:    time.Date(2021, 8, 27, 0, 15, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(46348.44),
				Time2:    time.Date(2021, 8, 29, 4, 00, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(47762.54),
			},
			BaselineOffsetPercent: tc.percent,
		}
		o.UpdateTriggerByBaselineAndOffset(tc.contractDirection)

		if !reflect.DeepEqual(tc.expectedTrigger, o.Trigger) {
			t.Errorf("TestEntryUpdateTriggerByBaselineAndOffset case '%s' - expect '%v', but got '%v'", tc.title, tc.expectedTrigger, o.Trigger)
		}
	}
}

func TestEntryUpdateOperator(t *testing.T) {
	testcases := []struct {
		title                   string
		contractDirection       ContractDirection
		trigger                 trigger.Trigger
		baselineTrigger         trigger.Trigger
		expectedTrigger         trigger.Trigger
		expectedBaselineTrigger trigger.Trigger
	}{
		{
			title:             "long - trigger & baseline trigger",
			contractDirection: LONG,
			trigger: &trigger.Limit{
				Operator: "<=",
				Price:    decimal.NewFromInt(100),
			},
			baselineTrigger: &trigger.Limit{
				Operator: "<=",
				Price:    decimal.NewFromInt(100),
			},
			expectedTrigger: &trigger.Limit{
				Operator: ">=",
				Price:    decimal.NewFromInt(100),
			},
			expectedBaselineTrigger: &trigger.Limit{
				Operator: ">=",
				Price:    decimal.NewFromInt(100),
			},
		},
		{
			title:             "long - trigger only",
			contractDirection: LONG,
			trigger: &trigger.Limit{
				Operator: "<=",
				Price:    decimal.NewFromInt(100),
			},
			expectedTrigger: &trigger.Limit{
				Operator: ">=",
				Price:    decimal.NewFromInt(100),
			},
		},
		{
			title:             "short - trigger & baseline trigger",
			contractDirection: SHORT,
			trigger: &trigger.Line{
				Operator: ">=",
				Time1:    time.Date(2021, 8, 27, 0, 15, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(46348.44),
				Time2:    time.Date(2021, 8, 29, 4, 00, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(47762.54),
			},
			baselineTrigger: &trigger.Line{
				Operator: ">=",
				Time1:    time.Date(2021, 8, 27, 0, 15, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(46348.44),
				Time2:    time.Date(2021, 8, 29, 4, 00, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(47762.54),
			},
			expectedTrigger: &trigger.Line{
				Operator: "<=",
				Time1:    time.Date(2021, 8, 27, 0, 15, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(46348.44),
				Time2:    time.Date(2021, 8, 29, 4, 00, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(47762.54),
			},
			expectedBaselineTrigger: &trigger.Line{
				Operator: "<=",
				Time1:    time.Date(2021, 8, 27, 0, 15, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(46348.44),
				Time2:    time.Date(2021, 8, 29, 4, 00, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(47762.54),
			},
		},
		{
			title:             "short - trigger only",
			contractDirection: SHORT,
			trigger: &trigger.Line{
				Operator: ">=",
				Time1:    time.Date(2021, 8, 27, 0, 15, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(46348.44),
				Time2:    time.Date(2021, 8, 29, 4, 00, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(47762.54),
			},
			expectedTrigger: &trigger.Line{
				Operator: "<=",
				Time1:    time.Date(2021, 8, 27, 0, 15, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(46348.44),
				Time2:    time.Date(2021, 8, 29, 4, 00, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(47762.54),
			},
		},
	}

	for _, tc := range testcases {
		entryOrder := &Entry{
			Trigger:         tc.trigger,
			BaselineTrigger: tc.baselineTrigger,
		}
		entryOrder.UpdateOperator(tc.contractDirection)

		if !reflect.DeepEqual(tc.expectedTrigger, entryOrder.Trigger) {
			t.Errorf("TestEntryUpdateOperator case '%s' - expect '%v', but got '%v'", tc.title, tc.expectedTrigger, entryOrder.Trigger)
		}
		if !reflect.DeepEqual(tc.expectedBaselineTrigger, entryOrder.BaselineTrigger) {
			t.Errorf("TestEntryUpdateOperator case '%s' - expect '%v', but got '%v'", tc.title, tc.expectedBaselineTrigger, entryOrder.BaselineTrigger)
		}
	}
}
