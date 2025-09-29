package eventparser

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

var (
	progressRe       = regexp.MustCompile(`\((\d+)\s+out of\s+(\d+)\)`)
	priceRe          = regexp.MustCompile(`Price:\s*(market|[\d.]+)(?:\s+([A-Za-z]{2,}))?`)
	sizeRe           = regexp.MustCompile(`Size:\s*([\d.]+)\s*([A-Za-z]{2,})`)
	baseSizeRe       = regexp.MustCompile(`\((?:[A-Za-z]+\s+)?([\d.]+)\s*([A-Za-z]{2,})\)`)
	profitRe         = regexp.MustCompile(`Profit:\s*([+-]?\d+(?:\.\d+)?)\s*([A-Za-z]{2,})`)
	profitUSDRe      = regexp.MustCompile(`\(([+-]?\d+(?:\.\d+)?)\s*\$\)`)
	profitPctRe      = regexp.MustCompile(`\(([+-]?\d+(?:\.\d+)?)%\s*`) // matches “(2.0% …”
	amountCurrencyRe = regexp.MustCompile(`([-+]?\d+(?:\.\d+)?)\s*([A-Za-z]{2,})`)
)

// Strategy enumerates the deal direction to decide BUY/SELL.
type Strategy string

const (
	StrategyUnknown Strategy = ""
	StrategyLong    Strategy = "long"
	StrategyShort   Strategy = "short"
)

// Action captures the verb in the event.
type Action string

const (
	ActionUnknown   Action = ""
	ActionPlace     Action = "Placing"
	ActionExecute   Action = "Execute"
	ActionCancel    Action = "Cancel"
	ActionCancelled Action = "Cancelled"
	ActionFinished  Action = "Finished"
	ActionCompleted Action = "Completed"
)

// OrderType mirrors the market order deal categories.
type OrderType string

const (
	OrderTypeUnknown      OrderType = ""
	OrderTypeBase         OrderType = "Base"
	OrderTypeSafety       OrderType = "Safety"
	OrderTypeManualSafety OrderType = "Manual Safety"
	OrderTypeTakeProfit   OrderType = "Take Profit"
	OrderTypeStopLoss     OrderType = "Stop Loss"
	OrderTypeSummary      OrderType = "Summary"
)

// Side indicates BUY/SELL.
type Side string

const (
	SideUnknown Side = ""
	SideBuy     Side = "BUY"
	SideSell    Side = "SELL"
)

// Status mirrors MarketOrderStatusString values we care about.
type Status string

const (
	StatusUnknown   Status = ""
	StatusActive    Status = "Active"
	StatusFilled    Status = "Filled"
	StatusCancelled Status = "Cancelled"
	StatusFinished  Status = "Finished"
)

// Context conveys deal-level metadata that messages omit.
type Context struct {
	Strategy      Strategy
	BaseCurrency  string
	QuoteCurrency string
}

// Event is the parsed form of a bot event message.
type Event struct {
	Action           Action
	OrderType        OrderType
	Side             Side
	Status           Status
	Coin             string
	QuoteCurrency    string
	QuoteVolume      float64
	Price            float64
	IsMarket         bool
	Size             float64
	OrderPosition    int
	OrderSize        int
	Profit           float64
	ProfitCurrency   string
	ProfitUSD        float64
	ProfitPercentage float64
	Text             string
}

// ErrEmptyMessage indicates the parser received nothing useful.
var ErrEmptyMessage = errors.New("eventparser: empty message")

