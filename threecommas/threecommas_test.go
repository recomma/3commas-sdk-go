// see https://pkg.go.dev/crypto/rsa#hdr-Minimum_key_size
//
//go:debug rsa1024min=0
package threecommas

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/cassette"
	"gopkg.in/dnaeon/go-vcr.v4/pkg/recorder"
)

var record = true

func defaultRecorderOpts(record bool) []recorder.Option {
	opts := []recorder.Option{
		recorder.WithHook(func(i *cassette.Interaction) error {
			i.Request.Headers.Del("Authorization")
			i.Request.Headers.Del("Apikey")
			i.Request.Headers.Del("Signature")
			return nil
		}, recorder.AfterCaptureHook),
		recorder.WithMatcher(cassette.NewDefaultMatcher(
			cassette.WithIgnoreHeaders("Authorization", "Apikey", "Signature"))),
		recorder.WithSkipRequestLatency(true),
	}

	if record {
		opts = append(opts,
			recorder.WithMode(recorder.ModeReplayWithNewEpisodes),
			recorder.WithRealTransport(http.DefaultTransport),
		)
	} else {
		opts = append(opts, recorder.WithMode(recorder.ModeReplayOnly))
	}

	return opts
}

func TestListBots(t *testing.T) {
	type tc struct {
		name         string
		cassetteName string
		config       ClientConfig
		options      []ListBotsParamsOption
		wantErr      string
		record       bool
	}

	cases := []tc{
		{
			name:    "invalid auth",
			config:  ClientConfig{APIKey: "somefakeapikey", PrivatePEM: []byte(fakeKey)},
			options: []ListBotsParamsOption{},
			wantErr: "API error 401: Unauthorized. Invalid or expired api key.",
		},
		{
			name:         "all bots",
			cassetteName: "Bots",
			config:       config,
			options:      []ListBotsParamsOption{},
			record:       true,
		},
		{
			name:   "filter on account",
			config: config,
			// cassetteName: "Bots",
			options: []ListBotsParamsOption{
				WithAccountIdForListBots(33256512),
			},
		},
		{
			name:         "enabled bots",
			config:       config,
			cassetteName: "Bots",
			options: []ListBotsParamsOption{
				WithScopeForListBots(Enabled),
			},
			// record: true,
		},
		{
			name:         "disabled bots",
			config:       config,
			cassetteName: "Bots",
			options: []ListBotsParamsOption{
				WithScopeForListBots(Disabled),
			},
			// record: true,
		},
		{
			name:   "Bots from certain create date",
			config: config,
			options: func() []ListBotsParamsOption {
				ts, _ := time.Parse(time.RFC3339, "2025-07-04T17:07:14Z")
				return []ListBotsParamsOption{
					WithFromForListBots(ts),
				}
			}(),
			record: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(tt *testing.T) {
			client, err := getClient(tt, tc.config, tc.record, tc.cassetteName)
			if err != nil {
				tt.Fatalf("Could not create client: %s", err)
			}

			bots, err := client.ListBots(context.Background(), tc.options...)
			if tc.wantErr != "" {
				require.EqualError(tt, err, tc.wantErr)
				return
			}

			require.NoError(tt, err)
			require.NotEmpty(tt, bots, "expected at least one bot, got empty list")
		})
	}
}

func TestGetListOfDeals(t *testing.T) {
	type tc struct {
		name         string
		cassetteName string
		config       ClientConfig
		options      []ListDealsParamsOption
		wantErr      string
		record       bool
		bots         []Bot
	}

	cases := []tc{
		{
			name:         "valid request",
			cassetteName: "Bots",
			config:       config,
			// record:       true,
		},
		{
			name:         "specific bot 16403596",
			cassetteName: "Bots",
			config:       config,
			options: []ListDealsParamsOption{
				WithBotIdForListDeals(16403596),
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(tt *testing.T) {
			client, err := getClient(tt, tc.config, tc.record, tc.cassetteName)
			require.NoErrorf(tt, err, "could not create client")

			bots := tc.bots
			if len(bots) == 0 {
				bots, err = client.ListBots(context.Background())
				if err != nil {
					tt.Fatalf("cannot list bots: %s", err)
				}
			}

			for _, bot := range bots {
				tc.options = append(tc.options, WithBotIdForListDeals(bot.Id))
				deals, err := client.GetListOfDeals(context.Background(), tc.options...)
				if err != nil {
					tt.Fatalf("Could not list deals: %s", err)
				}

				if tc.wantErr != "" {
					require.EqualError(tt, err, tc.wantErr)
					return
				}

				require.NoError(tt, err)
				require.NotEmpty(tt, deals, "expected at least one deal, got empty list")
			}
		})
	}
}

func TestGetTradesForDeal(t *testing.T) {
	type tc struct {
		name         string
		cassetteName string
		config       ClientConfig
		dealId       DealPathId
		wantErr      string
		record       bool
	}

	cases := []tc{
		{
			name:         "404",
			cassetteName: "Bots",
			config:       config,
			dealId:       1374390720784,
			wantErr:      "API error 404: Not Found",
		},
		{
			name:         "valid request",
			cassetteName: "Bots",
			config:       config,
			dealId:       2362612144,
		},
		{
			name:         "lots of market orders",
			cassetteName: "marketorders",
			config:       config,
			dealId:       2366275139,
			// record:       true,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(tt *testing.T) {
			client, err := getClient(tt, tc.config, tc.record, tc.cassetteName)
			require.NoErrorf(tt, err, "could not create client")

			trades, err := client.GetTradesForDeal(context.Background(), tc.dealId)
			if tc.wantErr != "" {
				require.EqualError(tt, err, tc.wantErr)
				return
			}

			require.NoError(tt, err)
			require.NotEmpty(tt, trades, "expected at least one trade, got empty list")
		})
	}
}

