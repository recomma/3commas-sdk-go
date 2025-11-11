# 3Commas Go SDK

This is a **community-maintained Go SDK** for the [3Commas API](https://developers.3commas.io), generated from an OpenAPI 3.0 specification. It enables access to 3Commas trading bots, deals, market data, and account-level features in idiomatic Go.

> **Disclaimer:** This SDK is provided "as is" with no guarantee of correctness or completeness. It is a best-effort tool built on top of the public 3Commas API documentation. Use at your own risk.

## Installation

```bash
go get github.com/recomma/3commas-sdk-go/threecommas
```

## Usage

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/recomma/3commas-sdk-go/threecommas"
)

func main() {
	privateKey, _ := os.ReadFile("private.pem")

	client, err := threecommas.New3CommasClient(
		threecommas.WithAPIKey("your-api-key"),
		threecommas.WithPrivatePEM(privateKey),
	)
	if err != nil {
		log.Fatal(err)
	}

	// List all bots for a specific account, limit the results, and sort by profit
	bots, err := client.ListBots(
		context.Background(),
		threecommas.WithAccountIdForListBots(12345678),
		threecommas.WithLimitForListBots(50),
		threecommas.WithSortByForListBots(threecommas.ListBotsParamsSortByProfit),
	)
	if err != nil {
		log.Fatalf("failed to list bots: %v", err)
	}

	for _, bot := range bots {
		fmt.Printf("Bot ID: %d, Name: %s, Profit: %s\n", bot.Id, *bot.Name, bot.ActiveDealsUsdProfit)
	}
}
```

## Configuration

The client uses functional options for configuration:

```go
client, err := threecommas.New3CommasClient(
	threecommas.WithAPIKey("your-api-key"),
	threecommas.WithPrivatePEM(privateKeyBytes),
	threecommas.WithPlanTier(threecommas.PlanPro),              // Optional: defaults to PlanExpert
	threecommas.WithThreeCommasBaseURL("https://custom-url"),   // Optional: defaults to official API
)
```

## Rate Limiting

The SDK includes automatic rate limiting based on your 3Commas subscription tier:

| Tier | Rate Limit | Access |
|------|-----------|--------|
| **Starter** | 5 requests/minute | Read-only |
| **Pro** | 50 requests/minute | Read-only |
| **Expert** | 120 requests/minute | Read/Write |

Configure your tier using the `WithPlanTier()` option:

```go
client, err := threecommas.New3CommasClient(
	threecommas.WithAPIKey("your-api-key"),
	threecommas.WithPrivatePEM(privateKey),
	threecommas.WithPlanTier(threecommas.PlanPro),  // Set your subscription tier
)
```

If not specified, the SDK defaults to `PlanExpert` (120 req/min). The rate limiter:
- Uses fixed-window rate limiting aligned to clock boundaries (matching 3Commas API behavior)
- Allows bursts within the same time window (e.g., all 5 requests in 2 seconds is fine for Starter)
- Proactively prevents 429 (rate limit) errors before they occur
- Automatically handles 429 responses with backoff if limits are exceeded
- Respects `Retry-After` headers from the server
- Protects against IP auto-ban (418 responses) with 10-minute cooldown

## Authentication

This SDK uses RSA signature-based authentication with your API key and a PEM-encoded private key. Every request is signed using your private key per 3Commas API requirements.

## Code Generation

Most of this SDK is automatically generated from an OpenAPI specification. Note that 3Commas does **not** provide an official OpenAPI spec. Instead, a community-maintained version is available here:

**→ [terwey/3commas-openapi](https://github.com/terwey/3commas-openapi)**

To regenerate the SDK after modifying the spec, run:

```bash
go generate ./...
```

This uses [oapi-codegen](https://github.com/deepmap/oapi-codegen) to produce typed clients and response structs.

## Error Handling

The SDK offers structured error decoding via `GetErrorFromResponse`, which returns a Go `error`. You can use `errors.As` to unwrap it into an `APIError`, which contains the full parsed JSON `*ErrorResponse` payload. This allows idiomatic Go-style error inspection and recovery.

Example:

```go
if err := threecommas.GetErrorFromResponse(resp); err != nil {
	var errResponse *threecommas.APIError
	if errors.As(err, &errResponse) {
		// inspect errResponse.StatusCode or errResponse.ErrorPayload
	}
}
```

## Features

* Full access to 3Commas REST API via typed methods
* Built-in RSA request signing
* Configurable rate limiting based on subscription tier (Starter/Pro/Expert)
* Automatic 429 handling with backoff and retry
* Proper error parsing with descriptive messages
* Support for typed request/response structs
* Functional options pattern for clean, flexible configuration
* VCR-based integration tests using `go-vcr`

## Testing

```bash
go test ./threecommas -v
```

Recorded interactions use [go-vcr](https://github.com/dnaeon/go-vcr) and are stored under `testdata/`.

## Contributing

Pull requests are welcome! Contributions that improve test coverage, fix bugs, or expand functionality are especially appreciated.

## License

This project is licensed under the [Apache 2.0](LICENSE).
