package contract

import (
	"crypto-trading-bot-engine/strategy/order"
	"crypto-trading-bot-engine/strategy/trigger"
	"reflect"
	"testing"
	"time"

	"github.com/shopspring/decimal"
)

type testFeed struct {
	// TODO make price and time as Mark
	price         decimal.Decimal
	time          time.Time
	expectedHooks []string
}

// For storing the func names that are triggered and being used to compare with expected results
type testHook struct {
	funcNames []string
}

func (th *testHook) resetFuncNames() {
	th.funcNames = nil
}

func (th *testHook) EntryTriggered(c *Contract, t time.Time, p decimal.Decimal) (decimal.Decimal, bool, error) {
	th.funcNames = append(th.funcNames, "EntryTriggered")
	return p, false, nil
}

func (th *testHook) StopLossTriggerCreated(c *Contract) (bool, error) {
	th.funcNames = append(th.funcNames, "StopLossTriggerCreated")
	return false, nil
}

func (th *testHook) StopLossTriggered(c *Contract) (bool, error) {
	th.funcNames = append(th.funcNames, "StopLossTriggered")
	return false, nil
}

func (th *testHook) EntryBaselineTriggerUpdated(c *Contract) {
	th.funcNames = append(th.funcNames, "EntryBaselineTriggerUpdated")
}

func (th *testHook) TakeProfitTriggered(c *Contract) error {
	th.funcNames = append(th.funcNames, "TakeProfitTriggered")
	return nil
}

func (th *testHook) ParamsUpdated(c *Contract) (bool, error) {
	return false, nil
}

func (th *testHook) BreakoutPeakUpdated(c *Contract) {
}

// entry_type 'limit'
func TestLimitAllOrders(t *testing.T) {
	testcases := []struct {
		title           string
		side            order.Side
		takeProfitOrder order.Order
		entryOrder      order.Order
		stopLossOrder   order.Order
		feeds           []testFeed
	}{
		{
			title: "long - (breakout) with stop-loss and take-profit order",
			side:  order.LONG,
			takeProfitOrder: &order.TakeProfit{Trigger: &trigger.Limit{
				Operator: ">=",
				Price:    decimal.NewFromFloat(48000),
			}},
			entryOrder: &order.Entry{Trigger: &trigger.Limit{
				Operator: ">=",
				Price:    decimal.NewFromFloat(47000),
			}},
			stopLossOrder: &order.StopLoss{Trigger: &trigger.Limit{
				Operator: "<=",
				Price:    decimal.NewFromFloat(46000),
			}},
			feeds: []testFeed{
				// time doesn't matter for 'limit'
				{price: decimal.NewFromFloat(46000), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(46999), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(47000), time: time.Now(), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}},
				{price: decimal.NewFromFloat(46001), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(46000), time: time.Now(), expectedHooks: []string{"StopLossTriggered"}},
				{price: decimal.NewFromFloat(46999), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(47000), time: time.Now(), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}},
				{price: decimal.NewFromFloat(47999), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(48000), time: time.Now(), expectedHooks: []string{"TakeProfitTriggered"}},
			},
		},
		{
			title: "long - (breakout) without stop-loss and with take-profit order",
			side:  order.LONG,
			takeProfitOrder: &order.TakeProfit{Trigger: &trigger.Limit{
				Operator: ">=",
				Price:    decimal.NewFromFloat(48000),
			}},
			entryOrder: &order.Entry{Trigger: &trigger.Limit{
				Operator: ">=",
				Price:    decimal.NewFromFloat(47000),
			}},
			feeds: []testFeed{
				// time doesn't matter for 'limit'
				{price: decimal.NewFromFloat(46999), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(47000), time: time.Now(), expectedHooks: []string{"EntryTriggered"}},
				{price: decimal.NewFromFloat(30000), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(47999), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(48000), time: time.Now(), expectedHooks: []string{"TakeProfitTriggered"}},
			},
		},
		{
			title: "long - (breakout) with stop-loss and without take-profit order",
			side:  order.LONG,
			entryOrder: &order.Entry{Trigger: &trigger.Limit{
				Operator: ">=",
				Price:    decimal.NewFromFloat(47000),
			}},
			stopLossOrder: &order.StopLoss{Trigger: &trigger.Limit{
				Operator: "<=",
				Price:    decimal.NewFromFloat(46000),
			}},
			feeds: []testFeed{
				// time doesn't matter for 'limit'
				{price: decimal.NewFromFloat(46999), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(47000), time: time.Now(), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}},
				{price: decimal.NewFromFloat(46001), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(46000), time: time.Now(), expectedHooks: []string{"StopLossTriggered"}},
				{price: decimal.NewFromFloat(46999), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(47000), time: time.Now(), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}},
				{price: decimal.NewFromFloat(100000), time: time.Now(), expectedHooks: nil},
			},
		},
		{
			title: "long - (breakout) without stop-loss and take-profit order",
			side:  order.LONG,
			entryOrder: &order.Entry{Trigger: &trigger.Limit{
				Operator: ">=",
				Price:    decimal.NewFromFloat(47000),
			}},
			feeds: []testFeed{
				// time doesn't matter for 'limit'
				{price: decimal.NewFromFloat(46000), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(47000), time: time.Now(), expectedHooks: []string{"EntryTriggered"}},
				{price: decimal.NewFromFloat(20000), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(100000), time: time.Now(), expectedHooks: nil},
			},
		},
		{
			title: "short - (breakout) with stop-loss and take-profit order",
			side:  order.SHORT,
			stopLossOrder: &order.StopLoss{Trigger: &trigger.Limit{
				Operator: ">=",
				Price:    decimal.NewFromFloat(48000),
			}},
			entryOrder: &order.Entry{Trigger: &trigger.Limit{
				Operator: "<=",
				Price:    decimal.NewFromFloat(47000),
			}},
			takeProfitOrder: &order.TakeProfit{Trigger: &trigger.Limit{
				Operator: "<=",
				Price:    decimal.NewFromFloat(46000),
			}},
			feeds: []testFeed{
				// time doesn't matter for 'limit'
				{price: decimal.NewFromFloat(47001), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(47000), time: time.Now(), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}},
				{price: decimal.NewFromFloat(47999), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(48000), time: time.Now(), expectedHooks: []string{"StopLossTriggered"}},
				{price: decimal.NewFromFloat(47999), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(47000), time: time.Now(), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}},
				{price: decimal.NewFromFloat(46001), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(46000), time: time.Now(), expectedHooks: []string{"TakeProfitTriggered"}},
			},
		},
		{
			title: "short - (breakout) without stop-loss and with take-profit order",
			side:  order.SHORT,
			entryOrder: &order.Entry{Trigger: &trigger.Limit{
				Operator: "<=",
				Price:    decimal.NewFromFloat(47000),
			}},
			takeProfitOrder: &order.TakeProfit{Trigger: &trigger.Limit{
				Operator: "<=",
				Price:    decimal.NewFromFloat(46000),
			}},
			feeds: []testFeed{
				// time doesn't matter for 'limit'
				{price: decimal.NewFromFloat(47001), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(47000), time: time.Now(), expectedHooks: []string{"EntryTriggered"}},
				{price: decimal.NewFromFloat(100000), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(46001), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(46000), time: time.Now(), expectedHooks: []string{"TakeProfitTriggered"}},
			},
		},
		{
			title: "short - (breakout) with stop-loss and without take-profit order",
			side:  order.SHORT,
			stopLossOrder: &order.StopLoss{Trigger: &trigger.Limit{
				Operator: ">=",
				Price:    decimal.NewFromFloat(48000),
			}},
			entryOrder: &order.Entry{Trigger: &trigger.Limit{
				Operator: "<=",
				Price:    decimal.NewFromFloat(47000),
			}},
			feeds: []testFeed{
				// time doesn't matter for 'limit'
				{price: decimal.NewFromFloat(47001), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(47000), time: time.Now(), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}},
				{price: decimal.NewFromFloat(47999), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(48000), time: time.Now(), expectedHooks: []string{"StopLossTriggered"}},
				{price: decimal.NewFromFloat(47000), time: time.Now(), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}},
				{price: decimal.NewFromFloat(47999), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(10000), time: time.Now(), expectedHooks: nil},
			},
		},
		{
			title: "short - (breakout) without stop-loss and take-profit order",
			side:  order.SHORT,
			entryOrder: &order.Entry{Trigger: &trigger.Limit{
				Operator: "<=",
				Price:    decimal.NewFromFloat(47000),
			}},
			feeds: []testFeed{
				// time doesn't matter for 'limit'
				{price: decimal.NewFromFloat(47001), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(47000), time: time.Now(), expectedHooks: []string{"EntryTriggered"}},
				{price: decimal.NewFromFloat(100000), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(10000), time: time.Now(), expectedHooks: nil},
			},
		},
	}

	for _, tc := range testcases {
		c := &Contract{
			Side:            tc.side,
			EntryType:       order.ENTRY_LIMIT,
			EntryOrder:      tc.entryOrder,
			TakeProfitOrder: tc.takeProfitOrder,
			StopLossOrder:   tc.stopLossOrder,
		}
		h := &testHook{}
		c.SetHook(h)

		for i, feed := range tc.feeds {
			c.CheckPrice(Mark{Time: feed.time, Price: feed.price})
			if !reflect.DeepEqual(feed.expectedHooks, h.funcNames) {
				t.Errorf("TestLimitAllOrders case '%s' (%d) - expect '%v', but got '%v'", tc.title, i, feed.expectedHooks, h.funcNames)
			}
			// Reset func names so that we can get fresh hooks each feed
			h.resetFuncNames()
		}
	}
}

