package threecommas

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"golang.org/x/time/rate"
)

func WithRateLimit(l *rate.Limiter) RequestEditorFn {
	return func(ctx context.Context, req *http.Request) error {
		if err := l.Wait(ctx); err != nil {
			return fmt.Errorf("rate limiter wait failed: %w", err)
		}
		return nil
	}
}

// WithDefaultRatelimit sets a rate limit of 100 requests per minute
func WithDefaultRatelimit() ClientOption {
	limiter := rate.NewLimiter(rate.Every(time.Minute), 100)
	return WithRequestEditorFn(WithRateLimit(limiter))
}