func TestGetMarketOrdersForDeal(t *testing.T) {
	client, err := getClient(t, config, false, "marketorders")
	require.NoErrorf(t, err, "could not create client")

	trades, err := client.GetMarketOrdersForDeal(context.Background(), 2366275139)
	require.NoError(t, err)
	require.NotEmpty(t, trades, "expected at least one trade, got empty list")

	// filtered := Filter(trades, func(o MarketOrder) bool {
	// 	return o.StatusString == MarketOrderStatusStringFilled
	// })

	filtered := Filter(trades, MarketOrderFilterStatusString(Filled))
	timestamp, err := time.Parse(time.RFC3339, "2025-08-04T17:07:14Z")
	require.NoErrorf(t, err, "Could not parse date")
	filtered = Filter(filtered, MarketOrderFilterCreatedAtAfter(timestamp))

	for _, trade := range filtered {
		tr, err := json.MarshalIndent(trade, "", "  ")
		if err != nil {
			log.Printf("failed to marshal trade: %v", err)
			continue
		}
		t.Logf("trade: %s", tr)
	}
}

func getClient(t *testing.T, config ClientConfig, record bool, cassetteName string) (*ThreeCommasClient, error) {
	opts := defaultRecorderOpts(record)

	base := strings.ReplaceAll(t.Name(), "/", "_")
	cassette := filepath.Join("testdata", func() string {
		if cassetteName != "" {
			return cassetteName
		}
		return base
	}())

	r, err := recorder.New(cassette, opts...)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		// Make sure recorder is stopped once done with it.
		if err := r.Stop(); err != nil {
			t.Error(err)
		}
	})

	httpClient := &http.Client{Transport: r}

	return New3CommasClient(config, WithHTTPClient(httpClient))
}

func getIntPointer(val int) *int {
	return &val
}

func getStringPointer(val string) *string {
	return &val
}

var fakeKey = `-----BEGIN RSA PRIVATE KEY-----
MIIBOQIBAAJAeR3EpgKGuWoCNWzIjRj34pQPoFD+hAqZl2jcfPma5xST4rTP0k+W
Wk8R6yGMB5wBxdTQpKAM0KzSWc4GlCee5wIDAQABAkAam72eMyPiDDYcAqA0z212
K80bDXA9Fg8UQodeNYAgkAlia9oc4mN9NJhacE64u0fKZiDBCiiLXCmJ/uOP4y2R
AiEAs75ndPumbOjG0Jtz1pHcnr3t9VLx6l/BIBUE89rORjMCIQCsf/SD5dYRcobE
+S8Fjyxe1yZY5eFQQGdS/9N29ItIfQIgXz7+Q5c2UW/oKpK1h3Yzmkq61czmNHQZ
Oo7o2O+RbtECIBb1CIOtSOoVhd4dE6b3wP32QEJAhdX6XEXtiiUgspC5AiEAidSE
m3b2qAUjJbT8LPdr/JordWF7RjdWrh3l7pUr1PE=
-----END RSA PRIVATE KEY-----`
var fakePublic = `-----BEGIN PUBLIC KEY-----
MFswDQYJKoZIhvcNAQEBBQADSgAwRwJAeR3EpgKGuWoCNWzIjRj34pQPoFD+hAqZ
l2jcfPma5xST4rTP0k+WWk8R6yGMB5wBxdTQpKAM0KzSWc4GlCee5wIDAQAB
-----END PUBLIC KEY-----`
