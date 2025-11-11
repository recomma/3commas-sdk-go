package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/recomma/3commas-sdk-go/threecommas"
)

func main() {
	privateKey, err := os.ReadFile("private.pem")
	if err != nil {
		log.Fatal(err)
	}

	// Custom request logger
	requestLogger := func(ctx context.Context, req *http.Request) error {
		log.Printf("Making request: %s %s", req.Method, req.URL.Path)
		return nil
	}

	client, err := threecommas.New3CommasClient(
		threecommas.WithAPIKey(os.Getenv("THREECOMMAS_API_KEY")),
		threecommas.WithPrivatePEM(privateKey),
		threecommas.WithPlanTier(threecommas.PlanPro),
		// Add custom request editor for logging
		threecommas.WithClientOption(
			threecommas.WithRequestEditorFn(requestLogger),
		),
	)
	if err != nil {
		log.Fatal(err)
	}

	// List bots - the request logger will log each request
	bots, err := client.ListBots(
		context.Background(),
		threecommas.WithLimitForListBots(10),
	)
	if err != nil {
		log.Fatalf("failed to list bots: %v", err)
	}

	fmt.Printf("Found %d bots\n", len(bots))
}
