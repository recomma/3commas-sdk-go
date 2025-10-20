package threecommas

import (
	"encoding/json"
	"fmt"
	"time"
)

// ExampleDeal_Events demonstrates how to extract parsed bot events from a Deal.
func ExampleDeal_Events() {
	msg := func(s string) *string { return &s }

	now := time.Now()

	deal := Deal{
		Status:       DealStatusBought,
		ToCurrency:   "DOGE",
		FromCurrency: "USDT",
		BotEvents: []struct {
			CreatedAt *time.Time `json:"created_at,omitempty"`
			Message   *string    `json:"message,omitempty"`
		}{
			{CreatedAt: &now, Message: msg("Placing base order. Price: market Size: 25.0 USDT (100.0 DOGE)")},
			{CreatedAt: &now, Message: msg("Base order executed. Price: 0.25 USDT. Size: 25.0 USDT (100.0 DOGE)")},
			{CreatedAt: &now, Message: msg("Placing TakeProfit trade. Price: 0.27 USDT Size: 27.0 USDT (100.0 DOGE), the price should rise for 8% to close the trade")},
		},
	}

	for _, event := range deal.Events() {
		fmt.Printf("%s %s %s\n", event.Action, event.OrderType, event.Type)
	}

	// Output:
	// Placing Base BUY
	// Execute Base BUY
	// Placing Take Profit SELL
}

// ExampleBotEvent_Fingerprint shows how to get an ID from a, and it's respective, fingerprint from a BotEvent
func ExampleBotEvent_Fingerprint() {
	var deal Deal
	_ = json.Unmarshal([]byte(exampleDeal), &deal)

	for _, event := range deal.Events() {
		fmt.Printf("%d : %s\n", event.FingerprintAsID(), event.Fingerprint())
	}

	// Output:
	// 1917367905 : Base|0|0|DOGE|USDT
	// 1917367905 : Base|0|0|DOGE|USDT
	// 3940649977 : Take Profit|0|0|DOGE|USDT
	// 2737626907 : Safety|1|9|DOGE|USDT
	// 518435797 : Safety|2|9|DOGE|USDT
	// 3278924368 : Safety|3|9|DOGE|USDT
	// 3187895304 : Safety|4|9|DOGE|USDT
	// 1670755725 : Safety|5|9|DOGE|USDT
	// 3730823491 : Safety|6|9|DOGE|USDT
	// 63504582 : Safety|7|9|DOGE|USDT
	// 616158707 : Safety|8|9|DOGE|USDT
	// 4180610166 : Safety|9|9|DOGE|USDT
	// 2737626907 : Safety|1|9|DOGE|USDT
	// 3940649977 : Take Profit|0|0|DOGE|USDT
	// 3940649977 : Take Profit|0|0|DOGE|USDT
	// 3940649977 : Take Profit|0|0|DOGE|USDT
	// 518435797 : Safety|2|9|DOGE|USDT
	// 3940649977 : Take Profit|0|0|DOGE|USDT
	// 3940649977 : Take Profit|0|0|DOGE|USDT
	// 3940649977 : Take Profit|0|0|DOGE|USDT
	// 3278924368 : Safety|3|9|DOGE|USDT
	// 3940649977 : Take Profit|0|0|DOGE|USDT
	// 3940649977 : Take Profit|0|0|DOGE|USDT
	// 3940649977 : Take Profit|0|0|DOGE|USDT
	// 3187895304 : Safety|4|9|DOGE|USDT
	// 3940649977 : Take Profit|0|0|DOGE|USDT
	// 3940649977 : Take Profit|0|0|DOGE|USDT
	// 3940649977 : Take Profit|0|0|DOGE|USDT
	// 1670755725 : Safety|5|9|DOGE|USDT
	// 3940649977 : Take Profit|0|0|DOGE|USDT
	// 3940649977 : Take Profit|0|0|DOGE|USDT
	// 3940649977 : Take Profit|0|0|DOGE|USDT
	// 3730823491 : Safety|6|9|DOGE|USDT
	// 3940649977 : Take Profit|0|0|DOGE|USDT
	// 3940649977 : Take Profit|0|0|DOGE|USDT
	// 3940649977 : Take Profit|0|0|DOGE|USDT
	// 63504582 : Safety|7|9|DOGE|USDT
	// 3940649977 : Take Profit|0|0|DOGE|USDT
	// 3940649977 : Take Profit|0|0|DOGE|USDT
	// 3940649977 : Take Profit|0|0|DOGE|USDT
	// 616158707 : Safety|8|9|DOGE|USDT
	// 3940649977 : Take Profit|0|0|DOGE|USDT
	// 3940649977 : Take Profit|0|0|DOGE|USDT
	// 3940649977 : Take Profit|0|0|DOGE|USDT
	// 4180610166 : Safety|9|9|DOGE|USDT
	// 3940649977 : Take Profit|0|0|DOGE|USDT
	// 3940649977 : Take Profit|0|0|DOGE|USDT
	// 3940649977 : Take Profit|0|0|DOGE|USDT
	// 3940649977 : Take Profit|0|0|DOGE|USDT
	// 149257158 : |0|0|DOGE|USDT
}