// entry_type 'baseline'
/*
LONG
{
  "entry_type": "baseline",
  "entry_order": {
    "baseline_trigger": {
      "trigger_type": "line",
      "operator": ">=",
      "time_1": "2021-08-17 11:45:00",
      "price_1": 47160,
      "time_2": "2021-08-18 10:00:00",
      "price_2": 45560
    },
    "baseline_offset_percent": 0.01
  },
  "stop_loss_order": {
    "loss_tolerance_percent": 0.01,
    "baseline_readjustment_enabled": false
  },
  "take_profit_order": {
    "trigger": {
      "trigger_type": "limit",
      "operator": ">=",
      "price": 46195
    }
  }
}

EntryTriggered             baseline: 45144.12  &{>= 2021-08-17 11:45:00 +0000 UTC 47160 2021-08-18 10:00:00 +0000 UTC 45560}
EntryTriggered                entry: 45595.56  &{>= 2021-08-17 11:45:00 +0000 UTC 47631.6 2021-08-18 10:00:00 +0000 UTC 46015.6}
EntryTriggered                  buy: 45727.76  '2021-08-18 15:47'
StopLossTriggerCreated    stop-loss: 45270.48  <=
! StopLossTriggered            sell: 45189.23  '2021-08-18 19:18'  ($1000 => $988)
EntryTriggered             baseline: 44590.41  &{>= 2021-08-17 11:45:00 +0000 UTC 47160 2021-08-18 10:00:00 +0000 UTC 45560}
EntryTriggered                entry: 45036.32  &{>= 2021-08-17 11:45:00 +0000 UTC 47631.6 2021-08-18 10:00:00 +0000 UTC 46015.6}
EntryTriggered                  buy: 45073.46  '2021-08-18 23:29'
StopLossTriggerCreated    stop-loss: 44622.73  <=
! StopLossTriggered            sell: 44600  '2021-08-19 00:15'  ($988 => $978)
EntryTriggered             baseline: 44493.33  &{>= 2021-08-17 11:45:00 +0000 UTC 47160 2021-08-18 10:00:00 +0000 UTC 45560}
EntryTriggered                entry: 44938.27  &{>= 2021-08-17 11:45:00 +0000 UTC 47631.6 2021-08-18 10:00:00 +0000 UTC 46015.6}
EntryTriggered                  buy: 44976.25  '2021-08-19 00:50'
StopLossTriggerCreated    stop-loss: 44526.49  <=
! StopLossTriggered            sell: 44496.45  '2021-08-19 03:06'  ($978 => $967)
EntryTriggered             baseline: 44043.90  &{>= 2021-08-17 11:45:00 +0000 UTC 47160 2021-08-18 10:00:00 +0000 UTC 45560}
EntryTriggered                entry: 44484.33  &{>= 2021-08-17 11:45:00 +0000 UTC 47631.6 2021-08-18 10:00:00 +0000 UTC 46015.6}
EntryTriggered                  buy: 44485.49  '2021-08-19 07:05'
StopLossTriggerCreated    stop-loss: 44040.64  <=
! TakeProfitTriggered          sell: 46392.42  '2021-08-19 18:25'  ($967 => $1009)


SHORT
{
  "entry_type": "baseline",
  "entry_order": {
    "baseline_trigger": {
      "trigger_type": "line",
      "operator": "<=",
      "time_1": "2021-08-09 01:15:00",
      "price_1": 42779,
      "time_2": "2021-08-10 16:30:00",
      "price_2": 44589.46
    },
    "baseline_offset_percent": 0.01
  },
  "stop_loss_order": {
    "loss_tolerance_percent": 0.01,
    "baseline_readjustment_enabled": false
  },
  "take_profit_order": {
    "trigger": {
      "trigger_type": "limit",
      "operator": "<=",
      "price": 44000
    }
  }
}

EntryTriggered             baseline: 46040.90  &{<= 2021-08-09 01:15:00 +0000 UTC 42779 2021-08-10 16:30:00 +0000 UTC 44589.46}
EntryTriggered                entry: 45580.49  &{<= 2021-08-09 01:15:00 +0000 UTC 42351.21 2021-08-10 16:30:00 +0000 UTC 44143.5654}
EntryTriggered                  buy: 45550  '2021-08-11 23:58'
StopLossTriggerCreated    stop-loss: 46005.50  >=
! StopLossTriggered            sell: 46052.25  '2021-08-12 01:32'  ($1000 => $1011)
EntryTriggered             baseline: 46243.09  &{<= 2021-08-09 01:15:00 +0000 UTC 42779 2021-08-10 16:30:00 +0000 UTC 44589.46}
EntryTriggered                entry: 45780.66  &{<= 2021-08-09 01:15:00 +0000 UTC 42351.21 2021-08-10 16:30:00 +0000 UTC 44143.5654}
EntryTriggered                  buy: 45777.12  '2021-08-12 04:21'
StopLossTriggerCreated    stop-loss: 46234.89  >=
! TakeProfitTriggered          sell: 43957.73  '2021-08-12 14:57'  ($1011 => $971)

*/
func TestBaselineAllOrders(t *testing.T) {
	testcases := []struct {
		title           string
		side            order.Side
		takeProfitOrder order.Order
		entryData       map[string]interface{}
		stopLossOrder   order.Order
		feeds           []testFeed
	}{
		{
			title: "long - (breakout) with stop-loss and take-profit order",
			side:  order.LONG,
			takeProfitOrder: &order.TakeProfit{Trigger: &trigger.Limit{
				Operator: ">=",
				Price:    decimal.NewFromFloat(46300),
			}},
			entryData: map[string]interface{}{
				"baseline_trigger": map[string]interface{}{
					"trigger_type": "line",
					"operator":     ">=",
					"time_1":       "2021-08-17T11:45:00Z",
					"price_1":      "47160.0",
					"time_2":       "2021-08-18T10:00:00Z",
					"price_2":      "45560.0",
				},
				"baseline_offset_percent": 0.01,
			},
			stopLossOrder: &order.StopLoss{
				BaselineReadjustmentEnabled: false,
				LossTolerancePercent:        0.01,
			},
			feeds: []testFeed{
				{price: decimal.NewFromFloat(45595.56), time: time.Date(2021, 8, 18, 15, 46, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(45727.76), time: time.Date(2021, 8, 18, 15, 47, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}}, // &{<= 45270.4824}
				{price: decimal.NewFromFloat(45270.49), time: time.Date(2021, 8, 18, 19, 17, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(45270.48), time: time.Date(2021, 8, 18, 19, 18, 0, 0, time.UTC), expectedHooks: []string{"StopLossTriggered"}},
				{price: decimal.NewFromFloat(45036), time: time.Date(2021, 8, 18, 23, 28, 0, 0, time.UTC), expectedHooks: nil}, // 45037.5265917602996176
				{price: decimal.NewFromFloat(45073.46), time: time.Date(2021, 8, 18, 23, 29, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}},
				{price: decimal.NewFromFloat(45036), time: time.Date(2021, 8, 18, 23, 30, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(44600), time: time.Date(2021, 8, 19, 0, 15, 0, 0, time.UTC), expectedHooks: []string{"StopLossTriggered"}},
				{price: decimal.NewFromFloat(44485), time: time.Date(2021, 8, 19, 7, 4, 0, 0, time.UTC), expectedHooks: nil}, // 44485.5445692883894544
				{price: decimal.NewFromFloat(44485.49), time: time.Date(2021, 8, 19, 7, 5, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}},
				{price: decimal.NewFromFloat(46299), time: time.Date(2021, 8, 19, 18, 24, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(46300), time: time.Date(2021, 8, 19, 18, 25, 0, 0, time.UTC), expectedHooks: []string{"TakeProfitTriggered"}},
			},
		},
		{
			title: "long - (breakout) without stop-loss and with take-profit order",
			side:  order.LONG,
			takeProfitOrder: &order.TakeProfit{Trigger: &trigger.Limit{
				Operator: ">=",
				Price:    decimal.NewFromFloat(46300),
			}},
			entryData: map[string]interface{}{
				"baseline_trigger": map[string]interface{}{
					"trigger_type": "line",
					"operator":     ">=",
					"time_1":       "2021-08-17T11:45:00Z",
					"price_1":      "47160",
					"time_2":       "2021-08-18T10:00:00Z",
					"price_2":      "45560",
				},
				"baseline_offset_percent": 0.01,
			},
			feeds: []testFeed{
				{price: decimal.NewFromFloat(45595.56), time: time.Date(2021, 8, 18, 15, 46, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(45727.76), time: time.Date(2021, 8, 18, 15, 47, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered"}},
				{price: decimal.NewFromFloat(45270.49), time: time.Date(2021, 8, 18, 19, 17, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(45036), time: time.Date(2021, 8, 18, 23, 28, 0, 0, time.UTC), expectedHooks: nil}, // 45037.5265917602996176
				{price: decimal.NewFromFloat(44485), time: time.Date(2021, 8, 19, 7, 4, 0, 0, time.UTC), expectedHooks: nil},   // 44485.5445692883894544
				{price: decimal.NewFromFloat(46299), time: time.Date(2021, 8, 19, 18, 24, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(46300), time: time.Date(2021, 8, 19, 18, 25, 0, 0, time.UTC), expectedHooks: []string{"TakeProfitTriggered"}},
			},
		},
		{
			title: "long - (breakout) with stop-loss and without take-profit order",
			side:  order.LONG,
			entryData: map[string]interface{}{
				"baseline_trigger": map[string]interface{}{
					"trigger_type": "line",
					"operator":     ">=",
					"time_1":       "2021-08-17T11:45:00Z",
					"price_1":      "47160",
					"time_2":       "2021-08-18T10:00:00Z",
					"price_2":      "45560",
				},
				"baseline_offset_percent": 0.01,
			},
			stopLossOrder: &order.StopLoss{
				BaselineReadjustmentEnabled: false,
				LossTolerancePercent:        0.01,
			},
			feeds: []testFeed{
				{price: decimal.NewFromFloat(45595.56), time: time.Date(2021, 8, 18, 15, 46, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(45727.76), time: time.Date(2021, 8, 18, 15, 47, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}}, // &{<= 45270.4824}
				{price: decimal.NewFromFloat(45270.49), time: time.Date(2021, 8, 18, 19, 17, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(45270.48), time: time.Date(2021, 8, 18, 19, 18, 0, 0, time.UTC), expectedHooks: []string{"StopLossTriggered"}},
				{price: decimal.NewFromFloat(45036), time: time.Date(2021, 8, 18, 23, 28, 0, 0, time.UTC), expectedHooks: nil}, // 45037.5265917602996176
				{price: decimal.NewFromFloat(45073.46), time: time.Date(2021, 8, 18, 23, 29, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}},
				{price: decimal.NewFromFloat(45036), time: time.Date(2021, 8, 18, 23, 30, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(44600), time: time.Date(2021, 8, 19, 0, 15, 0, 0, time.UTC), expectedHooks: []string{"StopLossTriggered"}},
				{price: decimal.NewFromFloat(44485), time: time.Date(2021, 8, 19, 7, 4, 0, 0, time.UTC), expectedHooks: nil}, // 44485.5445692883894544
				{price: decimal.NewFromFloat(44485.49), time: time.Date(2021, 8, 19, 7, 5, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}},
				{price: decimal.NewFromFloat(46299), time: time.Date(2021, 8, 19, 18, 24, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(46300), time: time.Date(2021, 8, 19, 18, 25, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(100000), time: time.Date(2021, 8, 19, 18, 25, 0, 0, time.UTC), expectedHooks: nil},
			},
		},
		{
			title: "long - (breakout) without stop-loss and take-profit order",
			side:  order.LONG,
			entryData: map[string]interface{}{
				"baseline_trigger": map[string]interface{}{
					"trigger_type": "line",
					"operator":     ">=",
					"time_1":       "2021-08-17T11:45:00Z",
					"price_1":      "47160",
					"time_2":       "2021-08-18T10:00:00Z",
					"price_2":      "45560",
				},
				"baseline_offset_percent": 0.01,
			},
			feeds: []testFeed{
				{price: decimal.NewFromFloat(45595.56), time: time.Date(2021, 8, 18, 15, 46, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(45727.76), time: time.Date(2021, 8, 18, 15, 47, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered"}},
				{price: decimal.NewFromFloat(45270.49), time: time.Date(2021, 8, 18, 19, 17, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(45270.48), time: time.Date(2021, 8, 18, 19, 18, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(10000), time: time.Date(2021, 8, 19, 18, 24, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(100000), time: time.Date(2021, 8, 19, 18, 25, 0, 0, time.UTC), expectedHooks: nil},
			},
		},
		{
			title: "short - (breakout) with stop-loss and take-profit order",
			side:  order.SHORT,
			stopLossOrder: &order.StopLoss{
				BaselineReadjustmentEnabled: false,
				LossTolerancePercent:        0.01,
			},
			entryData: map[string]interface{}{
				"baseline_trigger": map[string]interface{}{
					"trigger_type": "line",
					"operator":     "<=",
					"time_1":       "2021-08-09T01:15:00Z",
					"price_1":      "42779",
					"time_2":       "2021-08-10T16:30:00Z",
					"price_2":      "44589.46",
				},
				"baseline_offset_percent": 0.01,
			},
			takeProfitOrder: &order.TakeProfit{Trigger: &trigger.Limit{
				Operator: "<=",
				Price:    decimal.NewFromFloat(44000),
			}},
			feeds: []testFeed{
				{price: decimal.NewFromFloat(45581), time: time.Date(2021, 8, 11, 23, 58, 0, 0, time.UTC), expectedHooks: nil},                                                  // 45580.49
				{price: decimal.NewFromFloat(45550), time: time.Date(2021, 8, 11, 23, 58, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}}, // 45580.49
				{price: decimal.NewFromFloat(46005), time: time.Date(2021, 8, 12, 1, 32, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(46052.25), time: time.Date(2021, 8, 12, 1, 32, 0, 0, time.UTC), expectedHooks: []string{"StopLossTriggered"}},
				{price: decimal.NewFromFloat(45781), time: time.Date(2021, 8, 12, 4, 21, 0, 0, time.UTC), expectedHooks: nil}, // 45780.66
				{price: decimal.NewFromFloat(45780), time: time.Date(2021, 8, 12, 4, 21, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}},
				{price: decimal.NewFromFloat(44001), time: time.Date(2021, 8, 12, 14, 56, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(44000), time: time.Date(2021, 8, 12, 14, 57, 0, 0, time.UTC), expectedHooks: []string{"TakeProfitTriggered"}},
			},
		},
		{
			title: "short - (breakout) without stop-loss and with take-profit order",
			side:  order.SHORT,
			entryData: map[string]interface{}{
				"baseline_trigger": map[string]interface{}{
					"trigger_type": "line",
					"operator":     "<=",
					"time_1":       "2021-08-09T01:15:00Z",
					"price_1":      "42779",
					"time_2":       "2021-08-10T16:30:00Z",
					"price_2":      "44589.46",
				},
				"baseline_offset_percent": 0.01,
			},
			takeProfitOrder: &order.TakeProfit{Trigger: &trigger.Limit{
				Operator: "<=",
				Price:    decimal.NewFromFloat(44000),
			}},
			feeds: []testFeed{
				{price: decimal.NewFromFloat(45581), time: time.Date(2021, 8, 11, 23, 58, 0, 0, time.UTC), expectedHooks: nil},                        // 45580.49
				{price: decimal.NewFromFloat(45550), time: time.Date(2021, 8, 11, 23, 58, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered"}}, // 45580.49
				{price: decimal.NewFromFloat(46005), time: time.Date(2021, 8, 12, 1, 32, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(46052.25), time: time.Date(2021, 8, 12, 1, 32, 0, 0, time.UTC), expectedHooks: nil}, // won't trigger stop-loss
				{price: decimal.NewFromFloat(45781), time: time.Date(2021, 8, 12, 4, 21, 0, 0, time.UTC), expectedHooks: nil},    // 45780.66
				{price: decimal.NewFromFloat(100000), time: time.Date(2021, 8, 12, 14, 56, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(44001), time: time.Date(2021, 8, 12, 14, 56, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(44000), time: time.Date(2021, 8, 12, 14, 57, 0, 0, time.UTC), expectedHooks: []string{"TakeProfitTriggered"}},
			},
		},
		{
			title: "short - (breakout) with stop-loss and without take-profit order",
			side:  order.SHORT,
			stopLossOrder: &order.StopLoss{
				BaselineReadjustmentEnabled: false,
				LossTolerancePercent:        0.01,
			},
			entryData: map[string]interface{}{
				"baseline_trigger": map[string]interface{}{
					"trigger_type": "line",
					"operator":     "<=",
					"time_1":       "2021-08-09T01:15:00Z",
					"price_1":      "42779",
					"time_2":       "2021-08-10T16:30:00Z",
					"price_2":      "44589.46",
				},
				"baseline_offset_percent": 0.01,
			},
			feeds: []testFeed{
				{price: decimal.NewFromFloat(45581), time: time.Date(2021, 8, 11, 23, 58, 0, 0, time.UTC), expectedHooks: nil},                                                  // 45580.49
				{price: decimal.NewFromFloat(45550), time: time.Date(2021, 8, 11, 23, 58, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}}, // 45580.49
				{price: decimal.NewFromFloat(46005), time: time.Date(2021, 8, 12, 1, 32, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(46052.25), time: time.Date(2021, 8, 12, 1, 32, 0, 0, time.UTC), expectedHooks: []string{"StopLossTriggered"}},
				{price: decimal.NewFromFloat(45781), time: time.Date(2021, 8, 12, 4, 21, 0, 0, time.UTC), expectedHooks: nil}, // 45780.66
				{price: decimal.NewFromFloat(45780), time: time.Date(2021, 8, 12, 4, 21, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}},
				{price: decimal.NewFromFloat(44001), time: time.Date(2021, 8, 12, 14, 56, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(44000), time: time.Date(2021, 8, 12, 14, 57, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(10000), time: time.Date(2021, 8, 12, 14, 57, 0, 0, time.UTC), expectedHooks: nil},
			},
		},
		{
			title: "short - (breakout) without stop-loss and take-profit order",
			side:  order.SHORT,
			entryData: map[string]interface{}{
				"baseline_trigger": map[string]interface{}{
					"trigger_type": "line",
					"operator":     "<=",
					"time_1":       "2021-08-09T01:15:00Z",
					"price_1":      "42779",
					"time_2":       "2021-08-10T16:30:00Z",
					"price_2":      "44589.46",
				},
				"baseline_offset_percent": 0.01,
			},
			feeds: []testFeed{
				{price: decimal.NewFromFloat(45581), time: time.Date(2021, 8, 11, 23, 58, 0, 0, time.UTC), expectedHooks: nil},                        // 45580.49
				{price: decimal.NewFromFloat(45550), time: time.Date(2021, 8, 11, 23, 58, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered"}}, // 45580.49
				{price: decimal.NewFromFloat(10000), time: time.Date(2021, 8, 12, 14, 57, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(100000), time: time.Date(2021, 8, 12, 14, 57, 0, 0, time.UTC), expectedHooks: nil},
			},
		},
	}

	for _, tc := range testcases {
		// Entry order has its process when it initialises
		entryOrder, err := order.NewEntry(tc.side, "baseline", tc.entryData)
		if err != nil {
			t.Error("TestBaselineAllOrders ", err)
			continue
		}
		c := &Contract{
			Side:            tc.side,
			EntryType:       "baseline",
			EntryOrder:      entryOrder,
			TakeProfitOrder: tc.takeProfitOrder,
			StopLossOrder:   tc.stopLossOrder,
		}
		h := &testHook{}
		c.SetHook(h)

		for i, feed := range tc.feeds {
			c.CheckPrice(Mark{Time: feed.time, Price: feed.price})
			if !reflect.DeepEqual(feed.expectedHooks, h.funcNames) {
				t.Errorf("TestBaselineAllOrders case '%s' (%d) - expect '%v', but got '%v'", tc.title, i, feed.expectedHooks, h.funcNames)
			}
			// Reset func names so that we can get fresh hooks each feed
			h.resetFuncNames()
		}
	}
}

// entry_type 'baseline'
func TestBaselineOffsetAndLossTolerancePercent(t *testing.T) {
	testcases := []struct {
		title           string
		side            order.Side
		takeProfitOrder order.Order
		entryData       map[string]interface{}
		stopLossOrder   order.Order
		feeds           []testFeed
	}{
		{
			title: "long - +0.01 / 0.01",
			side:  order.LONG,
			entryData: map[string]interface{}{
				"baseline_trigger": map[string]interface{}{
					"trigger_type": "line",
					"operator":     ">=",
					"time_1":       "2021-08-17T11:45:00Z",
					"price_1":      "47160",
					"time_2":       "2021-08-18T10:00:00Z",
					"price_2":      "45560",
				},
				"baseline_offset_percent": 0.01,
			},
			stopLossOrder: &order.StopLoss{
				LossTolerancePercent: 0.01,
			},
			feeds: []testFeed{
				// 1st
				{price: decimal.NewFromFloat(43854), time: time.Date(2021, 8, 19, 15, 45, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(43855), time: time.Date(2021, 8, 19, 15, 45, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}}, // entry: 43854.8808988764044272
				{price: decimal.NewFromFloat(43417), time: time.Date(2021, 8, 19, 15, 45, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(43416), time: time.Date(2021, 8, 19, 15, 45, 0, 0, time.UTC), expectedHooks: []string{"StopLossTriggered"}}, // stop-loss: 43416.45 (43855*0.99)
				// 2nd
				{price: decimal.NewFromFloat(43854), time: time.Date(2021, 8, 19, 15, 45, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(43855), time: time.Date(2021, 8, 19, 15, 45, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}}, // entry: 43854.8808988764044272
				{price: decimal.NewFromFloat(43417), time: time.Date(2021, 8, 19, 15, 45, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(43416), time: time.Date(2021, 8, 19, 15, 45, 0, 0, time.UTC), expectedHooks: []string{"StopLossTriggered"}}, // stop-loss: 43416.45 (43855*0.99)
			},
		},
		{
			title: "long - -0.01 / 0.02",
			side:  order.LONG,
			entryData: map[string]interface{}{
				"baseline_trigger": map[string]interface{}{
					"trigger_type": "line",
					"operator":     ">=",
					"time_1":       "2021-08-17T11:45:00Z",
					"price_1":      "47160",
					"time_2":       "2021-08-18T10:00:00Z",
					"price_2":      "45560",
				},
				"baseline_offset_percent": -0.01,
			},
			stopLossOrder: &order.StopLoss{
				LossTolerancePercent: 0.02,
			},
			feeds: []testFeed{
				// 1st
				{price: decimal.NewFromFloat(42986), time: time.Date(2021, 8, 19, 15, 45, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(42987), time: time.Date(2021, 8, 19, 15, 45, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}}, // entry: 42986.4674157303370128
				{price: decimal.NewFromFloat(42128), time: time.Date(2021, 8, 19, 15, 45, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(42127), time: time.Date(2021, 8, 19, 15, 45, 0, 0, time.UTC), expectedHooks: []string{"StopLossTriggered"}}, // stop-loss: 42127.26 (42987*0.98)
				// 2nd
				{price: decimal.NewFromFloat(42986), time: time.Date(2021, 8, 19, 15, 45, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(42987), time: time.Date(2021, 8, 19, 15, 45, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}}, // entry: 42986.4674157303370128
				{price: decimal.NewFromFloat(42128), time: time.Date(2021, 8, 19, 15, 45, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(42127), time: time.Date(2021, 8, 19, 15, 45, 0, 0, time.UTC), expectedHooks: []string{"StopLossTriggered"}}, // stop-loss: 42127.26 (42987*0.98)
			},
		},
		{
			title: "short - +0.01 / 0.01",
			side:  order.SHORT,
			entryData: map[string]interface{}{
				"baseline_trigger": map[string]interface{}{
					"trigger_type": "line",
					"operator":     "<=",
					"time_1":       "2021-08-17T11:45:00Z",
					"price_1":      "47160",
					"time_2":       "2021-08-18T10:00:00Z",
					"price_2":      "49560",
				},
				"baseline_offset_percent": 0.01,
			},
			stopLossOrder: &order.StopLoss{
				LossTolerancePercent: 0.01,
			},
			feeds: []testFeed{
				// 1st
				{price: decimal.NewFromFloat(52242), time: time.Date(2021, 8, 19, 15, 45, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(52241), time: time.Date(2021, 8, 19, 15, 45, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}}, // entry: 52241.2988764044944808
				{price: decimal.NewFromFloat(52763), time: time.Date(2021, 8, 19, 15, 45, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(52764), time: time.Date(2021, 8, 19, 15, 45, 0, 0, time.UTC), expectedHooks: []string{"StopLossTriggered"}}, // stop-loss: 52763.41 (52241*1.01)
				// 2nd
				{price: decimal.NewFromFloat(52242), time: time.Date(2021, 8, 19, 15, 45, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(52241), time: time.Date(2021, 8, 19, 15, 45, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}}, // entry: 52241.2988764044944808
				{price: decimal.NewFromFloat(52763), time: time.Date(2021, 8, 19, 15, 45, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(52764), time: time.Date(2021, 8, 19, 15, 45, 0, 0, time.UTC), expectedHooks: []string{"StopLossTriggered"}}, // stop-loss: 52763.41 (52241*1.01)
			},
		},
		{
			title: "short - -0.01 / 0.02",
			side:  order.SHORT,
			entryData: map[string]interface{}{
				"baseline_trigger": map[string]interface{}{
					"trigger_type": "line",
					"operator":     "<=",
					"time_1":       "2021-08-17T11:45:00Z",
					"price_1":      "47160",
					"time_2":       "2021-08-18T10:00:00Z",
					"price_2":      "49560",
				},
				"baseline_offset_percent": -0.01,
			},
			stopLossOrder: &order.StopLoss{
				LossTolerancePercent: 0.02,
			},
			feeds: []testFeed{
				// 1st
				{price: decimal.NewFromFloat(53297), time: time.Date(2021, 8, 19, 15, 45, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(53296), time: time.Date(2021, 8, 19, 15, 45, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}}, // entry: 53296.6786516853933592
				{price: decimal.NewFromFloat(54361), time: time.Date(2021, 8, 19, 15, 45, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(54362), time: time.Date(2021, 8, 19, 15, 45, 0, 0, time.UTC), expectedHooks: []string{"StopLossTriggered"}}, // stop-loss: 54361.92 (53296*1.02)
				// 2nd
				{price: decimal.NewFromFloat(53297), time: time.Date(2021, 8, 19, 15, 45, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(53296), time: time.Date(2021, 8, 19, 15, 45, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}}, // entry: 53296.6786516853933592
				{price: decimal.NewFromFloat(54361), time: time.Date(2021, 8, 19, 15, 45, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(54362), time: time.Date(2021, 8, 19, 15, 45, 0, 0, time.UTC), expectedHooks: []string{"StopLossTriggered"}}, // stop-loss: 54361.92 (53296*1.02)
			},
		},
	}

	for _, tc := range testcases {
		// Entry order has its process when it initialises
		entryOrder, err := order.NewEntry(tc.side, "baseline", tc.entryData)
		if err != nil {
			t.Error("TestBaselineOffsetAndLossTolerancePercent ", err)
			continue
		}
		c := &Contract{
			Side:          tc.side,
			EntryType:     order.ENTRY_BASELINE,
			EntryOrder:    entryOrder,
			StopLossOrder: tc.stopLossOrder,
		}
		h := &testHook{}
		c.SetHook(h)

		for i, feed := range tc.feeds {
			c.CheckPrice(Mark{Time: feed.time, Price: feed.price})
			if !reflect.DeepEqual(feed.expectedHooks, h.funcNames) {
				t.Errorf("TestBaselineOffsetAndLossTolerancePercent case '%s' (%d) - expect '%v', but got '%v'", tc.title, i, feed.expectedHooks, h.funcNames)
			}
			// Reset func names so that we can get fresh hooks each feed
			h.resetFuncNames()
		}
	}
}

func TestLimitFlipOperatorEnabled(t *testing.T) {
	testcases := []struct {
		title           string
		side            order.Side
		takeProfitOrder order.Order
		entryOrder      order.Order
		stopLossOrder   order.Order
		feeds           []testFeed
	}{
		{
			title: "long - (buy the dip) with stop-loss and take-profit order",
			side:  order.LONG,
			takeProfitOrder: &order.TakeProfit{Trigger: &trigger.Limit{
				Operator: ">=",
				Price:    decimal.NewFromFloat(48000),
			}},
			entryOrder: &order.Entry{
				Trigger: &trigger.Limit{
					Operator: "<=",
					Price:    decimal.NewFromFloat(47000),
				},
				FlipOperatorEnabled: true,
			},
			stopLossOrder: &order.StopLoss{Trigger: &trigger.Limit{
				Operator: "<=",
				Price:    decimal.NewFromFloat(46000),
			}},
			feeds: []testFeed{
				// time doesn't matter for 'limit'
				{price: decimal.NewFromFloat(47001), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(47000), time: time.Now(), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}},
				{price: decimal.NewFromFloat(46001), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(46000), time: time.Now(), expectedHooks: []string{"StopLossTriggered"}},
				{price: decimal.NewFromFloat(46001), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(46999), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(47000), time: time.Now(), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}},
				{price: decimal.NewFromFloat(46001), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(46000), time: time.Now(), expectedHooks: []string{"StopLossTriggered"}},
				{price: decimal.NewFromFloat(46999), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(47000), time: time.Now(), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}},
				{price: decimal.NewFromFloat(47999), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(48000), time: time.Now(), expectedHooks: []string{"TakeProfitTriggered"}},
			},
		},
		{
			title: "short - (buy the dip) with stop-loss and take-profit order",
			side:  order.SHORT,
			stopLossOrder: &order.StopLoss{Trigger: &trigger.Limit{
				Operator: ">=",
				Price:    decimal.NewFromFloat(48000),
			}},
			entryOrder: &order.Entry{
				Trigger: &trigger.Limit{
					Operator: ">=",
					Price:    decimal.NewFromFloat(47000),
				},
				FlipOperatorEnabled: true,
			},
			takeProfitOrder: &order.TakeProfit{Trigger: &trigger.Limit{
				Operator: "<=",
				Price:    decimal.NewFromFloat(46000),
			}},
			feeds: []testFeed{
				// time doesn't matter for 'limit'
				{price: decimal.NewFromFloat(46999), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(47000), time: time.Now(), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}},
				{price: decimal.NewFromFloat(47999), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(48000), time: time.Now(), expectedHooks: []string{"StopLossTriggered"}},
				{price: decimal.NewFromFloat(47999), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(47001), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(47000), time: time.Now(), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}},
				{price: decimal.NewFromFloat(47999), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(48000), time: time.Now(), expectedHooks: []string{"StopLossTriggered"}},
				{price: decimal.NewFromFloat(47001), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(47000), time: time.Now(), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}},
				{price: decimal.NewFromFloat(46001), time: time.Now(), expectedHooks: nil},
				{price: decimal.NewFromFloat(46000), time: time.Now(), expectedHooks: []string{"TakeProfitTriggered"}},
			},
		},
	}

	for _, tc := range testcases {
		c := &Contract{
			Side:            tc.side,
			EntryType:       order.ENTRY_LIMIT,
			EntryOrder:      tc.entryOrder,
			TakeProfitOrder: tc.takeProfitOrder,
			StopLossOrder:   tc.stopLossOrder,
		}
		h := &testHook{}
		c.SetHook(h)

		for i, feed := range tc.feeds {
			c.CheckPrice(Mark{Time: feed.time, Price: feed.price})
			if !reflect.DeepEqual(feed.expectedHooks, h.funcNames) {
				t.Errorf("TestLimitFlipOperatorEnabled case '%s' (%d) - expect '%v', but got '%v'", tc.title, i, feed.expectedHooks, h.funcNames)
			}
			// Reset func names so that we can get fresh hooks each feed
			h.resetFuncNames()
		}
	}
}

func TestBaselineFlipOperatorEnabled(t *testing.T) {
	testcases := []struct {
		title           string
		side            order.Side
		takeProfitOrder order.Order
		entryData       map[string]interface{}
		stopLossOrder   order.Order
		feeds           []testFeed
	}{
		{
			title: "long - (parallel channel, buy the dip) with stop-loss and take-profit order",
			side:  order.LONG,
			takeProfitOrder: &order.TakeProfit{Trigger: &trigger.Line{
				Operator: ">=",
				Time1:    time.Date(2021, 8, 20, 12, 45, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(48819.17),
				Time2:    time.Date(2021, 8, 21, 11, 0, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(50139.96),
			}},
			entryData: map[string]interface{}{
				"baseline_trigger": map[string]interface{}{
					"trigger_type": "line",
					"operator":     "<=",
					"time_1":       "2021-08-20T12:45:00Z",
					"price_1":      "46909.6",
					"time_2":       "2021-08-21T11:00:00Z",
					"price_2":      "48217.18",
				},
				"baseline_offset_percent": 0.01,
				"flip_operator_enabled":   true,
			},
			stopLossOrder: &order.StopLoss{
				BaselineReadjustmentEnabled: false,
				LossTolerancePercent:        0.01,
			},
			feeds: []testFeed{
				{price: decimal.NewFromFloat(47750), time: time.Date(2021, 8, 20, 19, 00, 0, 0, time.UTC), expectedHooks: nil},                                                  //
				{price: decimal.NewFromFloat(47749), time: time.Date(2021, 8, 20, 19, 00, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}}, // entry: <= 47749.66910112359552936
				{price: decimal.NewFromFloat(47272), time: time.Date(2021, 8, 20, 19, 00, 0, 0, time.UTC), expectedHooks: nil},                                                  //
				{price: decimal.NewFromFloat(47271), time: time.Date(2021, 8, 20, 19, 00, 0, 0, time.UTC), expectedHooks: []string{"StopLossTriggered"}},                        // stop-loss: 47271.51

				// even though the price comes back, it won't trigger entry order because operator has been flipped
				{price: decimal.NewFromFloat(47272), time: time.Date(2021, 8, 20, 19, 00, 0, 0, time.UTC), expectedHooks: nil},

				{price: decimal.NewFromFloat(47749), time: time.Date(2021, 8, 20, 19, 00, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(47750), time: time.Date(2021, 8, 20, 19, 00, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}}, // entry: >= 47749.66910112359552936
				{price: decimal.NewFromFloat(47749), time: time.Date(2021, 8, 20, 19, 00, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(47273), time: time.Date(2021, 8, 20, 19, 00, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(47272), time: time.Date(2021, 8, 20, 19, 00, 0, 0, time.UTC), expectedHooks: []string{"StopLossTriggered"}}, // stop-loss: 47272.5

				// even though the price comes back, it won't trigger entry order because operator has been flipped
				{price: decimal.NewFromFloat(47273), time: time.Date(2021, 8, 20, 19, 00, 0, 0, time.UTC), expectedHooks: nil},

				{price: decimal.NewFromFloat(47750), time: time.Date(2021, 8, 20, 19, 00, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}},
				{price: decimal.NewFromFloat(49190), time: time.Date(2021, 8, 20, 19, 00, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(49191), time: time.Date(2021, 8, 20, 19, 00, 0, 0, time.UTC), expectedHooks: []string{"TakeProfitTriggered"}}, // take-profit: 49190.178426966292158576
			},
		},
		{
			title: "long - (parallel channel, buy the dip) without stop-loss and take-profit order",
			side:  order.LONG,
			entryData: map[string]interface{}{
				"baseline_trigger": map[string]interface{}{
					"trigger_type": "line",
					"operator":     "<=",
					"time_1":       "2021-08-20T12:45:00Z",
					"price_1":      "46909.6",
					"time_2":       "2021-08-21T11:00:00Z",
					"price_2":      "48217.18",
				},
				"baseline_offset_percent": 0.01,
				"flip_operator_enabled":   true,
			},
			feeds: []testFeed{
				{price: decimal.NewFromFloat(47750), time: time.Date(2021, 8, 20, 19, 00, 0, 0, time.UTC), expectedHooks: nil},                        //
				{price: decimal.NewFromFloat(47749), time: time.Date(2021, 8, 20, 19, 00, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered"}}, // entry: <= 47749.66910112359552936
				{price: decimal.NewFromFloat(10000), time: time.Date(2021, 8, 20, 19, 00, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(100000), time: time.Date(2021, 8, 20, 19, 00, 0, 0, time.UTC), expectedHooks: nil},
			},
		},
		{
			title: "short - (parallel channel, buy the dip) with stop-loss and take-profit order",
			side:  order.SHORT,
			stopLossOrder: &order.StopLoss{
				BaselineReadjustmentEnabled: false,
				LossTolerancePercent:        0.01,
			},
			entryData: map[string]interface{}{
				"baseline_trigger": map[string]interface{}{
					"trigger_type": "line",
					"operator":     ">=",
					"time_1":       "2021-08-12T03:30:00Z",
					"price_1":      "46214",
					"time_2":       "2021-08-12T20:30:00Z",
					"price_2":      "44500",
				},
				"baseline_offset_percent": 0.01,
				"flip_operator_enabled":   true,
			},
			takeProfitOrder: &order.TakeProfit{Trigger: &trigger.Line{
				Operator: "<=",
				Time1:    time.Date(2021, 8, 12, 3, 30, 0, 0, time.UTC),
				Price1:   decimal.NewFromFloat(45041.05),
				Time2:    time.Date(2021, 8, 12, 20, 30, 0, 0, time.UTC),
				Price2:   decimal.NewFromFloat(43344.19),
			}},
			feeds: []testFeed{
				{price: decimal.NewFromFloat(45402), time: time.Date(2021, 8, 12, 7, 0, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(45403), time: time.Date(2021, 8, 12, 7, 0, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}}, // entry: 45402.50647058823524421
				{price: decimal.NewFromFloat(45402), time: time.Date(2021, 8, 12, 7, 0, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(45857), time: time.Date(2021, 8, 12, 7, 0, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(45858), time: time.Date(2021, 8, 12, 7, 0, 0, 0, time.UTC), expectedHooks: []string{"StopLossTriggered"}}, // stop-loss: 45857.03 (45403*1.01)

				// even though the price comes back, it won't trigger entry order because operator has been flipped
				{price: decimal.NewFromFloat(45857), time: time.Date(2021, 8, 12, 7, 0, 0, 0, time.UTC), expectedHooks: nil},

				{price: decimal.NewFromFloat(45403), time: time.Date(2021, 8, 12, 7, 0, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(45402), time: time.Date(2021, 8, 12, 7, 0, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}},
				{price: decimal.NewFromFloat(45403), time: time.Date(2021, 8, 12, 7, 0, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(45856), time: time.Date(2021, 8, 12, 7, 0, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(45857), time: time.Date(2021, 8, 12, 7, 0, 0, 0, time.UTC), expectedHooks: []string{"StopLossTriggered"}}, // stop-loss: 45856.02 (45402*1.01)

				// even though the price comes back, it won't trigger entry order because operator has been flipped
				{price: decimal.NewFromFloat(45856), time: time.Date(2021, 8, 12, 7, 0, 0, 0, time.UTC), expectedHooks: nil},

				{price: decimal.NewFromFloat(45403), time: time.Date(2021, 8, 12, 7, 0, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(45402), time: time.Date(2021, 8, 12, 7, 0, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}},
				{price: decimal.NewFromFloat(44692), time: time.Date(2021, 8, 12, 7, 0, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(44691), time: time.Date(2021, 8, 12, 7, 0, 0, 0, time.UTC), expectedHooks: []string{"TakeProfitTriggered"}}, // take-profit: 44691.69647058823524421
			},
		},
		{
			title: "short - (parallel channel, buy the dip) without stop-loss and take-profit order",
			side:  order.SHORT,
			entryData: map[string]interface{}{
				"baseline_trigger": map[string]interface{}{
					"trigger_type": "line",
					"operator":     ">=",
					"time_1":       "2021-08-12T03:30:00Z",
					"price_1":      "46214",
					"time_2":       "2021-08-12T20:30:00Z",
					"price_2":      "44500",
				},
				"baseline_offset_percent": 0.01,
				"flip_operator_enabled":   true,
			},
			feeds: []testFeed{
				{price: decimal.NewFromFloat(45402), time: time.Date(2021, 8, 12, 7, 0, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(45403), time: time.Date(2021, 8, 12, 7, 0, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered"}}, // entry: 45402.50647058823524421
				{price: decimal.NewFromFloat(10000), time: time.Date(2021, 8, 12, 7, 0, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(100000), time: time.Date(2021, 8, 12, 7, 0, 0, 0, time.UTC), expectedHooks: nil},
			},
		},
	}

	for _, tc := range testcases {
		// Entry order has its process when it initialises
		entryOrder, err := order.NewEntry(tc.side, "baseline", tc.entryData)
		if err != nil {
			t.Error("TestBaselineFlipOperatorEnabled ", err)
			continue
		}
		c := &Contract{
			Side:            tc.side,
			EntryType:       order.ENTRY_BASELINE,
			TakeProfitOrder: tc.takeProfitOrder,
			EntryOrder:      entryOrder,
			StopLossOrder:   tc.stopLossOrder,
		}
		h := &testHook{}
		c.SetHook(h)

		for i, feed := range tc.feeds {
			c.CheckPrice(Mark{Time: feed.time, Price: feed.price})
			if !reflect.DeepEqual(feed.expectedHooks, h.funcNames) {
				t.Errorf("TestBaselineFlipOperatorEnabled case '%s' (%d) - expect '%v', but got '%v'", tc.title, i, feed.expectedHooks, h.funcNames)
			}
			// Reset func names so that we can get fresh hooks each feed
			h.resetFuncNames()
		}
	}
}

// entry_type 'baseline'
func TestBaselineReadjustmentTrue(t *testing.T) {
	// Short
	/*
		{
		  "entry_type": "baseline",
		  "entry_order": {
			"baseline_trigger": {
			  "trigger_type": "line",
			  "operator": "<=",
			  "time_1": "2021-08-09 01:15:00",
			  "price_1": 42779,
			  "time_2": "2021-08-10 16:30:00",
			  "price_2": 44589.46
			},
			"baseline_offset_percent": 0.01
		  },
		  "stop_loss_order": {
			"loss_tolerance_percent": 0.01,
			"baseline_readjustment_enabled": true
		  },
		  "take_profit_order": {
			"trigger": {
			  "trigger_type": "limit",
			  "operator": "<=",
			  "price": 44000
			}
		  }
		}

		EntryTriggered             baseline: 46040.90  &{<= 2021-08-09 01:15:00 +0000 UTC 42779 2021-08-10 16:30:00 +0000 UTC 44589.46}
		EntryTriggered                entry: 45580.49  &{<= 2021-08-09 01:15:00 +0000 UTC 42351.21 2021-08-10 16:30:00 +0000 UTC 44143.5654}
		EntryTriggered                  buy: 45550  '2021-08-11 23:58'
		StopLossTriggerCreated    stop-loss: 46005.50  >=
		! StopLossTriggered            sell: 46052.25  '2021-08-12 01:32'  ($1000 => $1011)
		- EntryBaselineTriggerUpdated breakout: {2021-08-12 00:00:00 +0000 UTC 45444.64}
		- EntryBaselineTriggerUpdated baseline: &{<= 2021-08-09 01:15:00 +0000 UTC 42779 2021-08-12 00:00:00 +0000 UTC 45444.64}
		- EntryBaselineTriggerUpdated    entry: &{<= 2021-08-09 01:15:00 +0000 UTC 42351.21 2021-08-12 00:00:00 +0000 UTC 44990.1936}
		-----------------------
		EntryTriggered             baseline: 45648.72  &{<= 2021-08-09 01:15:00 +0000 UTC 42779 2021-08-12 00:00:00 +0000 UTC 45444.64}
		EntryTriggered                entry: 45192.24  &{<= 2021-08-09 01:15:00 +0000 UTC 42351.21 2021-08-12 00:00:00 +0000 UTC 44990.1936}
		EntryTriggered                  buy: 45162.76  '2021-08-12 05:25'
		StopLossTriggerCreated    stop-loss: 45614.39  >=
		! StopLossTriggered            sell: 45619.82  '2021-08-12 10:15'  ($1011 => $1021)
		- EntryBaselineTriggerUpdated breakout: {2021-08-12 05:36:00 +0000 UTC 44859.07}
		- EntryBaselineTriggerUpdated baseline: &{<= 2021-08-09 01:15:00 +0000 UTC 42779 2021-08-12 05:36:00 +0000 UTC 44859.07}
		- EntryBaselineTriggerUpdated    entry: &{<= 2021-08-09 01:15:00 +0000 UTC 42351.21 2021-08-12 05:36:00 +0000 UTC 44410.4793}
		-----------------------
		EntryTriggered             baseline: 45045.69  &{<= 2021-08-09 01:15:00 +0000 UTC 42779 2021-08-12 05:36:00 +0000 UTC 44859.07}
		EntryTriggered                entry: 44595.23  &{<= 2021-08-09 01:15:00 +0000 UTC 42351.21 2021-08-12 05:36:00 +0000 UTC 44410.4793}
		EntryTriggered                  buy: 44483.92  '2021-08-12 12:27'
		StopLossTriggerCreated    stop-loss: 44928.76  >=
		! TakeProfitTriggered          sell: 43957.73  '2021-08-12 14:57'  ($1021 => $1009)
		-----------------------
		'btcusdt future_contract' $1000 => $1009 (0.9%) (2021-08-01 ~ 2021-08-31) 428.581825ms
	*/
	testcases := []struct {
		title           string
		side            order.Side
		takeProfitOrder order.Order
		entryData       map[string]interface{}
		stopLossOrder   order.Order
		feeds           []testFeed
	}{
		{
			title: "long - baseline_readjustment_enabled 'true', p1 > p2, p1 < p2",
			side:  order.LONG,
			takeProfitOrder: &order.TakeProfit{Trigger: &trigger.Limit{
				Operator: ">=",
				Price:    decimal.NewFromFloat(50000),
			}},
			entryData: map[string]interface{}{
				"baseline_trigger": map[string]interface{}{
					"trigger_type": "line",
					"operator":     ">=",
					"time_1":       "2021-08-16T03:00:00Z",
					"price_1":      "48053.83",
					"time_2":       "2021-08-17T11:45:00Z",
					"price_2":      "47160",
				},
				"baseline_offset_percent": 0.01,
			},
			stopLossOrder: &order.StopLoss{
				BaselineReadjustmentEnabled: true,
				LossTolerancePercent:        0.01,
			},
			feeds: []testFeed{
				// p1 > p2
				{price: decimal.NewFromFloat(46163), time: time.Date(2021, 8, 19, 17, 0, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(46164), time: time.Date(2021, 8, 19, 17, 0, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}}, // 46163.74009236641225233208
				{price: decimal.NewFromFloat(45703), time: time.Date(2021, 8, 19, 17, 0, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(46877), time: time.Date(2021, 8, 19, 20, 0, 0, 0, time.UTC), expectedHooks: nil}, // set breakout peak
				{price: decimal.NewFromFloat(45703), time: time.Date(2021, 8, 19, 21, 0, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(45702), time: time.Date(2021, 8, 19, 21, 0, 0, 0, time.UTC), expectedHooks: []string{"StopLossTriggered", "EntryBaselineTriggerUpdated"}}, // stop-loss 45702.36, entry: &{<= 2021-08-16 03:00:00 +0000 UTC 48534.3683 2021-08-19 20:00:00 +0000 UTC 47345.77}
				{price: decimal.NewFromFloat(47158), time: time.Date(2021, 8, 20, 10, 0, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(47159), time: time.Date(2021, 8, 20, 10, 0, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}}, // entry: 47158.79948089887634973873

				// p1 < p2
				{price: decimal.NewFromFloat(49999), time: time.Date(2021, 8, 20, 20, 0, 0, 0, time.UTC), expectedHooks: nil}, // Set breakout peak
				{price: decimal.NewFromFloat(46688), time: time.Date(2021, 8, 20, 20, 0, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(46687), time: time.Date(2021, 8, 20, 20, 0, 0, 0, time.UTC), expectedHooks: []string{"StopLossTriggered", "EntryBaselineTriggerUpdated"}}, // stop-loss: 46687.41, entry: 48534.3683 p1 = p2
				{price: decimal.NewFromFloat(48534), time: time.Date(2021, 8, 20, 25, 0, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(48535), time: time.Date(2021, 8, 20, 25, 0, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}}, // entry: 48534.3683
				// re-trigger stop-loss and entry
				{price: decimal.NewFromFloat(49999), time: time.Date(2021, 8, 20, 25, 0, 0, 0, time.UTC), expectedHooks: nil}, // Set breakout peak
				{price: decimal.NewFromFloat(48050), time: time.Date(2021, 8, 20, 25, 0, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(48049), time: time.Date(2021, 8, 20, 25, 0, 0, 0, time.UTC), expectedHooks: []string{"StopLossTriggered", "EntryBaselineTriggerUpdated"}}, // stop-loss: 48049.65
				{price: decimal.NewFromFloat(48534), time: time.Date(2021, 8, 20, 25, 0, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(48535), time: time.Date(2021, 8, 20, 25, 0, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}}, // entry: 48534.3683

				{price: decimal.NewFromFloat(49999), time: time.Date(2021, 8, 20, 30, 0, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(50000), time: time.Date(2021, 8, 20, 30, 0, 0, 0, time.UTC), expectedHooks: []string{"TakeProfitTriggered"}},
			},
		},
		{
			title: "short - baseline_readjustment_enabled 'true', p1 < p2, p1 > p2",
			side:  order.SHORT,
			stopLossOrder: &order.StopLoss{
				BaselineReadjustmentEnabled: true,
				LossTolerancePercent:        0.01,
			},
			entryData: map[string]interface{}{
				"baseline_trigger": map[string]interface{}{
					"trigger_type": "line",
					"operator":     "<=",
					"time_1":       "2021-08-09T01:15:00Z",
					"price_1":      "42779",
					"time_2":       "2021-08-10T16:30:00Z",
					"price_2":      "44589.46",
				},
				"baseline_offset_percent": 0.01,
			},
			takeProfitOrder: &order.TakeProfit{Trigger: &trigger.Limit{
				Operator: "<=",
				Price:    decimal.NewFromFloat(40000),
			}},
			feeds: []testFeed{
				// p1 < p2
				{price: decimal.NewFromFloat(45580), time: time.Date(2021, 8, 11, 23, 57, 0, 0, time.UTC), expectedHooks: nil},                                                  // 45579.7329752866242551949
				{price: decimal.NewFromFloat(45550), time: time.Date(2021, 8, 11, 23, 58, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}}, // 45580.49406038216555410284
				{price: decimal.NewFromFloat(45444.64), time: time.Date(2021, 8, 12, 0, 0, 0, 0, time.UTC), expectedHooks: nil},                                                 // record breakout peak
				{price: decimal.NewFromFloat(46005), time: time.Date(2021, 8, 12, 1, 31, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(46006), time: time.Date(2021, 8, 12, 1, 32, 0, 0, time.UTC), expectedHooks: []string{"StopLossTriggered", "EntryBaselineTriggerUpdated"}},
				{price: decimal.NewFromFloat(45192), time: time.Date(2021, 8, 12, 5, 24, 0, 0, time.UTC), expectedHooks: nil},                                                     // 45191.61425639575967814936
				{price: decimal.NewFromFloat(45162.76), time: time.Date(2021, 8, 12, 5, 25, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}}, // 45192.23592508833933482604
				{price: decimal.NewFromFloat(44859.07), time: time.Date(2021, 8, 12, 5, 36, 0, 0, time.UTC), expectedHooks: nil},                                                  // set breakout peak
				{price: decimal.NewFromFloat(45614), time: time.Date(2021, 8, 12, 10, 14, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(45619.82), time: time.Date(2021, 8, 12, 10, 15, 0, 0, time.UTC), expectedHooks: []string{"StopLossTriggered", "EntryBaselineTriggerUpdated"}},
				{price: decimal.NewFromFloat(44595), time: time.Date(2021, 8, 12, 12, 26, 0, 0, time.UTC), expectedHooks: nil},                                                     // 44594.78412711198434700978
				{price: decimal.NewFromFloat(44483.92), time: time.Date(2021, 8, 12, 12, 27, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}}, // 44595.23

				// p1 > p2
				{price: decimal.NewFromFloat(41000), time: time.Date(2021, 8, 12, 14, 56, 0, 0, time.UTC), expectedHooks: nil}, // set breakout peak = 41000
				{price: decimal.NewFromFloat(44928), time: time.Date(2021, 8, 12, 14, 56, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(44929), time: time.Date(2021, 8, 12, 14, 56, 0, 0, time.UTC), expectedHooks: []string{"StopLossTriggered", "EntryBaselineTriggerUpdated"}}, // stop-loss: 44928.7592  , baseline: 42779 p1 = p2
				{price: decimal.NewFromFloat(42352), time: time.Date(2021, 8, 12, 14, 56, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(42351), time: time.Date(2021, 8, 12, 14, 56, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}}, // entry: 42351.21 (p1 42779*0.99) (p1 = p2)
				// re-trigger stop-loss and entry
				{price: decimal.NewFromFloat(41000), time: time.Date(2021, 8, 12, 14, 56, 0, 0, time.UTC), expectedHooks: nil}, // set breakout peak = 41000
				{price: decimal.NewFromFloat(42774), time: time.Date(2021, 8, 12, 14, 56, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(42775), time: time.Date(2021, 8, 12, 14, 56, 0, 0, time.UTC), expectedHooks: []string{"StopLossTriggered", "EntryBaselineTriggerUpdated"}}, // stop-loss: 42774.51 (42351*1.01)
				{price: decimal.NewFromFloat(42352), time: time.Date(2021, 8, 12, 14, 56, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(42351), time: time.Date(2021, 8, 12, 14, 56, 0, 0, time.UTC), expectedHooks: []string{"EntryTriggered", "StopLossTriggerCreated"}}, // entry: 42351.21 (42779*0.99) (p1 = p2)

				{price: decimal.NewFromFloat(40001), time: time.Date(2021, 8, 12, 14, 57, 0, 0, time.UTC), expectedHooks: nil},
				{price: decimal.NewFromFloat(40000), time: time.Date(2021, 8, 12, 14, 57, 0, 0, time.UTC), expectedHooks: []string{"TakeProfitTriggered"}},
			},
		},
	}

	for _, tc := range testcases {
		// Entry order has its process when it initialises
		entryOrder, err := order.NewEntry(tc.side, order.ENTRY_BASELINE, tc.entryData)
		if err != nil {
			t.Error("TestBaselineReadjustment ", err)
			continue
		}
		c := &Contract{
			Side:            tc.side,
			EntryType:       order.ENTRY_BASELINE,
			EntryOrder:      entryOrder,
			TakeProfitOrder: tc.takeProfitOrder,
			StopLossOrder:   tc.stopLossOrder,
		}
		h := &testHook{}
		c.SetHook(h)
		for i, feed := range tc.feeds {
			c.CheckPrice(Mark{Time: feed.time, Price: feed.price})
			if !reflect.DeepEqual(feed.expectedHooks, h.funcNames) {
				t.Errorf("TestBaselineReadjustment case '%s' (%d) - expect '%v', but got '%v'", tc.title, i, feed.expectedHooks, h.funcNames)
			}
			// Reset func names so that we can get fresh hooks each feed
			h.resetFuncNames()
		}
	}
}
