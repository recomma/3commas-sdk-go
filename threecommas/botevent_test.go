package threecommas

import (
	"context"
	"fmt"
	"log"
	"strings"
	"testing"

	"github.com/recomma/3commas-sdk-go/threecommas/eventparser"
	"github.com/stretchr/testify/require"
)

func TestGetDeal(t *testing.T) {
	type tc struct {
		cassetteName string
		config       ClientConfig
		dealId       DealPathId
		wantErr      string
		record       bool
		skip         bool
	}

	cases := []tc{
		{
			cassetteName: "getdeal",
			config:       config,
			dealId:       2376446537,
		},
		{
			cassetteName: "getdeal",
			config:       config,
			dealId:       2376405906,
		},
		{
			cassetteName: "getdeal",
			config:       config,
			dealId:       2376389742,
		},
		{
			cassetteName: "getdeal",
			config:       config,
			record:       false,
			dealId:       2376436112,
		},
		{
			cassetteName: "getdeal",
			config:       config,
			record:       false,
			dealId:       2376434644,
		},
		{
			cassetteName: "getdeal",
			config:       config,
			record:       false,
			dealId:       2376134037,
		},
		{
			cassetteName: "getdeal",
			config:       config,
			record:       false,
			dealId:       0,
			skip:         true,
		},
	}
	for _, tc := range cases {
		if tc.skip {
			continue
		}
		var dealIds []DealPathId
		if tc.dealId == 0 {
			// we gonna loop da loop!
			client, err := getClient(t, tc.config, tc.record, tc.cassetteName)
			require.NoErrorf(t, err, "could not create client")

			deals, err := client.GetListOfDeals(context.Background(), WithLimitForListDeals(1000))
			require.NoErrorf(t, err, "could not list deals")

			for _, d := range deals {
				dealIds = append(dealIds, d.Id)
			}
		} else {
			dealIds = append(dealIds, tc.dealId)
		}

		for _, dealId := range dealIds {
			t.Run(fmt.Sprintf("Deal %d", dealId), func(tt *testing.T) {
				client, err := getClient(tt, tc.config, tc.record, tc.cassetteName)
				require.NoErrorf(tt, err, "could not create client")

				deal, err := client.GetDealForID(context.Background(), dealId)
				if tc.wantErr != "" {
					require.EqualError(tt, err, tc.wantErr)
					return
				}

				require.NoError(tt, err)
				require.NotEmpty(tt, deal, "expected at least one deal, got nothing")

				ctx := eventparser.Context{
					Strategy:      DealStrategy(deal),
					BaseCurrency:  strings.ToUpper(deal.ToCurrency),
					QuoteCurrency: strings.ToUpper(deal.FromCurrency),
				}

				for _, raw := range deal.BotEvents {
					if raw.Message == nil {
						continue
					}

					parsed, err := eventparser.Parse(*raw.Message, ctx)
					if err != nil {
						log.Printf("Failed to parse: %s", err)
						continue
					}

					if parsed.Action == eventparser.ActionUnknown {
						t.Logf("%#v", parsed)
						t.Fatalf("ActionUnknown")
					}

					if parsed.OrderType == eventparser.OrderTypeUnknown {
						t.Logf("%#v", parsed)
						t.Fatalf("OrderTypeUnknown")
					}

					// summary events are allowed to have no side
					if parsed.OrderType == eventparser.OrderTypeSummary {
						continue
					}

					if parsed.Side == eventparser.SideUnknown {
						t.Logf("%#v", parsed)
						t.Fatalf("SideUnknown")
					}
				}

				// var output strings.Builder

				// // now we can check the events
				// for _, event := range deal.Events() {
				// 	output.WriteString(fmt.Sprintf("%#v\n", event))
				// }

				// t.Logf("events:\n\n%s", output.String())

			})
		}
	}
}
