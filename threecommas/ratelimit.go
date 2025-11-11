package threecommas

import (
	"context"
	"net/http"
	"regexp"
	"strconv"
	"sync"
	"time"
)

// PlanTier represents the 3commas subscription tier
type PlanTier int

const (
	// PlanStarter: 5 requests per minute (read-only)
	PlanStarter PlanTier = iota
	// PlanPro: 50 requests per minute (read-only)
	PlanPro
	// PlanExpert: 120 requests per minute (read/write)
	PlanExpert
)

// fixedWindowLimiter implements a fixed-window rate limiter that aligns to clock boundaries.
// For example, with a 1-minute window, windows align to 12:30:00, 12:31:00, 12:32:00, etc.
// This matches the 3commas API rate limiting behavior where limits reset at the start of each window.
type fixedWindowLimiter struct {
	windowSize  time.Duration
	limit       int
	mu          sync.Mutex
	windowStart time.Time
	count       int
}

func newFixedWindowLimiter(windowSize time.Duration, limit int) *fixedWindowLimiter {
	return &fixedWindowLimiter{
		windowSize: windowSize,
		limit:      limit,
	}
}

// Wait blocks until the limiter allows the request or context is cancelled.
// It uses clock-aligned windows that reset at fixed time boundaries.
func (l *fixedWindowLimiter) Wait(ctx context.Context) error {
	for {
		l.mu.Lock()
		now := time.Now()

		// Align to window boundary (e.g., 12:30:37 -> 12:30:00 for 1-minute window)
		currentWindowStart := now.Truncate(l.windowSize)

		// If we've entered a new window, reset the counter
		if currentWindowStart.After(l.windowStart) {
			l.windowStart = currentWindowStart
			l.count = 0
		}

		// Check if we can make a request in this window
		if l.count < l.limit {
			l.count++
			l.mu.Unlock()
			return nil
		}

		// Need to wait for next window
		nextWindow := l.windowStart.Add(l.windowSize)
		l.mu.Unlock()

		waitDuration := time.Until(nextWindow)
		if waitDuration <= 0 {
			// Window should have already passed, try again
			continue
		}

		timer := time.NewTimer(waitDuration)
		select {
		case <-ctx.Done():
			timer.Stop()
			return ctx.Err()
		case <-timer.C:
			// Window expired, try again
		}
	}
}

func tierLimiterForPlan(tier PlanTier) *fixedWindowLimiter {
	switch tier {
	case PlanStarter:
		return newFixedWindowLimiter(time.Minute, 5)
	case PlanPro:
		return newFixedWindowLimiter(time.Minute, 50)
	case PlanExpert:
		return newFixedWindowLimiter(time.Minute, 120)
	default:
		return tierLimiterForPlan(PlanExpert)
	}
}

type routeLimiter struct {
	name       string
	method     string
	re         *regexp.Regexp
	limiter    *fixedWindowLimiter
	mitigation time.Duration // how long to block after 429 (unless Retry-After overrides)
}

func threeCommasRoutes() []routeLimiter {
	return []routeLimiter{
		{
			name:       "deals_list",
			method:     http.MethodGet,
			re:         regexp.MustCompile(`^/ver1/deals$`),
			limiter:    newFixedWindowLimiter(time.Minute, 120), // 120/min
			mitigation: 60 * time.Second,
		},
		{
			name:       "deal_show",
			method:     http.MethodGet,
			re:         regexp.MustCompile(`^/ver1/deals/\d+/show$`),
			limiter:    newFixedWindowLimiter(time.Minute, 120), // 120/min
			mitigation: 60 * time.Second,
		},
		{
			name:       "smart_trades",
			method:     http.MethodGet,
			re:         regexp.MustCompile(`^/ver1/smart_trades(?:/|$)`),
			limiter:    newFixedWindowLimiter(10*time.Second, 40), // 40 / 10s
			mitigation: 10 * time.Second,
		},
	}
}

type rlEngine struct {
	tier    *fixedWindowLimiter
	routes  []routeLimiter
	mu      sync.Mutex
	blocked map[string]time.Time // key: "tier" or route.name -> blocked-until
}

func newRLEngine(tier PlanTier) *rlEngine {
	return &rlEngine{
		tier:    tierLimiterForPlan(tier),
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
	if err := d.eng.waitBlocked(req.Context(), "tier"); err != nil {
		return nil, err
	}
	if matched := d.eng.match(req); matched != nil {
		if err := d.eng.waitBlocked(req.Context(), matched.name); err != nil {
			return nil, err
		}
	}

	// Wait on TIER limiter first (subscription plan limit)
	if err := d.eng.tier.Wait(req.Context()); err != nil {
		return nil, err
	}
	// Wait on ROUTE limiter if matched (additional per-endpoint limit)
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
		// Getting a 429 means our rate limiting failed to prevent it.
		// Since the TIER limit is the primary constraint for most users,
		// we should block the TIER limiter (not just the route).
		block := 5 * time.Minute // default per docs
		if matched := d.eng.match(req); matched != nil {
			block = matched.mitigation
		}
		if ra := parseRetryAfter(resp.Header.Get("Retry-After")); ra > 0 {
			block = ra // prefer server hint
		}
		// Always block TIER limiter since it's the account-wide limit
		d.eng.backoff("tier", block)
		// Also block the specific route if matched
		if matched := d.eng.match(req); matched != nil {
			d.eng.backoff(matched.name, block)
		}
	case 418: // auto-ban
		// Be conservative: set a generous block on the tier limiter.
		d.eng.backoff("tier", 10*time.Minute)
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

// WithThreeCommasRateLimits installs the rate limiter for the specified tier.
// If tier is not specified, defaults to PlanExpert.
func WithThreeCommasRateLimits(tier ...PlanTier) ClientOption {
	return func(c *Client) error {
		t := PlanExpert
		if len(tier) > 0 {
			t = tier[0]
		}

		// IMPORTANT: if c.Client is nil here, NewClient will NOT assign a default later
		// once we set c.Client to our wrapper. So make sure the wrapper has a non-nil base.
		base := c.Client
		if base == nil {
			base = &http.Client{}
		}
		c.Client = &rateLimitDoer{
			base: base,
			eng:  newRLEngine(t),
		}
		return nil
	}
}
