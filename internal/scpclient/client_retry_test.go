package scpclient_test

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/rixlhq/terraform-provider-netcup/internal/scpclient"
)

type retryTransport func(*http.Request) (*http.Response, error)

func (f retryTransport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type failedBody struct{}

func (failedBody) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (failedBody) Close() error             { return nil }

func TestActionsAreNotRetriedAfterAmbiguousFailure(t *testing.T) {
	t.Parallel()
	for _, method := range []string{http.MethodPost, http.MethodPatch} {
		for _, failure := range []string{"transport", "response body", "server error"} {
			t.Run(method+"/"+failure, func(t *testing.T) {
				t.Parallel()
				var calls atomic.Int64
				transport := retryTransport(func(_ *http.Request) (*http.Response, error) {
					calls.Add(1)
					switch failure {
					case "transport":
						return nil, io.ErrUnexpectedEOF
					case "response body":
						return &http.Response{StatusCode: http.StatusOK, Body: failedBody{}}, nil
					default:
						return retryResponse(http.StatusInternalServerError, `{"error":"backend failure"}`), nil
					}
				})
				c := scpclient.New("token", "", "https://scp.invalid", &http.Client{Transport: transport})
				err := sendRequest(t, c, method, []byte(`{}`))
				if err == nil {
					t.Fatal("expected failure")
				}
				if got := calls.Load(); got != 1 {
					t.Fatalf("executed action %d times, want exactly one", got)
				}
				if failure == "server error" {
					var apiErr *scpclient.APIError
					if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusInternalServerError {
						t.Fatalf("server diagnostic lost: %v", err)
					}
				}
			})
		}
	}
}

func TestIdempotentRequestsStillRetry(t *testing.T) {
	t.Parallel()
	for _, method := range []string{http.MethodGet, http.MethodPut, http.MethodDelete} {
		t.Run(method, func(t *testing.T) {
			t.Parallel()
			var calls atomic.Int64
			c := scpclient.New("token", "", "https://scp.invalid", &http.Client{Transport: retryTransport(func(_ *http.Request) (*http.Response, error) {
				if calls.Add(1) == 1 {
					return retryResponse(http.StatusServiceUnavailable, `{"error":"busy"}`), nil
				}
				return retryResponse(http.StatusOK, `{}`), nil
			})})
			if err := sendRequest(t, c, method, nil); err != nil {
				t.Fatal(err)
			}
			if calls.Load() != 2 {
				t.Fatalf("requests=%d, want two", calls.Load())
			}
		})
	}
}

func TestConcurrentUnauthorizedRequestsRefreshOnce(t *testing.T) {
	t.Parallel()
	const requests = 8
	var denied, refreshes atomic.Int64
	allDenied := make(chan struct{})
	transport := retryTransport(func(r *http.Request) (*http.Response, error) {
		if r.URL.Path == "/realms/scp/protocol/openid-connect/token" {
			refreshes.Add(1)
			return retryResponse(http.StatusOK, `{"access_token":"new-token","refresh_token":"rotated-token"}`), nil
		}
		if r.Header.Get("Authorization") == "Bearer old-token" {
			if denied.Add(1) == requests {
				close(allDenied)
			}
			select {
			case <-allDenied:
			case <-r.Context().Done():
				return nil, r.Context().Err()
			}
			return retryResponse(http.StatusUnauthorized, `{}`), nil
		}
		if r.Header.Get("Authorization") != "Bearer new-token" {
			t.Errorf("unexpected bearer token")
		}
		return retryResponse(http.StatusOK, `{}`), nil
	})
	c := scpclient.New("old-token", "refresh-token", "https://scp.invalid", &http.Client{Transport: transport})
	var workers sync.WaitGroup
	for range requests {
		workers.Go(func() {
			if _, err := c.Post(t.Context(), "/api/v1/action", nil); err != nil {
				t.Error(err)
			}
		})
	}
	workers.Wait()
	if refreshes.Load() != 1 {
		t.Fatalf("refresh requests=%d, want one for the rejected token", refreshes.Load())
	}
}

func TestRefreshRejectsMissingAccessToken(t *testing.T) {
	t.Parallel()
	var requests atomic.Int64
	c := scpclient.New("expired", "refresh", "https://scp.invalid", &http.Client{Transport: retryTransport(func(r *http.Request) (*http.Response, error) {
		if strings.Contains(r.URL.Path, "/protocol/openid-connect/token") {
			return retryResponse(http.StatusOK, `{}`), nil
		}
		requests.Add(1)
		return retryResponse(http.StatusUnauthorized, `{}`), nil
	})})
	_, err := c.Get(t.Context(), "/api/v1/servers", nil)
	if err == nil || !strings.Contains(err.Error(), "access_token") {
		t.Fatalf("expected invalid token response error, got %v", err)
	}
	if requests.Load() != 1 {
		t.Fatalf("sent %d API requests after invalid refresh", requests.Load())
	}
}

func retryResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), Header: make(http.Header)}
}
