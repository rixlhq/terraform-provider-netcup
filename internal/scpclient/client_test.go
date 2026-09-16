package scpclient_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/rixlhq/terraform-provider-netcup/internal/scpclient"
)

func TestDoSetsContentTypeByMethod(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		method string
		body   []byte
		want   string
	}{
		"POST without body":   {method: http.MethodPost, body: nil, want: "application/json"},
		"POST with body":      {method: http.MethodPost, body: []byte(`{"a":1}`), want: "application/json"},
		"PUT without body":    {method: http.MethodPut, body: nil, want: "application/json"},
		"PUT with body":       {method: http.MethodPut, body: []byte(`{"a":1}`), want: "application/json"},
		"DELETE without body": {method: http.MethodDelete, body: nil, want: "application/json"},
		"PATCH without body":  {method: http.MethodPatch, body: nil, want: "application/merge-patch+json"},
		"PATCH with body":     {method: http.MethodPatch, body: []byte(`{"a":1}`), want: "application/merge-patch+json"},
		"GET has no body":     {method: http.MethodGet, body: nil, want: ""},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			var got string
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got = r.Header.Get("Content-Type")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{}`))
			}))
			defer srv.Close()

			c := scpclient.New("token", "", srv.URL, srv.Client())
			if err := sendRequest(t, c, tc.method, tc.body); err != nil {
				t.Fatalf("%s request failed: %v", tc.method, err)
			}
			if got != tc.want {
				t.Fatalf("Content-Type = %q, want %q", got, tc.want)
			}
		})
	}
}

func sendRequest(t *testing.T, c *scpclient.Client, method string, body []byte) error {
	t.Helper()

	ctx := context.Background()
	const path = "/api/v1/test"

	var err error
	switch method {
	case http.MethodGet:
		_, err = c.Get(ctx, path, nil)
	case http.MethodPost:
		_, err = c.Post(ctx, path, body)
	case http.MethodPut:
		_, err = c.Put(ctx, path, body)
	case http.MethodPatch:
		_, err = c.Patch(ctx, path, body)
	case http.MethodDelete:
		_, err = c.Delete(ctx, path)
	default:
		t.Fatalf("unhandled method %q", method)
	}
	return err
}
