package trigger

import (
	"reflect"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

func TestNewLine(t *testing.T) {
	testcases := []struct {
		title         string
		params        map[string]interface{}
		expectedError bool
	}{
		{
			title: "valid params",
			params: map[string]interface{}{
				"operator": ">=",
				"time_1":   "2021-07-02 23:00:00",
				"price_1":  float64(33620),
				"time_2":   "2021-07-04 02:00:00",
				"price_2":  float64(34387),
			},
			expectedError: false,
		},
		{
			title: "missing operator",
			params: map[string]interface{}{
				"time_1":  "2021-07-02 23:00:00",
				"price_1": float64(33620),
				"time_2":  "2021-07-04 02:00:00",
				"price_2": float64(34387),
			},
			expectedError: true,
		},
		{
			title: "missing time_1",
			params: map[string]interface{}{
				"operator": ">=",
				"price_1":  float64(33620),
				"time_2":   "2021-07-04 02:00:00",
				"price_2":  float64(34387),
			},
			expectedError: true,
		},
		{
			title: "missing price_1",
			params: map[string]interface{}{
				"operator": ">=",
				"time_1":   "2021-07-02 23:00:00",
				"time_2":   "2021-07-04 02:00:00",
				"price_2":  float64(34387),
			},
			expectedError: true,
		},
		{
			title: "missing time_2",
			params: map[string]interface{}{
				"operator": ">=",
				"time_1":   "2021-07-02 23:00:00",
				"price_1":  float64(33620),
				"price_2":  float64(34387),
			},
			expectedError: true,
		},
		{
			title: "missing price_2",
			params: map[string]interface{}{
				"operator": ">=",
				"time_1":   "2021-07-02 23:00:00",
				"price_1":  float64(33620),
				"time_2":   "2021-07-04 02:00:00",
			},
			expectedError: true,
		},
		{
			title: "time_1 is later than time_2",
			params: map[string]interface{}{
				"operator": ">=",
				"time_1":   "2021-07-02 23:00:00",
				"price_1":  float64(33620),
				"time_2":   "2021-07-01 02:00:00",
				"price_2":  float64(34387),
			},
			expectedError: true,
		},
	}

	for _, tc := range testcases {
		_, err := newLine(tc.params)
		hasError := (err != nil)
		if tc.expectedError != hasError {
			t.Errorf("TestNewLine case '%s' - expect '%t', but got '%t'", tc.title, tc.expectedError, hasError)
		}
	}
}

func TestLineGetPrice(t *testing.T) {
	testcases := []struct {
		title         string
		trigger       Trigger
		time          time.Time
		expectedPrice decimal.Decimal
	}{
		{
			title: "trigger_type: line (uptrend), before time_1",
			trigger: &Line{
				Operator: ">=",
				Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(33874.98),
				Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(38443.27),
			},
			time:          time.Date(2021, 7, 21, 17, 0, 0, 0, time.UTC),
			expectedPrice: decimal.NewFromFloat(30144.542314410480331391),
		},
		{
			title: "trigger_type: line (uptrend), before time_1",
			trigger: &Line{
				Operator: ">=",
				Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(33874.98),
				Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(38443.27),
			},
			time:          time.Date(2021, 7, 23, 16, 30, 0, 0, time.UTC),
			expectedPrice: decimal.NewFromFloat(32039.684454148471589787),
		},
		{
			title: "trigger_type: line (uptrend), equal to time_1",
			trigger: &Line{
				Operator: ">=",
				Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(33874.98),
				Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(38443.27),
			},
			time:          time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
			expectedPrice: decimal.NewFromFloat(33874.98),
		},
		{
			title: "trigger_type: line (uptrend), during time period",
			trigger: &Line{
				Operator: ">=",
				Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(33874.98),
				Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(38443.27),
			},
			time:          time.Date(2021, 7, 26, 2, 30, 0, 0, time.UTC),
			expectedPrice: decimal.NewFromFloat(34353.752751091703242293),
		},
		{
			title: "trigger_type: line (uptrend), during time period",
			trigger: &Line{
				Operator: ">=",
				Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(33874.98),
				Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(38443.27),
			},
			time:          time.Date(2021, 7, 27, 19, 30, 0, 0, time.UTC),
			expectedPrice: decimal.NewFromFloat(35989.55965065502185401),
		},
		{
			title: "trigger_type: line (uptrend), during time period",
			trigger: &Line{
				Operator: ">=",
				Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(33874.98),
				Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(38443.27),
			},
			time:          time.Date(2021, 7, 29, 17, 0, 0, 0, time.UTC),
			expectedPrice: decimal.NewFromFloat(37804.906331877729467105),
		},
		{
			title: "trigger_type: line (uptrend), equal to time_2",
			trigger: &Line{
				Operator: ">=",
				Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(33874.98),
				Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(38443.27),
			},
			time:          time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
			expectedPrice: decimal.NewFromFloat(38443.27),
		},
		{
			title: "trigger_type: line (uptrend), equal to time_2",
			trigger: &Line{
				Operator: ">=",
				Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(33874.98),
				Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(38443.27),
			},
			time:          time.Date(2021, 7, 30, 16, 0, 0, 0, time.UTC),
			expectedPrice: decimal.NewFromFloat(38722.554104803493443797),
		},
		{
			title: "trigger_type: line (uptrend), equal to time_2",
			trigger: &Line{
				Operator: ">=",
				Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(33874.98),
				Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(38443.27),
			},
			time:          time.Date(2021, 8, 1, 20, 30, 0, 0, time.UTC),
			expectedPrice: decimal.NewFromFloat(40817.184890829694500689),
		},
		{
			title: "trigger_type: line (downtrend), before time_1",
			trigger: &Line{
				Operator: ">=",
				Time1:    time.Date(2021, 6, 30, 19, 0, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(34835.08),
				Time2:    time.Date(2021, 7, 10, 20, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(33004.78),
			},
			time:          time.Date(2021, 6, 21, 1, 0, 0, 0, time.UTC),
			expectedPrice: decimal.NewFromFloat(36612.2177593360995319),
		},
		{
			title: "trigger_type: line (downtrend), before time_1",
			trigger: &Line{
				Operator: ">=",
				Time1:    time.Date(2021, 6, 30, 19, 0, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(34835.08),
				Time2:    time.Date(2021, 7, 10, 20, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(33004.78),
			},
			time:          time.Date(2021, 6, 26, 10, 0, 0, 0, time.UTC),
			expectedPrice: decimal.NewFromFloat(35632.51360995850628938),
		},
		{
			title: "trigger_type: line (downtrend), equal to time_1",
			trigger: &Line{
				Operator: ">=",
				Time1:    time.Date(2021, 6, 30, 19, 0, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(34835.08),
				Time2:    time.Date(2021, 7, 10, 20, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(33004.78),
			},
			time:          time.Date(2021, 6, 30, 19, 0, 0, 0, time.UTC),
			expectedPrice: decimal.NewFromFloat(34835.08),
		},
		{
			title: "trigger_type: line (downtrend), during time period",
			trigger: &Line{
				Operator: ">=",
				Time1:    time.Date(2021, 6, 30, 19, 0, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(34835.08),
				Time2:    time.Date(2021, 7, 10, 20, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(33004.78),
			},
			time:          time.Date(2021, 7, 2, 10, 0, 0, 0, time.UTC),
			expectedPrice: decimal.NewFromFloat(34538.89037344398347236),
		},
		{
			title: "trigger_type: line (downtrend), during time period",
			trigger: &Line{
				Operator: ">=",
				Time1:    time.Date(2021, 6, 30, 19, 0, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(34835.08),
				Time2:    time.Date(2021, 7, 10, 20, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(33004.78),
			},
			time:          time.Date(2021, 7, 6, 17, 0, 0, 0, time.UTC),
			expectedPrice: decimal.NewFromFloat(33756.64597510373445958),
		},
		{
			title: "trigger_type: line (downtrend), equal to time_2",
			trigger: &Line{
				Operator: ">=",
				Time1:    time.Date(2021, 6, 30, 19, 0, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(34835.08),
				Time2:    time.Date(2021, 7, 10, 20, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(33004.78),
			},
			time:          time.Date(2021, 7, 10, 20, 0, 0, 0, time.UTC),
			expectedPrice: decimal.NewFromFloat(33004.78),
		},
		{
			title: "trigger_type: line (downtrend), after time_2",
			trigger: &Line{
				Operator: ">=",
				Time1:    time.Date(2021, 6, 30, 19, 0, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(34835.08),
				Time2:    time.Date(2021, 7, 10, 20, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(33004.78),
			},
			time:          time.Date(2021, 7, 12, 6, 0, 0, 0, time.UTC),
			expectedPrice: decimal.NewFromFloat(32746.56340248962648083),
		},
		{
			title: "trigger_type: line (downtrend), after time_2",
			trigger: &Line{
				Operator: ">=",
				Time1:    time.Date(2021, 6, 30, 19, 0, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(34835.08),
				Time2:    time.Date(2021, 7, 10, 20, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(33004.78),
			},
			time:          time.Date(2021, 7, 16, 15, 0, 0, 0, time.UTC),
			expectedPrice: decimal.NewFromFloat(31949.12979253112037448),
		},
	}

	for _, tc := range testcases {
		p := tc.trigger.GetPrice(tc.time)
		if tc.expectedPrice.StringFixed(8) != p.StringFixed(8) {
			t.Errorf("TestLineGetPrice case '%s' - expect '%s', but got '%s'", tc.title, tc.expectedPrice.String(), p.String())
		}
	}
}

func TestLineGetOperator(t *testing.T) {
	testcases := []struct {
		title            string
		trigger          Trigger
		expectedOperator string
		expectedPrice    decimal.Decimal
	}{
		{
			title: "trigger_type: line",
			trigger: &Line{
				Operator: ">=",
				Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(33874.98),
				Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(38443.27),
			},
			expectedOperator: ">=",
		},
		{
			title: "trigger_type: line",
			trigger: &Line{
				Operator: "<=",
				Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(33874.98),
				Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(38443.27),
			},
			expectedOperator: "<=",
		},
	}

	for _, tc := range testcases {
		o := tc.trigger.GetOperator()
		if o != tc.expectedOperator {
			t.Errorf("TestLineGetOperator case '%s' - expect '%s', but got '%s'", tc.title, tc.expectedOperator, o)
		}
	}
}

func TestLineSetOperator(t *testing.T) {
	trigger := &Line{
		Operator: "<=",
		Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
		Price1:   decimal.NewFromFloat(33874.98),
		Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
		Price2:   decimal.NewFromFloat(38443.27),
	}
	expectedOperator := ">="
	trigger.SetOperator(expectedOperator)
	if expectedOperator != trigger.GetOperator() {
		t.Errorf("TestLineSetOperator - expect '%s', but got '%s'", expectedOperator, trigger.GetOperator())
	}
}

func TestLineReadjustPrice(t *testing.T) {
	testcases := []struct {
		title           string
		price           decimal.Decimal
		t               time.Time
		trigger         Line
		expectedTrigger Line
	}{
		{
			title: "readjust time_2 and price_2",
			price: decimal.NewFromFloat(31235),
			t:     time.Date(2021, 7, 29, 12, 0, 0, 0, time.UTC),
			trigger: Line{
				Operator: ">=",
				Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(33874.98),
				Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(38443.27),
			},
			expectedTrigger: Line{
				Operator: ">=",
				Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(33874.98),
				Time2:    time.Date(2021, 7, 29, 12, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(31235),
			},
		},
	}

	for _, tc := range testcases {
		tc.trigger.ReadjustPrice(tc.price, tc.t)
		if !reflect.DeepEqual(tc.expectedTrigger, tc.trigger) {
			t.Errorf("TestLineReadjustPrice case '%s' - trigger and expectedTrigger aren't equal", tc.title)
		}
	}
}

func TestLineUpdatePriceByPercent(t *testing.T) {
	testcases := []struct {
		title           string
		percent         decimal.Decimal
		trigger         Line
		expectedTrigger Line
	}{
		{
			title:   "update price1 and price2 by percent",
			percent: decimal.NewFromFloat(1.0001),
			trigger: Line{
				Operator: ">=",
				Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(1100.2),
				Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(1200.3),
			},
			expectedTrigger: Line{
				Operator: ">=",
				Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(1100.31002),
				Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(1200.42003),
			},
		},
	}

	for _, tc := range testcases {
		tc.trigger.UpdatePriceByPercent(tc.percent)
		if !reflect.DeepEqual(tc.expectedTrigger, tc.trigger) {
			t.Errorf("TestLineUpdatePriceByPercent case '%s' - trigger and expectedTrigger aren't equal", tc.title)
		}
	}
}

func TestLineClone(t *testing.T) {
	source := &Line{
		Operator: ">=",
		Time1:    time.Date(2021, 7, 25, 14, 30, 0, 0, time.UTC),
		Price1:   decimal.NewFromFloat(33874.98),
		Time2:    time.Date(2021, 7, 30, 9, 0, 0, 0, time.UTC),
		Price2:   decimal.NewFromFloat(38443.27),
	}

	// Clone trigger from source
	time := time.Date(2021, 8, 2, 11, 30, 0, 0, time.UTC)
	clone := source.Clone()
	clone.ReadjustPrice(decimal.NewFromFloat(37882.12), time)

	if reflect.DeepEqual(clone, source) {
		t.Error("TestLimitClone - trigger and expectedTrigger are equal")
	}
}
