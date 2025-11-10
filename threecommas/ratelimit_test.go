package threecommas

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestPlanTierRateLimits(t *testing.T) {
	tests := []struct {
		name            string
		tier            PlanTier
		expectedLimit   int
		timeWindow      time.Duration
		requestsToMake  int
		expectBlocking  bool
	}{
		{
			name:            "Starter tier - 5 req/min",
			tier:            PlanStarter,
			expectedLimit:   5,
			timeWindow:      time.Minute,
			requestsToMake:  6,
			expectBlocking:  true,
		},
		{
			name:            "Pro tier - 50 req/min",
			tier:            PlanPro,
			expectedLimit:   50,
			timeWindow:      time.Minute,
			requestsToMake:  51,
			expectBlocking:  true,
		},
		{
			name:            "Expert tier - 120 req/min",
			tier:            PlanExpert,
			expectedLimit:   120,
			timeWindow:      time.Minute,
			requestsToMake:  10, // just test a few
			expectBlocking:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var requestCount atomic.Int32

			// Create a test server that counts requests
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestCount.Add(1)
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"id": 123}`))
			}))
			defer server.Close()

			// Create client with specific tier
			client, err := New3CommasClient(
				WithAPIKey("test-key"),
				WithPrivatePEM([]byte(fakeKey)),
				WithThreeCommasBaseURL(server.URL),
				WithPlanTier(tt.tier),
			)
			require.NoError(t, err)

			// Make rapid requests up to the limit
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()

			start := time.Now()
			successCount := 0

			for i := 0; i < tt.requestsToMake; i++ {
				_, err := client.GetDealWithResponse(ctx, DealPathId(123))
				if err != nil {
					// Context timeout or other error
					break
				}
				successCount++
			}
			elapsed := time.Since(start)

			t.Logf("Made %d successful requests in %v", successCount, elapsed)

			// If we expect blocking and made more requests than the limit,
			// it should take noticeably longer than instant
			if tt.expectBlocking && successCount > tt.expectedLimit {
				// Should have been rate limited
				minExpectedTime := time.Duration(float64(time.Minute) / float64(tt.expectedLimit))
				require.Greater(t, elapsed, minExpectedTime,
					"Expected rate limiting to slow down requests beyond limit")
			}
		})
	}
}

func TestWithPlanTierOption(t *testing.T) {
	tests := []struct {
		name string
		tier PlanTier
	}{
		{"Starter", PlanStarter},
		{"Pro", PlanPro},
		{"Expert", PlanExpert},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := New3CommasClient(
				WithAPIKey("test-key"),
				WithPrivatePEM([]byte(fakeKey)),
				WithPlanTier(tt.tier),
			)
			require.NoError(t, err)
			require.NotNil(t, client)
		})
	}
}

func TestDefaultPlanTier(t *testing.T) {
	// When no tier is specified, should default to Expert
	client, err := New3CommasClient(
		WithAPIKey("test-key"),
		WithPrivatePEM([]byte(fakeKey)),
	)
	require.NoError(t, err)
	require.NotNil(t, client)
	// Default should be Expert (120 req/min), but we can't easily test the internal state
	// Just verify the client was created successfully
}

func TestGlobalLimiterForTier(t *testing.T) {
	tests := []struct {
		name          string
		tier          PlanTier
		expectedBurst int
	}{
		{"Starter", PlanStarter, 5},
		{"Pro", PlanPro, 50},
		{"Expert", PlanExpert, 120},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limiter := globalLimiterForTier(tt.tier)
			require.NotNil(t, limiter)
			require.Equal(t, tt.expectedBurst, limiter.Burst())
		})
	}
}
