package scpclient

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

const (
	logKeyMethod  = "method"
	logKeyPath    = "path"
	logKeyStatus  = "status"
	logKeyError   = "error"
	logKeyAttempt = "attempt"
)

func (c *Client) doWithRetry(ctx context.Context, method, path string, query url.Values, body []byte) ([]byte, error) {
	backoff := initialBackoff
	var lastErr error

	for attempt := range maxRetries {
		if attempt > 0 {
			tflog.Debug(ctx, "retrying SCP request", map[string]any{
				logKeyMethod:  method,
				logKeyPath:    path,
				logKeyAttempt: attempt,
			})
			if err := backoffWait(ctx, backoff); err != nil {
				return nil, err
			}
			backoff = nextBackoff(backoff)
		}

		accessToken := c.currentAccessToken()
		bodyBytes, status, err := c.do(ctx, method, path, query, body, accessToken)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			if !retrySafe(method) {
				return nil, fmt.Errorf("scp request outcome may be uncertain; not automatically retrying %s: %w", method, err)
			}
			lastErr = err
			tflog.Warn(ctx, "SCP request failed, will retry", map[string]any{
				logKeyMethod:  method,
				logKeyPath:    path,
				logKeyError:   err.Error(),
				logKeyAttempt: attempt + 1,
			})
			continue
		}

		res, retry, err := c.handleResult(ctx, method, path, bodyBytes, status, accessToken)
		if !retry {
			return res, err
		}
		lastErr = err
	}

	return nil, fmt.Errorf("scp request %s %s failed after %d attempts: %w", method, path, maxRetries, lastErr)
}

func (c *Client) handleResult(ctx context.Context, method, path string, body []byte, status int, accessToken string) ([]byte, bool, error) {
	switch {
	case status == http.StatusAccepted:
		tflog.Debug(ctx, "SCP request returned 202, waiting for task", map[string]any{
			logKeyMethod: method,
			logKeyPath:   path,
		})
		taskResult, err := c.waitForTask(ctx, body)
		if err != nil {
			return nil, false, err
		}
		// Some endpoints return the created object in the task result.
		// Return it when present so callers can derive a stable id.
		if taskResult != nil {
			return taskResult, false, nil
		}
		// Otherwise callers perform a read-back.
		return nil, false, nil
	case status == http.StatusUnauthorized:
		retry, err := c.refresh(ctx, accessToken)
		if err != nil {
			return nil, false, fmt.Errorf("token refresh failed: %w", err)
		}
		return nil, retry, &APIError{StatusCode: status, Body: string(body)}
	case status >= 500:
		if !retrySafe(method) {
			return nil, false, &APIError{StatusCode: status, Body: string(body)}
		}
		tflog.Warn(ctx, "SCP request returned server error, will retry", map[string]any{
			logKeyMethod: method,
			logKeyPath:   path,
			logKeyStatus: status,
		})
		return nil, true, &APIError{StatusCode: status, Body: string(body)}
	case status >= 400:
		return nil, false, &APIError{StatusCode: status, Body: string(body)}
	default:
		return body, false, nil
	}
}

func backoffWait(ctx context.Context, d time.Duration) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-time.After(d):
		return nil
	}
}

func nextBackoff(current time.Duration) time.Duration {
	return time.Duration(math.Min(float64(current)*2, float64(30*time.Second)))
}

// Only methods with idempotent semantics may be replayed after an ambiguous
// transport error or server failure. POST/PATCH actions have no idempotency key.
func retrySafe(method string) bool {
	return method == http.MethodGet || method == http.MethodPut || method == http.MethodDelete
}
