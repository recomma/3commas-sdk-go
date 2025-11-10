package threecommas

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"net/http"
	"sort"
	"strings"
)

type clientConfig struct {
	baseURL    string
	apiKey     string
	privatePEM []byte
	planTier   PlanTier
	httpClient HttpRequestDoer
}

// ThreeCommasClientOption configures the 3commas client.
type ThreeCommasClientOption func(*clientConfig)

// WithAPIKey sets the API key for authentication.
func WithAPIKey(key string) ThreeCommasClientOption {
	return func(c *clientConfig) {
		c.apiKey = key
	}
}

// WithPrivatePEM sets the RSA private key for signing requests.
func WithPrivatePEM(pem []byte) ThreeCommasClientOption {
	return func(c *clientConfig) {
		c.privatePEM = pem
	}
}

// WithThreeCommasBaseURL sets the base URL for the API.
// Defaults to https://api.3commas.io/public/api
func WithThreeCommasBaseURL(url string) ThreeCommasClientOption {
	return func(c *clientConfig) {
		c.baseURL = url
	}
}

// WithPlanTier sets the subscription plan tier for rate limiting.
// Defaults to PlanExpert.
func WithPlanTier(tier PlanTier) ThreeCommasClientOption {
	return func(c *clientConfig) {
		c.planTier = tier
	}
}

// withHTTPClient is an internal option for testing
func withHTTPClient(client HttpRequestDoer) ThreeCommasClientOption {
	return func(c *clientConfig) {
		c.httpClient = client
	}
}

// New3CommasClient creates a fully-wired client with RSA signing and rate limiting.
func New3CommasClient(opts ...ThreeCommasClientOption) (*ThreeCommasClient, error) {
	cfg := &clientConfig{
		baseURL:  "https://api.3commas.io/public/api",
		planTier: PlanExpert,
	}

	for _, opt := range opts {
		opt(cfg)
	}

	if cfg.apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}
	if len(cfg.privatePEM) == 0 {
		return nil, fmt.Errorf("private key PEM is required")
	}

	priv, err := parseRSAPrivate(cfg.privatePEM)
	if err != nil {
		return nil, err
	}

	signer := newRSASigner(cfg.apiKey, priv)

	clientOpts := []ClientOption{
		WithRequestEditorFn(signer),
		WithThreeCommasRateLimits(cfg.planTier),
	}

	// If a custom HTTP client was provided (for testing), use it
	if cfg.httpClient != nil {
		clientOpts = append(clientOpts, WithHTTPClient(cfg.httpClient))
	}

	raw, err := NewClientWithResponses(cfg.baseURL, clientOpts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create API client: %w", err)
	}
	return &ThreeCommasClient{ClientWithResponses: raw}, nil
}

func parseRSAPrivate(pemBytes []byte) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode(pemBytes)
	if block == nil || !strings.Contains(block.Type, "PRIVATE KEY") {
		return nil, fmt.Errorf("invalid RSA key PEM")
	}

	// Accept either PKCS#1 or PKCS#8
	if key, err := x509.ParsePKCS1PrivateKey(block.Bytes); err == nil {
		return key, nil
	}
	key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("parse RSA key: %w", err)
	}
	rsaKey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA private key")
	}
	return rsaKey, nil
}

func newRSASigner(apiKey string, priv *rsa.PrivateKey) RequestEditorFn {
	return func(_ context.Context, req *http.Request) error {
		payload := req.URL.EscapedPath()
		if qs := sortedQuery(req); qs != "" {
			payload += "?" + qs
		}

		digest := sha256.Sum256([]byte(payload))
		rawSig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, digest[:])
		if err != nil {
			return fmt.Errorf("rsa sign: %w", err)
		}

		req.Header.Set("Apikey", apiKey)
		req.Header.Set("Signature", base64.StdEncoding.EncodeToString(rawSig))
		return nil
	}
}

func sortedQuery(r *http.Request) string {
	if r.URL.RawQuery == "" {
		return ""
	}
	parts := strings.Split(r.URL.RawQuery, "&")
	sort.Strings(parts) // 3Commas examples sort lexicographically
	return strings.Join(parts, "&")
}

type ThreeCommasClient struct {
	*ClientWithResponses
}

