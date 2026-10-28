// Package httpdownload downloads pinned public source files. A response is only
// returned after the entire bounded body has been read successfully.
package httpdownload

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strconv"
	"time"
)

const attempts = 3

// Get retries transient GET failures, including timeouts while reading the body.
// The caller validates source-specific status codes and content before caching.
func Get(ctx context.Context, client *http.Client, url string, limit int64) ([]byte, int, error) {
	return get(ctx, client, url, limit, wait)
}

func wait(ctx context.Context, d time.Duration) error {
	timer := time.NewTimer(d)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func get(ctx context.Context, client *http.Client, url string, limit int64, pause func(context.Context, time.Duration) error) ([]byte, int, error) {
	var last error
	for attempt := 1; attempt <= attempts; attempt++ {
		if err := ctx.Err(); err != nil {
			return nil, 0, err
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
		if err != nil {
			return nil, 0, err
		}
		req.Header.Set("User-Agent", "ForeverRoutes (https://github.com/Battlefeld/ForeverRoutes)")
		resp, err := client.Do(req)
		retry := false
		delay := time.Duration(attempt) * time.Second
		status := 0
		if err != nil {
			last = err
			var ne net.Error
			retry = errors.As(err, &ne) || errors.Is(err, io.EOF) || errors.Is(err, io.ErrUnexpectedEOF)
		} else {
			status = resp.StatusCode
			body, readErr := io.ReadAll(io.LimitReader(resp.Body, limit+1))
			resp.Body.Close()
			if readErr != nil {
				last, retry = readErr, true
			} else if int64(len(body)) > limit {
				return nil, status, fmt.Errorf("download %s: response exceeds %d bytes", url, limit)
			} else if status == http.StatusRequestTimeout || status == http.StatusTooManyRequests || status >= 500 && status <= 599 {
				last, retry = fmt.Errorf("HTTP %d", status), true
			} else {
				return body, status, nil
			}
			// Do not retry earlier than the server requests, or wait indefinitely.
			if value := resp.Header.Get("Retry-After"); value != "" {
				if seconds, e := strconv.Atoi(value); e == nil && seconds >= 0 {
					if d := time.Duration(seconds) * time.Second; d > delay {
						delay = d
					}
				} else if when, e := http.ParseTime(value); e == nil && time.Until(when) > delay {
					delay = time.Until(when)
				}
				if delay > 30*time.Second {
					retry = false
				}
			}
		}
		if err := ctx.Err(); err != nil {
			return nil, status, err
		}
		if !retry || attempt == attempts {
			return nil, status, fmt.Errorf("download %s (attempt %d/%d): %w", url, attempt, attempts, last)
		}
		if err := pause(ctx, delay); err != nil {
			return nil, status, err
		}
	}
	panic("unreachable")
}
