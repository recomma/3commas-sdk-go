// see https://pkg.go.dev/crypto/rsa#hdr-Minimum_key_size
//
//go:debug rsa1024min=0
package threecommas

import (
	"context"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

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

func TestListBotsWithResponse(t *testing.T) {
	type tc struct {
		name         string
		cassetteName string
		config       ClientConfig
		params       *ListBotsParams
		wantStatus   int
		wantErrCode  string
		record       bool
	}

	cases := []tc{
		{
			name:        "invalid auth",
			config:      ClientConfig{APIKey: "somefakeapikey", PrivatePEM: []byte(fakeKey)},
			params:      &ListBotsParams{},
			wantStatus:  http.StatusUnauthorized,
			wantErrCode: "API error 401: Unauthorized. Invalid or expired api key.",
		},
		{
			name:         "valid request",
			cassetteName: "Bots",
			config:       config,
			params:       &ListBotsParams{},
			wantStatus:   http.StatusOK,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(tt *testing.T) {
			client, err := getClient(tt, tc.config, tc.record, tc.cassetteName)
			if err != nil {
				tt.Fatalf("Could not create client: %s", err)
			}

			resp, err := client.ListBotsWithResponse(context.Background(), &ListBotsParams{})
			if err != nil {
				tt.Fatalf("Could not list bots: %s", err)
			}

			if got := resp.StatusCode(); got != tc.wantStatus {
				tt.Errorf("status = %d; want %d", got, tc.wantStatus)
			}

			if tc.wantErrCode != "" {
				errResp := GetErrorFromResponse(resp)
				if errResp == nil {
					tt.Errorf("expected error code %q, got none", tc.wantErrCode)
				} else if errResp.Error() != tc.wantErrCode {
					tt.Errorf("error code = %q; want %q", errResp.Error(), tc.wantErrCode)
				}
				return
			}

			if resp.JSON200 == nil {
				tt.Error("expected JSON200 payload, got nil")
			} else if len(*resp.JSON200) == 0 {
				tt.Error("expected at least one bot, got empty list")
			}
		})
	}
}

func TestListDealsWithResponse(t *testing.T) {
	type tc struct {
		name             string
		cassetteName     string
		config           ClientConfig
		params           *ListDealsParams
		wantStatus       int
		wantErrCode      string
		record           bool
		requiresListBots bool
	}

	cases := []tc{
		{
			name:             "valid request",
			cassetteName:     "Bots",
			config:           config,
			params:           &ListDealsParams{},
			wantStatus:       http.StatusOK,
			requiresListBots: true,
			// record:       true,
		},
		// {
		// 	// this test is futile, their API just sends back StatusOK
		// 	// on a Bot that doesn't exist for us
		// 	name: "invalid request",
		// 	// cassetteName:     "Bots",
		// 	config:           config,
		// 	params:           &ListDealsParams{BotId: getIntPointer(5)},
		// 	wantStatus:       http.StatusOK,
		// 	requiresListBots: false,
		// 	// record:           true,
		// },
	}

	for _, tc := range cases {
		t.Run(tc.name, func(tt *testing.T) {
			client, err := getClient(tt, tc.config, tc.record, tc.cassetteName)
			if err != nil {
				tt.Fatalf("Could not create client: %s", err)
			}

			// determine which bot IDs to test against
			var botIDs []int
			if tc.requiresListBots {
				listResp, err := client.ListBotsWithResponse(context.Background(), &ListBotsParams{})
				if err != nil {
					tt.Fatalf("Could not list bots: %s", err)
				}

				if listResp.StatusCode() != http.StatusOK {
					if errResponse := GetErrorFromResponse(listResp); errResponse != nil {
						tt.Fatal(errResponse)
					}
					tt.Fatalf("unexpected status: %d", listResp.StatusCode())
				}
			} else {
				if tc.params.BotId == nil {
					t.Fatal("test case must provide BotId when requiresListBots is false")
				}
				botIDs = []int{*tc.params.BotId}
			}

			for _, id := range botIDs {
				tc.params.BotId = &id
				resp, err := client.ListDealsWithResponse(context.Background(), tc.params)
				if err != nil {
					tt.Fatalf("Could not list deals: %s", err)
				}

				if got := resp.StatusCode(); got != tc.wantStatus {
					tt.Errorf("status = %d; want %d", got, tc.wantStatus)
				}

				if tc.wantErrCode != "" {
					errResp := GetErrorFromResponse(resp)
					if errResp == nil {
						tt.Errorf("expected error code %q, got none", tc.wantErrCode)
					} else if errResp.Error() != tc.wantErrCode {
						tt.Errorf("error code = %q; want %q", errResp.Error(), tc.wantErrCode)
					}
					return
				}

				if resp.JSON200 == nil {
					tt.Error("expected JSON200 payload, got nil")
				} else if len(*resp.JSON200) == 0 {
					tt.Error("expected at least one deal, got empty list")
				}
			}
		})
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
