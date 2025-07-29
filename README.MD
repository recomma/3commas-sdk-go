# 3Commas Go SDK

This is a **community-maintained Go SDK** for the [3Commas API](https://developers.3commas.io), generated from an OpenAPI 3.0 specification. It enables access to 3Commas trading bots, deals, market data, and account-level features in idiomatic Go.

> **Disclaimer:** This SDK is provided "as is" with no guarantee of correctness or completeness. It is a best-effort tool built on top of the public 3Commas API documentation. Use at your own risk.

## Installation

```bash
go get github.com/terwey/3commas-sdk-go/threecommas
```

## Usage

```go
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/terwey/3commas-sdk-go/threecommas"
)

func main() {
	privateKey, _ := os.ReadFile("private.pem")

	client, err := threecommas.New3CommasClient(threecommas.ClientConfig{
		APIKey:     "your-api-key",
		PrivatePEM: privateKey,
	})
	if err != nil {
		log.Fatal(err)
	}

	resp, err := client.ListBotsWithResponse(context.Background(),
        &threecommas.ListBotsParams{})
	if err != nil {
		panic(err)
	}

	for _, bot := range *resp.JSON200 {
		fmt.Println("Bot ID:", bot.Id, "Name:", *bot.Name)
	}
}
```

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
* Proper error parsing with descriptive messages
* Support for typed request/response structs
* VCR-based integration tests using `go-vcr`

## Testing

```bash
go test ./threecommas -v
```

Recorded interactions use [go-vcr](https://github.com/dnaeon/go-vcr) and are stored under `testdata/`.

## Contributing

Pull requests are welcome! Contributions that improve test coverage, fix bugs, or expand functionality are especially appreciated.

## License

This project is licensed under the [MIT License](LICENSE).