// Parse analyses a single bot event message.
func Parse(message string, ctx Context) (Event, error) {
	raw := strings.TrimSpace(message)
	if raw == "" {
		return Event{}, ErrEmptyMessage
	}

	normalized := normalize(raw)

	event := Event{
		Text: raw,
	}

	firstClause := firstSentence(normalized)

	action, subject := classifyAction(firstClause)
	event.Action = action
	event.OrderType = classifyOrderType(subject)
	event.Status = inferStatus(action)

	if pos, total, ok := parseProgress(subject); ok {
		event.OrderPosition = pos
		event.OrderSize = total
		if event.OrderType == OrderTypeUnknown {
			event.OrderType = OrderTypeSafety
		}
	}

	if price, currency, isMarket := parsePrice(normalized); currency != "" || isMarket {
		event.Price = price
		event.IsMarket = isMarket
		if !isMarket && currency != "" && event.QuoteCurrency == "" {
			event.QuoteCurrency = currency
		}
	}

	if quoteVol, quoteCur, baseVol, baseCur := parseSize(normalized); quoteVol > 0 || baseVol > 0 {
		if quoteVol > 0 {
			event.QuoteVolume = quoteVol
		}
		if quoteCur != "" {
			event.QuoteCurrency = quoteCur
		}
		if baseVol > 0 {
			event.Size = baseVol
		}
		if baseCur != "" {
			event.Coin = baseCur
		}
	}

	if profit, cur, usd, pct := parseProfit(normalized); profit != 0 || cur != "" || usd != 0 || pct != 0 {
		event.Profit = profit
		event.ProfitCurrency = cur
		event.ProfitUSD = usd
		event.ProfitPercentage = pct
	}

	if event.Coin == "" {
		event.Coin = ctx.BaseCurrency
	}
	if event.QuoteCurrency == "" {
		event.QuoteCurrency = ctx.QuoteCurrency
	}

	event.Side = inferSide(event.OrderType, ctx)

	return event, nil
}

func parseProfit(input string) (amount float64, currency string, usd float64, pct float64) {
	lower := strings.ToLower(input)
	if match := profitRe.FindStringSubmatch(input); len(match) == 3 {
		if val, err := strconv.ParseFloat(match[1], 64); err == nil {
			amount = val
			currency = match[2]
		}
	}

	if match := profitUSDRe.FindStringSubmatch(input); len(match) == 2 {
		if val, err := strconv.ParseFloat(match[1], 64); err == nil {
			usd = val
		}
	}

	if match := profitPctRe.FindStringSubmatch(input); len(match) == 2 {
		if val, err := strconv.ParseFloat(match[1], 64); err == nil {
			pct = val
		}
	}

	if amount == 0 && currency == "" && (strings.Contains(lower, "stop loss") || strings.Contains(lower, "stoploss")) && !strings.Contains(lower, "price:") {
		if matches := amountCurrencyRe.FindAllStringSubmatch(input, -1); len(matches) > 0 {
			for _, m := range matches {
				if len(m) != 3 {
					continue
				}
				val, err := strconv.ParseFloat(m[1], 64)
				if err != nil {
					continue
				}
				amount = val
				currency = m[2]
				break
			}
		}
	}

	return amount, currency, usd, pct
}

func normalize(input string) string {
	noEmoji := strings.Map(func(r rune) rune {
		if r > unicode.MaxASCII {
			return -1
		}
		return r
	}, input)
	beforeHash := noEmoji
	if idx := strings.Index(beforeHash, " #"); idx != -1 {
		beforeHash = beforeHash[:idx]
	}
	return strings.Join(strings.Fields(beforeHash), " ")
}

func firstSentence(input string) string {
	if idx := strings.Index(input, ". "); idx != -1 {
		return strings.TrimSuffix(input[:idx], ".")
	}
	return strings.TrimSuffix(input, ".")
}

func classifyAction(clause string) (Action, string) {
	lower := strings.ToLower(clause)

	switch {
	case strings.HasPrefix(lower, "placing "):
		return ActionPlace, strings.TrimSpace(clause[len("Placing "):])
	case strings.HasPrefix(lower, "cancelling "):
		return ActionCancel, strings.TrimSpace(clause[len("Cancelling "):])
	case strings.HasPrefix(lower, "takeprofit trade cancelled"):
		return ActionCancelled, strings.TrimSpace(clause)
	case strings.Contains(lower, "trade completed"):
		return ActionCompleted, strings.TrimSpace(clause)
	case strings.HasPrefix(lower, "stop loss") || strings.HasPrefix(lower, "stoploss"):
		return ActionCancelled, strings.TrimSpace(clause)
	case strings.HasSuffix(lower, " finished"):
		return ActionFinished, strings.TrimSpace(clause[:len(clause)-len(" finished")])
	case strings.HasSuffix(lower, " executed"):
		return ActionExecute, strings.TrimSpace(clause[:len(clause)-len(" executed")])
	case strings.HasSuffix(lower, " cancelled"):
		return ActionCancelled, strings.TrimSpace(clause[:len(clause)-len(" cancelled")])
	default:
		return ActionUnknown, strings.TrimSpace(clause)
	}
}

