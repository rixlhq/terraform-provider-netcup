package scpclient_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/rixlhq/terraform-provider-netcup/internal/scpclient"
)

const jsonContentType = "application/json"

func TestDoSetsContentTypeByMethod(t *testing.T) {
	t.Parallel()

	tests := map[string]struct {
		method string
		body   []byte
		want   string
	}{
		"POST without body":     {method: http.MethodPost, body: nil, want: jsonContentType},
		"POST with empty body":  {method: http.MethodPost, body: []byte{}, want: jsonContentType},
		"POST with body":        {method: http.MethodPost, body: []byte(`{"a":1}`), want: jsonContentType},
		"PUT without body":      {method: http.MethodPut, body: nil, want: jsonContentType},
		"PUT with empty body":   {method: http.MethodPut, body: []byte{}, want: jsonContentType},
		"PUT with body":         {method: http.MethodPut, body: []byte(`{"a":1}`), want: jsonContentType},
		"DELETE without body":   {method: http.MethodDelete, body: nil, want: jsonContentType},
		"PATCH without body":    {method: http.MethodPatch, body: nil, want: "application/merge-patch+json"},
		"PATCH with empty body": {method: http.MethodPatch, body: []byte{}, want: "application/merge-patch+json"},
		"PATCH with body":       {method: http.MethodPatch, body: []byte(`{"a":1}`), want: "application/merge-patch+json"},
		"GET has no body":       {method: http.MethodGet, body: nil, want: ""},
	}

	for name, tc := range tests {
		t.Run(name, func(t *testing.T) {
			t.Parallel()

			checkRequest(t, tc.method, tc.body, tc.want)
		})
	}
}

func checkRequest(t *testing.T, method string, body []byte, want string) {
	t.Helper()
	var got string
	var gotBody []byte
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = r.Header.Get("Content-Type")
		var err error
		gotBody, err = io.ReadAll(r.Body)
		if err != nil {
			t.Errorf("read request body: %v", err)
		}
		w.Header().Set("Content-Type", jsonContentType)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	c := scpclient.New("token", "", srv.URL, srv.Client())
	if err := sendRequest(t, c, method, body); err != nil {
		t.Fatalf("%s request failed: %v", method, err)
	}
	if got != want {
		t.Fatalf("Content-Type = %q, want %q", got, want)
	}
	if !bytes.Equal(gotBody, body) {
		t.Fatalf("request body = %q, want %q", gotBody, body)
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

var apiErrorTests = []struct {
	name       string
	err        scpclient.APIError
	wantSubstr []string
}{
	{
		name:       "empty body",
		err:        scpclient.APIError{StatusCode: 500, Body: ""},
		wantSubstr: []string{"scp API error 500"},
	},
	{
		name:       "plain text body",
		err:        scpclient.APIError{StatusCode: 502, Body: "Bad Gateway"},
		wantSubstr: []string{"scp API error 502: Bad Gateway"},
	},
	{
		name: "validation error with field errors",
		err: scpclient.APIError{
			StatusCode: 400,
			Body:       `{"code":"server.snapshot.create.error.disk","message":"Validation failed","errors":[{"field":"diskName","message":"disk is required for offline snapshot"}]}`,
		},
		wantSubstr: []string{
			"scp API error 400",
			"Validation failed",
			"diskName: disk is required for offline snapshot",
		},
	},
	{
		name: "response error with message only",
		err: scpclient.APIError{
			StatusCode: 404,
			Body:       `{"code":"server.not.found","message":"Server not found."}`,
		},
		wantSubstr: []string{"scp API error 404", "Server not found."},
	},
}

func TestAPIError_Error(t *testing.T) {
	t.Parallel()

	for _, tc := range apiErrorTests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := tc.err.Error()
			for _, sub := range tc.wantSubstr {
				if !strings.Contains(got, sub) {
					t.Errorf("Error() = %q, want it to contain %q", got, sub)
				}
			}
		})
	}
}