const exampleDeal = `
{
	"status": "completed",
	"from_currency": "USDT",
    "to_currency": "DOGE",
	"bot_events": [
        {
            "message": "Averaging order (7 out of 9) executed. Price: 0.22446424 USDT Size: 24.01767368 USDT (107.0 DOGE)",
            "created_at": "2025-09-25T20:34:48.398Z"
        },
        {
            "message": "Cancelling TakeProfit trade. Price: 0.2311 USDT Size: 171.4762 USDT (742.0 DOGE)",
            "created_at": "2025-09-25T20:34:48.523Z"
        },
        {
            "message": "TakeProfit trade cancelled. Price: 0.2311 USDT Size: 171.4762 USDT (742.0 DOGE)",
            "created_at": "2025-09-25T20:34:48.537Z"
        },
        {
            "message": "Placing TakeProfit trade.  Price: 0.23086 USDT Size: 196.00014 USDT (849.0 DOGE), the price should rise for 2.96% to close the trade",
            "created_at": "2025-09-25T20:34:48.600Z"
        },
        {
            "message": "Averaging order (8 out of 9) executed. Price: 0.22398376 USDT Size: 23.96626232 USDT (107.0 DOGE)",
            "created_at": "2025-09-25T21:32:52.632Z"
        },
        {
            "message": "Cancelling TakeProfit trade. Price: 0.23086 USDT Size: 196.00014 USDT (849.0 DOGE)",
            "created_at": "2025-09-25T21:32:52.787Z"
        },
        {
            "message": "TakeProfit trade cancelled. Price: 0.23086 USDT Size: 196.00014 USDT (849.0 DOGE)",
            "created_at": "2025-09-25T21:32:52.803Z"
        },
        {
            "message": "Placing TakeProfit trade.  Price: 0.23062 USDT Size: 220.47272 USDT (956.0 DOGE), the price should rise for 3.18% to close the trade",
            "created_at": "2025-09-25T21:32:52.897Z"
        },
        {
            "message":"Averaging order (9 out of 9) executed. Price: 0.22350328 USDT Size: 23.91485096 USDT (107.0 DOGE) #lastAO",
            "created_at": "2025-09-25T21:33:35.793Z"
        },
        {
            "message": "Cancelling TakeProfit trade. Price: 0.23062 USDT Size: 220.47272 USDT (956.0 DOGE)",
            "created_at": "2025-09-25T21:33:35.976Z"
        },
        {
            "message": "TakeProfit trade cancelled. Price: 0.23062 USDT Size: 220.47272 USDT (956.0 DOGE)",
            "created_at": "2025-09-25T21:33:35.991Z"
        },
        {
            "message": "Placing TakeProfit trade.  Price: 0.23038 USDT Size: 244.89394 USDT (1063.0 DOGE), the price should rise for 3.16% to close the trade",
            "created_at": "2025-09-25T21:33:36.073Z"
        },
        {
            "message": "TakeProfit trade finished. Price: 0.23014962 USDT Size: 244.64904606 USDT (1063.0 DOGE)",
            "created_at": "2025-09-26T17:24:07.729Z"
        },
        {
            "message":"(USDT_DOGE): Trade completed. Profit:  +4.80727389 USDT (4.81 $) (2.0% from total volume)). #profit about 23 hours",
            "created_at": "2025-09-26T17:24:08.441Z"
        },
        {
            "message": "Placing base order. Price: market Size: 23.86965 USDT (Risk reduction 1.1366 USDT) (105.0 DOGE)",
            "created_at": "2025-09-25T18:53:52.312Z"
        },
        {
            "message": "Base order executed.  Price: 0.22758736 USDT.  Size: 23.8966728 USDT (105.0 DOGE)",
            "created_at": "2025-09-25T18:53:52.960Z"
        },
        {
            "message": "Placing TakeProfit trade.  Price: 0.23238 USDT Size: 24.3999 USDT (105.0 DOGE), the price should rise for 2.23% to close the trade",
            "created_at": "2025-09-25T18:53:53.167Z"
        },
        {
            "message": "Placing averaging order (1 out of 9). Price: 0.2271 USDT Size: 24.0726 USDT (Risk reduction 1.1366 USDT) (106.0 DOGE)",
            "created_at": "2025-09-25T18:53:53.530Z"
        },
        {
            "message": "Placing averaging order (2 out of 9). Price: 0.22663 USDT Size: 24.02278 USDT (Risk reduction 1.1366 USDT) (106.0 DOGE)",
            "created_at": "2025-09-25T18:53:53.691Z"
        },
        {
            "message": "Placing averaging order (3 out of 9). Price: 0.22615 USDT Size: 23.9719 USDT (Risk reduction 1.1366 USDT) (106.0 DOGE)",
            "created_at": "2025-09-25T18:53:53.810Z"
        },
        {
            "message": "Placing averaging order (4 out of 9). Price: 0.22567 USDT Size: 23.92102 USDT (Risk reduction 1.1366 USDT) (106.0 DOGE)",
            "created_at": "2025-09-25T18:53:53.896Z"
        },
        {
            "message": "Placing averaging order (5 out of 9). Price: 0.22519 USDT Size: 23.87014 USDT (Risk reduction 1.1366 USDT) (106.0 DOGE)",
            "created_at": "2025-09-25T18:53:53.963Z"
        },
        {
            "message": "Placing averaging order (6 out of 9). Price: 0.22471 USDT Size: 24.04397 USDT (Risk reduction 1.1366 USDT) (107.0 DOGE)",
            "created_at": "2025-09-25T18:53:54.018Z"
        },
        {
            "message": "Placing averaging order (7 out of 9). Price: 0.22424 USDT Size: 23.99368 USDT (Risk reduction 1.1366 USDT) (107.0 DOGE)",
            "created_at": "2025-09-25T18:53:54.071Z"
        },
        {
            "message": "Placing averaging order (8 out of 9). Price: 0.22376 USDT Size: 23.94232 USDT (Risk reduction 1.1366 USDT) (107.0 DOGE)",
            "created_at": "2025-09-25T18:53:54.149Z"
        },
        {
            "message": "Placing averaging order (9 out of 9). Price: 0.22328 USDT Size: 23.89096 USDT (Risk reduction 1.1366 USDT) (107.0 DOGE)",
            "created_at": "2025-09-25T18:53:54.215Z"
        },
        {
            "message": "Averaging order (1 out of 9) executed. Price: 0.2273271 USDT Size: 24.0966726 USDT (106.0 DOGE)",
            "created_at": "2025-09-25T19:45:54.263Z"
        },
        {
            "message": "Cancelling TakeProfit trade. Price: 0.23238 USDT Size: 24.3999 USDT (105.0 DOGE)",
            "created_at": "2025-09-25T19:45:54.389Z"
        },
        {
            "message": "TakeProfit trade cancelled. Price: 0.23238 USDT Size: 24.3999 USDT (105.0 DOGE)",
            "created_at": "2025-09-25T19:45:54.402Z"
        },
        {
            "message": "Placing TakeProfit trade.  Price: 0.23224 USDT Size: 49.00264 USDT (211.0 DOGE), the price should rise for 2.28% to close the trade",
            "created_at": "2025-09-25T19:45:54.483Z"
        },
        {
            "message": "Averaging order (2 out of 9) executed. Price: 0.22685663 USDT Size: 24.04680278 USDT (106.0 DOGE)",
            "created_at": "2025-09-25T19:50:39.769Z"
        },
        {
            "message": "Cancelling TakeProfit trade. Price: 0.23224 USDT Size: 49.00264 USDT (211.0 DOGE)",
            "created_at": "2025-09-25T19:50:39.937Z"
        },
        {
            "message": "TakeProfit trade cancelled. Price: 0.23224 USDT Size: 49.00264 USDT (211.0 DOGE)",
            "created_at": "2025-09-25T19:50:39.952Z"
        },
        {
            "message": "Placing TakeProfit trade.  Price: 0.23204 USDT Size: 73.55668 USDT (317.0 DOGE), the price should rise for 2.36% to close the trade",
            "created_at": "2025-09-25T19:50:40.022Z"
        },
        {
            "message": "Averaging order (3 out of 9) executed. Price: 0.22637615 USDT Size: 23.9958719 USDT (106.0 DOGE)",
            "created_at": "2025-09-25T19:55:20.560Z"
        },
        {
            "message": "Cancelling TakeProfit trade. Price: 0.23204 USDT Size: 73.55668 USDT (317.0 DOGE)",
            "created_at": "2025-09-25T19:55:20.690Z"
        },
        {
            "message": "TakeProfit trade cancelled. Price: 0.23204 USDT Size: 73.55668 USDT (317.0 DOGE)",
            "created_at": "2025-09-25T19:55:20.704Z"
        },
        {
            "message": "Placing TakeProfit trade.  Price: 0.23181 USDT Size: 98.05563 USDT (423.0 DOGE), the price should rise for 2.48% to close the trade",
            "created_at": "2025-09-25T19:55:20.769Z"
        },
        {
            "message": "Averaging order (4 out of 9) executed. Price: 0.22589567 USDT Size: 23.94494102 USDT (106.0 DOGE)",
            "created_at": "2025-09-25T20:12:00.956Z"
        },
        {
            "message": "Cancelling TakeProfit trade. Price: 0.23181 USDT Size: 98.05563 USDT (423.0 DOGE)",
            "created_at": "2025-09-25T20:12:01.101Z"
        },
        {
            "message": "TakeProfit trade cancelled. Price: 0.23181 USDT Size: 98.05563 USDT (423.0 DOGE)",
            "created_at": "2025-09-25T20:12:01.114Z"
        },
        {
            "message": "Placing TakeProfit trade.  Price: 0.23158 USDT Size: 122.50582 USDT (529.0 DOGE), the price should rise for 2.61% to close the trade",
            "created_at": "2025-09-25T20:12:01.176Z"
        },
        {
            "message": "Averaging order (5 out of 9) executed. Price: 0.22541519 USDT Size: 23.89401014 USDT (106.0 DOGE)",
            "created_at": "2025-09-25T20:13:22.654Z"
        },
        {
            "message": "Cancelling TakeProfit trade. Price: 0.23158 USDT Size: 122.50582 USDT (529.0 DOGE)",
            "created_at": "2025-09-25T20:13:22.864Z"
        },
        {
            "message": "TakeProfit trade cancelled. Price: 0.23158 USDT Size: 122.50582 USDT (529.0 DOGE)",
            "created_at": "2025-09-25T20:13:22.888Z"
        },
        {
            "message": "Placing TakeProfit trade.  Price: 0.23134 USDT Size: 146.9009 USDT (635.0 DOGE), the price should rise for 2.73% to close the trade",
            "created_at": "2025-09-25T20:13:23.000Z"
        },
        {
            "message": "Averaging order (6 out of 9) executed. Price: 0.22493471 USDT Size: 24.06801397 USDT (107.0 DOGE)",
            "created_at": "2025-09-25T20:32:14.588Z"
        },
        {
            "message": "Cancelling TakeProfit trade. Price: 0.23134 USDT Size: 146.9009 USDT (635.0 DOGE)",
            "created_at": "2025-09-25T20:32:14.871Z"
        },
        {
            "message": "TakeProfit trade cancelled. Price: 0.23134 USDT Size: 146.9009 USDT (635.0 DOGE)",
            "created_at": "2025-09-25T20:32:14.885Z"
        },
        {
            "message": "Placing TakeProfit trade.  Price: 0.2311 USDT Size: 171.4762 USDT (742.0 DOGE), the price should rise for 2.79% to close the trade",
            "created_at": "2025-09-25T20:32:14.995Z"
        }
    ]
}`
