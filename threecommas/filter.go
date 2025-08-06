package threecommas

import "time"

func Filter[T any](s []T, keep func(T) bool) []T {
	result := make([]T, 0, len(s))
	for _, item := range s {
		if keep(item) {
			result = append(result, item)
		}
	}
	return result
}

func MarketOrderFilterCreatedAtAfter(u time.Time) func(o MarketOrder) bool {
	return func(o MarketOrder) bool {
		return o.CreatedAt.After(u)
	}
}

func MarketOrderFilterStatusString(status MarketOrderStatusString) func(o MarketOrder) bool {
	return func(o MarketOrder) bool {
		return o.StatusString == status
	}
}

func MarketOrderFilter(orderType MarketOrderOrderType) func(o MarketOrder) bool {
	return func(o MarketOrder) bool {
		return o.OrderType == orderType
	}
}