func classifyOrderType(subject string) OrderType {
	lower := strings.ToLower(subject)
	switch {
	case strings.Contains(lower, "base order"):
		return OrderTypeBase
	case strings.Contains(lower, "averaging order"):
		return OrderTypeSafety
	case strings.Contains(lower, "manual safety"):
		return OrderTypeManualSafety
	case strings.Contains(lower, "takeprofit"):
		return OrderTypeTakeProfit
	case strings.Contains(lower, "stop loss") || strings.Contains(lower, "stoploss"):
		return OrderTypeStopLoss
	case strings.Contains(lower, "trade completed"):
		return OrderTypeSummary
	default:
		return OrderTypeUnknown
	}
}

func parseProgress(subject string) (position int, total int, ok bool) {
	match := progressRe.FindStringSubmatch(subject)
	if len(match) != 3 {
		return 0, 0, false
	}
	pos, err1 := strconv.Atoi(match[1])
	tot, err2 := strconv.Atoi(match[2])
	if err1 != nil || err2 != nil {
		return 0, 0, false
	}
	return pos, tot, true
}

func parsePrice(input string) (price float64, currency string, isMarket bool) {
	match := priceRe.FindStringSubmatch(input)
	if len(match) < 2 {
		return 0, "", false
	}
	if match[1] == "market" {
		return 0, "", true
	}
	val, err := strconv.ParseFloat(match[1], 64)
	if err != nil {
		return 0, "", false
	}
	if len(match) >= 3 {
		return val, match[2], false
	}
	return val, "", false
}

func parseSize(input string) (quoteVol float64, quoteCur string, baseVol float64, baseCur string) {
	match := sizeRe.FindStringSubmatch(input)
	if len(match) < 3 {
		return 0, "", 0, ""
	}
	qVol, err := strconv.ParseFloat(match[1], 64)
	if err == nil {
		quoteVol = qVol
	}
	quoteCur = match[2]
	sizeSegment := input[strings.Index(input, match[0]):]
	if combos := baseSizeRe.FindAllStringSubmatch(sizeSegment, -1); len(combos) > 0 {
		last := combos[len(combos)-1]
		if len(last) >= 3 {
			if bVol, err := strconv.ParseFloat(last[1], 64); err == nil {
				baseVol = bVol
			}
			baseCur = last[2]
		}
	}
	return quoteVol, quoteCur, baseVol, baseCur
}

func inferStatus(action Action) Status {
	switch action {
	case ActionPlace:
		return StatusActive
	case ActionExecute:
		return StatusFilled
	case ActionCancel, ActionCancelled:
		return StatusCancelled
	case ActionFinished, ActionCompleted:
		return StatusFinished
	default:
		return StatusUnknown
	}
}

func inferSide(orderType OrderType, ctx Context) Side {
	if orderType == OrderTypeUnknown || orderType == OrderTypeSummary {
		return SideUnknown
	}
	switch ctx.Strategy {
	case StrategyLong:
		if orderType == OrderTypeTakeProfit || orderType == OrderTypeStopLoss {
			return SideSell
		}
		return SideBuy
	case StrategyShort:
		if orderType == OrderTypeTakeProfit || orderType == OrderTypeStopLoss {
			return SideBuy
		}
		return SideSell
	default:
		switch orderType {
		case OrderTypeTakeProfit, OrderTypeStopLoss:
			return SideSell
		case OrderTypeBase, OrderTypeSafety, OrderTypeManualSafety:
			return SideBuy
		default:
			return SideUnknown
		}
	}
}
