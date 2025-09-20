package threecommas

import (
	"context"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"

	"golang.org/x/time/rate"
)

type routeLimiter struct {
	name       string
	method     string
	re         *regexp.Regexp
	limiter    *rate.Limiter
	mitigation time.Duration // how long to block after 429 (unless Retry-After overrides)
}

func threeCommasRoutes() []routeLimiter {
	return []routeLimiter{
		{
			name:       "deals_list",
			method:     http.MethodGet,
			re:         regexp.MustCompile(`^/ver1/deals$`),
			limiter:    rate.NewLimiter(rate.Every(time.Minute/120), 120), // 120/min
			mitigation: 60 * time.Second,
		},
		{
			name:       "deal_show",
			method:     http.MethodGet,
			re:         regexp.MustCompile(`^/ver1/deals/\d+/show$`),
			limiter:    rate.NewLimiter(rate.Every(time.Minute/120), 120), // 120/min
			mitigation: 60 * time.Second,
		},
		{
			name:       "smart_trades",
			method:     http.MethodGet,
			re:         regexp.MustCompile(`^/ver1/smart_trades(?:/|$)`),
			limiter:    rate.NewLimiter(rate.Every((10*time.Second)/40), 40), // 40 / 10s
			mitigation: 10 * time.Second,
		},
	}
}

type rlEngine struct {
	global  *rate.Limiter
	routes  []routeLimiter
	mu      sync.Mutex
	blocked map[string]time.Time // key: "global" or route.name -> blocked-until
}

func newRLEngine() *rlEngine {
	return &rlEngine{
		// Correct 100/minute config: token every 600ms, burst 100
		global:  rate.NewLimiter(rate.Every(time.Minute/100), 100),
		routes:  threeCommasRoutes(),
		blocked: make(map[string]time.Time),
	}
}

func (e *rlEngine) match(r *http.Request) *routeLimiter {
	path := r.URL.EscapedPath()
	for i := range e.routes {
		rl := &e.routes[i]
		if rl.method == r.Method && rl.re.MatchString(path) {
			return rl
		}
	}
	return nil
}

func (e *rlEngine) waitBlocked(ctx context.Context, key string) error {
	for {
		e.mu.Lock()
		until := e.blocked[key]
		e.mu.Unlock()

		if until.IsZero() {
			return nil
		}
		d := time.Until(until)
		if d <= 0 {
			e.mu.Lock()
			delete(e.blocked, key)
			e.mu.Unlock()
			return nil
		}
		t := time.NewTimer(d)
		select {
		case <-ctx.Done():
			t.Stop()
			return ctx.Err()
		case <-t.C:
		}
	}
}

func (e *rlEngine) backoff(key string, d time.Duration) {
	if d <= 0 {
		return
	}
	deadline := time.Now().Add(d)
	e.mu.Lock()
	if cur, ok := e.blocked[key]; !ok || deadline.After(cur) {
		e.blocked[key] = deadline
	}
	e.mu.Unlock()
}

type rateLimitDoer struct {
	base HttpRequestDoer
	eng  *rlEngine
}

func (d *rateLimitDoer) Do(req *http.Request) (*http.Response, error) {
	// Respect any active blocks
	if err := d.eng.waitBlocked(req.Context(), "global"); err != nil {
		return nil, err
	}
	if matched := d.eng.match(req); matched != nil {
		if err := d.eng.waitBlocked(req.Context(), matched.name); err != nil {
			return nil, err
		}
	}

	// Wait on global and per-route buckets
	if err := d.eng.global.Wait(req.Context()); err != nil {
		return nil, err
	}
	if matched := d.eng.match(req); matched != nil {
		if err := matched.limiter.Wait(req.Context()); err != nil {
			return nil, err
		}
	}

	// Send
	resp, err := d.base.Do(req)
	if err != nil {
		return resp, err
	}

	// Observe and react
	switch resp.StatusCode {
	case http.StatusTooManyRequests: // 429
		block := 5 * time.Minute // global default per docs
		if matched := d.eng.match(req); matched != nil {
			block = matched.mitigation
		}
		if ra := parseRetryAfter(resp.Header.Get("Retry-After")); ra > 0 {
			block = ra // prefer server hint
		}
		if matched := d.eng.match(req); matched != nil {
			d.eng.backoff(matched.name, block)
		} else {
			d.eng.backoff("global", block)
		}
	case 418: // auto-ban
		// Be conservative: set a generous global block so callers don’t loop.
		d.eng.backoff("global", 10*time.Minute)
	}

	return resp, nil
}

func parseRetryAfter(v string) time.Duration {
	if v == "" {
		return 0
	}
	if secs, err := strconv.Atoi(v); err == nil && secs > 0 {
		return time.Duration(secs) * time.Second
	}
	if when, err := http.ParseTime(v); err == nil {
		if d := time.Until(when); d > 0 {
			return d
		}
	}
	return 0
}

// Public option to install the limiter+backoff around the oapi-codegen client.
func WithThreeCommasRateLimits() ClientOption {
	return func(c *Client) error {
		// IMPORTANT: if c.Client is nil here, NewClient will NOT assign a default later
		// once we set c.Client to our wrapper. So make sure the wrapper has a non-nil base.
		base := c.Client
		if base == nil {
			base = &http.Client{}
		}
		c.Client = &rateLimitDoer{
			base: base,
			eng:  newRLEngine(),
		}
		return nil
	}
}
