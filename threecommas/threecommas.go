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
// signs every request with RSA
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

// TradesOfDeal is a thin wrapper around GetTradesOfDealWithResponse that
// returns the slice of MarketOrder on 200 OK, or an error otherwise.
func (c *ThreeCommasClient) TradesOfDeal(
	ctx context.Context,
	dealId DealPathId,
) ([]MarketOrder, error) {
	resp, err := c.GetTradesOfDealWithResponse(ctx, dealId)
	if err != nil {
		return nil, fmt.Errorf("could not list trades for deal %d: %w", dealId, err)
	}

	if resp.StatusCode() != http.StatusOK {
		if err := GetErrorFromResponse(resp); err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode())
	}

	// should never be nil on 200, but guard just in case
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("no trades returned for deal %d", dealId)
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

func GetErrorFromResponse(v APIErrorResponses) error {
	var payload *ErrorResponse

	switch v.StatusCode() {
	case 400:
		payload = v.GetJSON400()
	case 401:
		payload = v.GetJSON401()
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
