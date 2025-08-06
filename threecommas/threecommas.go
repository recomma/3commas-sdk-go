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

// ClientConfig is all the caller needs to supply.
type ClientConfig struct {
	BaseURL    string
	APIKey     string
	PrivatePEM []byte // PEM-encoded RSA PRIVATE KEY
}

// NewClient returns a fully-wired oapi-codegen client that
// signs every request with RSA, has a ratelimit of 100req/min
func New3CommasClient(cfg ClientConfig, opts ...ClientOption) (*ThreeCommasClient, error) {
	priv, err := parseRSAPrivate(cfg.PrivatePEM)
	if err != nil {
		return nil, err
	}

	signer := newRSASigner(cfg.APIKey, priv)

	if cfg.BaseURL == "" {
		cfg.BaseURL = "https://api.3commas.io/public/api"
	}

	opts = append(opts, WithRequestEditorFn(signer))
	opts = append(opts, WithDefaultRatelimit())

	raw, err := NewClientWithResponses(cfg.BaseURL, opts...)
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

// APIError wraps the raw ErrorResponse plus the HTTP status code.
type APIError struct {
	StatusCode   int
	ErrorPayload *ErrorResponse
}

// Error implements the error interface.
func (e *APIError) Error() string {
	// You can customize this however you like.
	return fmt.Sprintf("API error %d: %s", e.StatusCode, *e.ErrorPayload.ErrorDescription)
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
