package threecommas

import (
	"fmt"
	"hash/crc32"
	"sort"
	"strings"
	"time"

	"github.com/terwey/3commas-sdk-go/threecommas/eventparser"
)

type BotEventAction string

const (
	BotEventActionPlace     BotEventAction = "Placing"
	BotEventActionExecute   BotEventAction = "Execute"
	BotEventActionCancel    BotEventAction = "Cancel"
	BotEventActionCancelled BotEventAction = "Cancelled"
	BotEventActionModify    BotEventAction = "Modify"
)

type BotEvent struct {
	CreatedAt time.Time

	Action BotEventAction

	Coin string // DOGE

	Type MarketOrderOrderType // BUY

	Status MarketOrderStatusString // Active

	// Price is the cost set for the order
	Price float64 // 25.0654404

	// Size is the size of COIN to order
	Size float64 // 110.0

	// MarketOrderDealOrderTypeSafety Safety
	OrderType MarketOrderDealOrderType

	// OrderSize of the group of orders send, e.g. 9 orders
	OrderSize int // 9
	// OrderPosition of the group of orders, e.g. 8 out of 9
	OrderPosition int // 8

	QuoteVolume      float64
	QuoteCurrency    string
	IsMarket         bool
	Profit           float64
	ProfitCurrency   string
	ProfitUSD        float64
	ProfitPercentage float64

	// Example: Averaging order (8 out of 9) executed. Price: market Size: 25.0654404 USDT (110.0 DOGE)
	Text string
}

// Fingerprint can be used to identify the same BotEvent across different states
func (event *BotEvent) Fingerprint() string {
	return fmt.Sprintf(
		"%s|%d|%d|%s|%s",
		event.OrderType,
		event.OrderPosition,
		event.OrderSize,
		strings.ToUpper(event.Coin),
		strings.ToUpper(event.QuoteCurrency),
	)
}

// FingerprintAsID is an uint32 that can be used to identify the same BotEvent across different states
// Could be seen as a replacement for a MarketOrder ID, however they share no relation
func (event *BotEvent) FingerprintAsID() uint32 {
	return crc32.ChecksumIEEE([]byte(event.Fingerprint()))
}

// Events returns the parsed BotEvents sorted on CreatedAt
func (d *Deal) Events() []BotEvent {
	ctx := eventparser.Context{
		Strategy:      DealStrategy(d),
		BaseCurrency:  strings.ToUpper(d.ToCurrency),
		QuoteCurrency: strings.ToUpper(d.FromCurrency),
	}

	events := make([]BotEvent, 0, len(d.BotEvents))

	for _, raw := range d.BotEvents {
		if raw.Message == nil {
			continue
		}

		parsed, err := eventparser.Parse(*raw.Message, ctx)
		if err != nil {
			continue
		}

		events = append(events, BotEvent{
			CreatedAt:        *raw.CreatedAt,
			Action:           BotEventAction(parsed.Action),
			Coin:             parsed.Coin,
			Type:             MarketOrderOrderType(parsed.Side),
			Status:           MarketOrderStatusString(parsed.Status),
			Price:            parsed.Price,
			Size:             parsed.Size,
			OrderType:        mapOrderType(parsed.OrderType),
			OrderSize:        parsed.OrderSize,
			OrderPosition:    parsed.OrderPosition,
			QuoteVolume:      parsed.QuoteVolume,
			QuoteCurrency:    parsed.QuoteCurrency,
			IsMarket:         parsed.IsMarket,
			Profit:           parsed.Profit,
			ProfitCurrency:   parsed.ProfitCurrency,
			ProfitUSD:        parsed.ProfitUSD,
			ProfitPercentage: parsed.ProfitPercentage,
			Text:             parsed.Text,
		})
	}

	sort.Slice(events, func(i, j int) bool {
		return events[i].CreatedAt.Before(events[j].CreatedAt)
	})

	return events
}

func mapOrderType(t eventparser.OrderType) MarketOrderDealOrderType {
	switch t {
	case eventparser.OrderTypeBase:
		return MarketOrderDealOrderTypeBase
	case eventparser.OrderTypeSafety:
		return MarketOrderDealOrderTypeSafety
	case eventparser.OrderTypeManualSafety:
		return MarketOrderDealOrderTypeManualSafety
	case eventparser.OrderTypeTakeProfit:
		return MarketOrderDealOrderTypeTakeProfit
	case eventparser.OrderTypeStopLoss:
		return MarketOrderDealOrderTypeStopLoss
	default:
		return ""
	}
}

func DealStrategy(d *Deal) eventparser.Strategy {
	if d == nil {
		return eventparser.StrategyUnknown
	}

	switch strings.ToLower(string(d.Status)) {
	case "bought", "buying", "active", "completed":
		return eventparser.StrategyLong
	case "sold", "selling":
		return eventparser.StrategyShort
	case "failed":
		return eventparser.StrategyUnknown
	default:
		return eventparser.StrategyUnknown
	}
}
