package threecommas

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/require"
)

// TestWithClientOption demonstrates that users can now pass through
// ClientOptions like WithRequestEditorFn for logging/modifying requests.
func TestWithClientOption(t *testing.T) {
	var capturedRequests []*http.Request

	// Custom request editor to capture/log requests
	requestLogger := func(ctx context.Context, req *http.Request) error {
		capturedRequests = append(capturedRequests, req)
		t.Logf("Request: %s %s", req.Method, req.URL.Path)
		return nil
	}

	client, err := New3CommasClient(
		WithAPIKey("test-key"),
		WithPrivatePEM([]byte(fakeKey)),
		WithClientOption(WithRequestEditorFn(requestLogger)),
	)
	require.NoError(t, err)
	require.NotNil(t, client)

	// Verify the client was created with our custom request editor
	require.NotNil(t, client.ClientWithResponses)

	// The request editor should be stored in clientOptions
	require.Len(t, client.clientOptions, 1, "expected one client option")
}

// TestMultipleClientOptions verifies that multiple ClientOptions can be chained.
func TestMultipleClientOptions(t *testing.T) {
	var requestCount int

	editor1 := func(ctx context.Context, req *http.Request) error {
		requestCount++
		req.Header.Set("X-Custom-Header-1", "value1")
		return nil
	}

	editor2 := func(ctx context.Context, req *http.Request) error {
		requestCount++
		req.Header.Set("X-Custom-Header-2", "value2")
		return nil
	}

	client, err := New3CommasClient(
		WithAPIKey("test-key"),
		WithPrivatePEM([]byte(fakeKey)),
		WithClientOption(WithRequestEditorFn(editor1)),
		WithClientOption(WithRequestEditorFn(editor2)),
	)
	require.NoError(t, err)
	require.NotNil(t, client)

	// Verify multiple client options were stored
	require.Len(t, client.clientOptions, 2, "expected two client options")
}