func (c *ThreeCommasClient) GetMarketOrdersForDeal(ctx context.Context, dealId DealPathId) ([]MarketOrder, error) {
	return c.GetTradesForDeal(ctx, dealId)
}

// GetTradesForDeal is a thin wrapper around GetTradesOfDealWithResponse that
// returns the slice of MarketOrder on 200 OK, or an error otherwise.
func (c *ThreeCommasClient) GetTradesForDeal(ctx context.Context, dealId DealPathId) ([]MarketOrder, error) {
	resp, err := c.GetTradesOfDealWithResponse(ctx, dealId)
	if err != nil {
		return nil, fmt.Errorf("request failed for deal %d: %w", dealId, err)
	}

	if err := GetErrorFromResponse(resp); err != nil {
		return nil, err
	}

	return *resp.JSON200, nil
}

// GetListOfDeals is a thin wrapper around ListDealsWithResponse that
// returns the slice of Deal on 200 OK, or an error otherwise.
func (c *ThreeCommasClient) GetListOfDeals(ctx context.Context, opts ...ListDealsParamsOption) ([]Deal, error) {
	p := ListDealsParamsFromOptions(opts...)
	resp, err := c.ListDealsWithResponse(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w, params: %v", err, p)
	}

	if err := GetErrorFromResponse(resp); err != nil {
		return nil, err
	}

	return *resp.JSON200, nil
}

// ListBots is a thin wrapper around ListBotsWithResponse that
// returns the slice of Deal on 200 OK, or an error otherwise.
func (c *ThreeCommasClient) ListBots(ctx context.Context, opts ...ListBotsParamsOption) ([]Bot, error) {
	p := ListBotsParamsFromOptions(opts...)
	resp, err := c.ListBotsWithResponse(ctx, p)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w, params: %v", err, p)
	}

	if err := GetErrorFromResponse(resp); err != nil {
		return nil, err
	}

	return *resp.JSON200, nil
}

func (c *ThreeCommasClient) GetDealForID(ctx context.Context, dealId DealPathId) (*Deal, error) {
	resp, err := c.GetDealWithResponse(ctx, dealId)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}

	if err := GetErrorFromResponse(resp); err != nil {
		return nil, err
	}

	deal := Deal(*resp.JSON200)

	return &deal, nil
}

// APIError wraps the raw ErrorResponse plus the HTTP status code.
type APIError struct {
	StatusCode   int
	ErrorPayload *ErrorResponse
}

// Error implements the error interface.
func (e *APIError) Error() string {
	if e.ErrorPayload.ErrorDescription != nil {
		return fmt.Sprintf("API error %d: %s", e.StatusCode, *e.ErrorPayload.ErrorDescription)
	}
	return fmt.Sprintf("API error %d: %s", e.StatusCode, e.ErrorPayload.Error)
}

func (e *ErrorResponse) String() string {
	var s strings.Builder
	s.WriteString("Error: ")
	s.WriteString(e.Error)
	s.WriteString("\n")
	if e.ErrorDescription != nil {
		s.WriteString("Description: ")
		s.WriteString(*e.ErrorDescription)
		s.WriteString("\n")
	}
	if e.ErrorAttributes != nil {
		s.WriteString("Attributes:\n")
		for k, v := range *e.ErrorAttributes {
			s.WriteString(" ")
			s.WriteString(k)
			s.WriteString(": ")
			s.WriteString(strings.Join(v, ", "))
			s.WriteString("\n")
		}
		s.WriteString("\n")
	}

	return s.String()
}

func GetErrorFromResponse(v APIErrorResponses) error {
	// Treat anything in the 200–299 range as OK:
	if 200 <= v.StatusCode() && v.StatusCode() <= 299 {
		return nil
	}

	var payload *ErrorResponse

	switch v.StatusCode() {
	case 400:
		payload = v.GetJSON400()
	case 401:
		payload = v.GetJSON401()
	case 404:
		payload = v.GetJSON404()
	case 418:
		payload = v.GetJSON418()
	case 429:
		payload = v.GetJSON429()
	case 500:
		payload = v.GetJSON500()
	case 504:
		payload = v.GetJSON504()
	default:
		// we don’t have an ErrorResponse type for this code
		return nil
	}

	if payload == nil {
		// status code but no JSON body → treat as non-API error
		return nil
	}
	return &APIError{v.StatusCode(), payload}
}
