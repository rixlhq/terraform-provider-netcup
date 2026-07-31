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

		bodyBytes, status, err := c.do(ctx, method, path, query, body)
		if err != nil {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			tflog.Warn(ctx, "SCP request failed, will retry", map[string]any{
				logKeyMethod:  method,
				logKeyPath:    path,
				logKeyError:   err.Error(),
				logKeyAttempt: attempt + 1,
			})
			continue
		}

		res, retry, err := c.handleResult(ctx, method, path, bodyBytes, status)
		if err != nil {
			return nil, err
		}
		if retry {
			continue
		}
		return res, nil
	}

	return nil, fmt.Errorf("scp request %s %s failed after %d attempts", method, path, maxRetries)
}

func (c *Client) handleResult(ctx context.Context, method, path string, body []byte, status int) ([]byte, bool, error) {
	switch {
	case status == http.StatusAccepted:
		tflog.Debug(ctx, "SCP request returned 202, waiting for task", map[string]any{
			logKeyMethod: method,
			logKeyPath:   path,
		})
		if err := c.waitForTask(ctx, body); err != nil {
			return nil, false, err
		}
		// Return an empty body so callers perform a read-back.
		return nil, false, nil
	case status == http.StatusUnauthorized && c.refreshToken != "":
		if err := c.refresh(ctx); err != nil {
			return nil, false, fmt.Errorf("token refresh failed: %w", err)
		}
		return nil, true, nil
	case status >= 500:
		tflog.Warn(ctx, "SCP request returned server error, will retry", map[string]any{
			logKeyMethod: method,
			logKeyPath:   path,
			logKeyStatus: status,
		})
		return nil, true, nil
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
