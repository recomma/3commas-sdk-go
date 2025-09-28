package eventparser

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/google/go-cmp/cmp/cmpopts"
)

func TestParse(t *testing.T) {
	ctx := Context{
		Strategy:      StrategyLong,
		BaseCurrency:  "DOGE",
		QuoteCurrency: "USDT",
	}

	tests := []struct {
		name    string
		message string
		want    Event
	}{
		{
			name:    "placing_averaging_9_9",
			message: "Placing averaging order (9 out of 9). Price: market Size: 25.0008 USDT (110.0 DOGE)",
			want: Event{
				Action:        ActionPlace,
				OrderType:     OrderTypeSafety,
				Side:          SideBuy,
				Status:        StatusActive,
				OrderPosition: 9,
				OrderSize:     9,
				Coin:          "DOGE",
				QuoteCurrency: "USDT",
				QuoteVolume:   25.0008,
				Price:         0,
				IsMarket:      true,
				Size:          110.0,
			},
		},
		{
			name:    "executed_averaging_9_9",
			message: "Averaging order (9 out of 9) executed. Price: market Size: 25.0269019 USDT (110.0 DOGE) #lastAO 😬",
			want: Event{
				Action:        ActionExecute,
				OrderType:     OrderTypeSafety,
				Side:          SideBuy,
				Status:        StatusFilled,
				OrderPosition: 9,
				OrderSize:     9,
				Coin:          "DOGE",
				QuoteCurrency: "USDT",
				QuoteVolume:   25.0269019,
				Price:         0,
				IsMarket:      true,
				Size:          110.0,
			},
		},
		{
			name:    "cancelling_tp",
			message: "Cancelling TakeProfit trade. Price: 0.23469 USDT Size: 230.93496 USDT (984.0 DOGE)",
			want: Event{
				Action:        ActionCancel,
				OrderType:     OrderTypeTakeProfit,
				Side:          SideSell,
				Status:        StatusCancelled,
				Coin:          "DOGE",
				QuoteCurrency: "USDT",
				QuoteVolume:   230.93496,
				Price:         0.23469,
				IsMarket:      false,
				Size:          984.0,
			},
		},
		{
			name:    "cancelling_buy_order",
			message: "Cancelling buy order (3 out of 9). Price: 0.22815 USDT Size: 25.0965 USDT (110.0 DOGE)",
			want: Event{
				Action:        ActionCancel,
				OrderType:     OrderTypeSafety,
				Side:          SideBuy,
				Status:        StatusCancelled,
				OrderPosition: 3,
				OrderSize:     9,
				Coin:          "DOGE",
				QuoteCurrency: "USDT",
				QuoteVolume:   25.0965,
				Price:         0.22815,
				IsMarket:      false,
				Size:          110.0,
			},
		},
		{
			name:    "cancelled_tp",
			message: "TakeProfit trade cancelled. Price: 0.23469 USDT Size: 230.93496 USDT (984.0 DOGE)",
			want: Event{
				Action:        ActionCancelled,
				OrderType:     OrderTypeTakeProfit,
				Side:          SideSell,
				Status:        StatusCancelled,
				Coin:          "DOGE",
				QuoteCurrency: "USDT",
				QuoteVolume:   230.93496,
				Price:         0.23469,
				IsMarket:      false,
				Size:          984.0,
			},
		},
		{
			name:    "placing_tp",
			message: "Placing TakeProfit trade.  Price: 0.23445 USDT Size: 256.4883 USDT (1094.0 DOGE), the price should rise for 3.16% to close the trade",
			want: Event{
				Action:        ActionPlace,
				OrderType:     OrderTypeTakeProfit,
				Side:          SideSell,
				Status:        StatusActive,
				Coin:          "DOGE",
				QuoteCurrency: "USDT",
				QuoteVolume:   256.4883,
				Price:         0.23445,
				IsMarket:      false,
				Size:          1094.0,
			},
		},
		{
			name:    "placing_base_order_risk_reduction",
			message: "Placing base order. Price: market Size: 39.38256 USDT (Risk reduction 5.62584 USDT) (168.0 DOGE)",
			want: Event{
				Action:        ActionPlace,
				OrderType:     OrderTypeBase,
				Side:          SideBuy,
				Status:        StatusActive,
				Coin:          "DOGE",
				QuoteCurrency: "USDT",
				QuoteVolume:   39.38256,
				Price:         0,
				IsMarket:      true,
				Size:          168.0,
			},
		},
		{
			name:    "placing_stoploss_trade",
			message: "Placing StopLoss trade. Price: market Size: 378.81169326 USDT (1698.0 DOGE)",
			want: Event{
				Action:        ActionPlace,
				OrderType:     OrderTypeStopLoss,
				Side:          SideSell,
				Status:        StatusActive,
				Coin:          "DOGE",
				QuoteCurrency: "USDT",
				QuoteVolume:   378.81169326,
				Price:         0,
				IsMarket:      true,
				Size:          1698.0,
			},
		},
		{
			name:    "stoploss_summary",
			message: "Stop loss 📛  -17.51435838 USDT (-17.51 $) (-4.43% from total volume) #stoploss",
			want: Event{
				Action:           ActionCancelled,
				OrderType:        OrderTypeStopLoss,
				Side:             SideSell,
				Status:           StatusCancelled,
				Coin:             "DOGE",
				QuoteCurrency:    "USDT",
				Profit:           -17.51435838,
				ProfitCurrency:   "USDT",
				ProfitUSD:        -17.51,
				ProfitPercentage: -4.43,
			},
		},
		{
			name:    "takeprofit_finished",
			message: "TakeProfit trade finished. Price: 0.23072904 USDT Size: 230.95976904 USDT (1001.0 DOGE)",
			want: Event{
				Action:        ActionFinished,
				OrderType:     OrderTypeTakeProfit,
				Side:          SideSell,
				Status:        StatusFinished,
				Coin:          "DOGE",
				QuoteCurrency: "USDT",
				QuoteVolume:   230.95976904,
				Price:         0.23072904,
				Size:          1001.0,
			},
		},
		{
			name:    "trade_completed_summary",
			message: "(USDT_DOGE): Trade completed. Profit:  +4.53711258 USDT (4.54 $) (2.0% from total volume) 💰💰💰). #profit about 5 hours",
			want: Event{
				Action:           ActionCompleted,
				OrderType:        OrderTypeSummary,
				Side:             SideUnknown,
				Status:           StatusFinished,
				Coin:             "DOGE",
				QuoteCurrency:    "USDT",
				Profit:           4.53711258,
				ProfitCurrency:   "USDT",
				ProfitUSD:        4.54,
				ProfitPercentage: 2.0,
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.message, ctx)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}

			// Always expect original text.
			if got.Text != strings.TrimSpace(tt.message) {
				t.Fatalf("Parse() Text mismatch: got %q", got.Text)
			}

			// Zero out Text for comparison.
			got.Text = ""
			if tt.want.Text != "" {
				tt.want.Text = ""
			}

			got.Text = strings.TrimSpace(tt.message)

			diff := cmp.Diff(
				tt.want,
				got,
				cmpopts.IgnoreFields(Event{}, "Text"),
				cmpopts.EquateApprox(0, 1e-6),
			)
			if diff != "" {
				t.Fatalf("Parse() mismatch (-want +got):\n%s", diff)
			}
		})
	}
}
