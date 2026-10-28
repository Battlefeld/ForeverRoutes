package httpdownload

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

type brokenBody struct{ closed bool }

func (*brokenBody) Read([]byte) (int, error) { return 0, io.ErrUnexpectedEOF }
func (b *brokenBody) Close() error           { b.closed = true; return nil }
func response(status int, body io.ReadCloser) *http.Response {
	return &http.Response{StatusCode: status, Header: make(http.Header), Body: body}
}
func noWait(context.Context, time.Duration) error { return nil }

func TestRetryBodyFailure(t *testing.T) {
	calls := 0
	broken := &brokenBody{}
	client := &http.Client{Transport: roundTrip(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.Method != "GET" || r.Header.Get("User-Agent") == "" {
			t.Fatal("request identity missing")
		}
		if calls == 1 {
			return response(200, broken), nil
		}
		return response(200, io.NopCloser(strings.NewReader("complete"))), nil
	})}
	b, status, err := get(context.Background(), client, "https://source.test/file", 100, noWait)
	if err != nil || status != 200 || string(b) != "complete" || calls != 2 || !broken.closed {
		t.Fatalf("body=%q status=%d err=%v calls=%d closed=%v", b, status, err, calls, broken.closed)
	}
}
func TestTransientStatusRetryAndBound(t *testing.T) {
	for _, status := range []int{408, 429, 500, 503} {
		t.Run(http.StatusText(status), func(t *testing.T) {
			calls := 0
			client := &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
				calls++
				return response(status, io.NopCloser(strings.NewReader("busy"))), nil
			})}
			b, _, err := get(context.Background(), client, "https://source.test/file", 100, noWait)
			if err == nil || b != nil || calls != 3 {
				t.Fatalf("err=%v body=%q calls=%d", err, b, calls)
			}
		})
	}
}
func TestPermanentStatusNotRetried(t *testing.T) {
	for _, status := range []int{401, 403, 404} {
		calls := 0
		client := &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
			calls++
			return response(status, io.NopCloser(strings.NewReader("missing"))), nil
		})}
		_, got, err := get(context.Background(), client, "https://source.test/file", 100, noWait)
		if err != nil || got != status || calls != 1 {
			t.Fatalf("status=%d got=%d calls=%d err=%v", status, got, calls, err)
		}
	}
}
func TestDownloadLimit(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
		calls++
		return response(200, io.NopCloser(strings.NewReader("too long"))), nil
	})}
	b, _, err := get(context.Background(), client, "https://source.test/file", 3, noWait)
	if err == nil || b != nil || calls != 1 {
		t.Fatalf("err=%v body=%q calls=%d", err, b, calls)
	}
}
func TestCancellationDuringBackoff(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	calls := 0
	client := &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) { calls++; cancel(); return nil, context.Canceled })}
	_, _, err := get(ctx, client, "https://source.test/file", 10, noWait)
	if !errors.Is(err, context.Canceled) || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}
func TestRetryAfter(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
		calls++
		r := response(503, io.NopCloser(strings.NewReader("busy")))
		r.Header.Set("Retry-After", "7")
		return r, nil
	})}
	pauses := 0
	_, _, err := get(context.Background(), client, "https://source.test/file", 100, func(_ context.Context, d time.Duration) error {
		pauses++
		if d != 7*time.Second {
			t.Fatalf("delay=%v", d)
		}
		return nil
	})
	if err == nil || calls != 3 || pauses != 2 {
		t.Fatalf("calls=%d pauses=%d err=%v", calls, pauses, err)
	}
}
func TestTransportTimeoutThenSuccess(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: roundTrip(func(*http.Request) (*http.Response, error) {
		calls++
		if calls == 1 {
			return nil, context.DeadlineExceeded
		}
		return response(200, io.NopCloser(strings.NewReader("ok"))), nil
	})}
	b, _, err := get(context.Background(), client, "https://source.test/file", 100, noWait)
	if err != nil || string(b) != "ok" || calls != 2 {
		t.Fatalf("calls=%d body=%q err=%v", calls, b, err)
	}
}
