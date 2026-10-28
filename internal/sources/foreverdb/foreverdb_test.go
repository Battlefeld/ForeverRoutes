package foreverdb

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func fake(t *testing.T, total int) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("X-API-Key") != "secret" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		switch r.URL.Path {
		case "/meta":
			fmt.Fprint(w, `{"data":{"build":"1.60.1.70450","levelCap":30},"meta":{"build":"1.60.1.70450"}}`)
		case "/quests":
			off, _ := strconv.Atoi(r.URL.Query().Get("offset"))
			var rows []string
			for i := off; i < total && i < off+100; i++ {
				rows = append(rows, fmt.Sprintf(`{"id":%d,"name":"Q%d","level":5,"requiredLevel":3,"source":"forever","echo":"secret"}`, i+1, i+1))
			}
			fmt.Fprintf(w, `{"data":[%s],"meta":{"total":%d}}`, strings.Join(rows, ","), total)
		case "/changes-seen":
			fmt.Fprint(w, `{"data":{"changes":[{"kind":"quest","id":1}]},"meta":{}}`)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
}

func client(url, key string) *Client {
	return &Client{Key: key, HTTP: http.DefaultClient, Base: url, MaxCalls: 50}
}

func TestFetchPagesAndRedacts(t *testing.T) {
	srv := fake(t, 250)
	defer srv.Close()
	c := client(srv.URL, "secret")
	s, err := c.Fetch(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(s.Quests) != 250 || len(s.Changes) != 1 || s.Build() != "1.60.1.70450" {
		t.Fatalf("quests %d changes %d build %q", len(s.Quests), len(s.Changes), s.Build())
	}
	if c.Calls != 1+3+1 {
		t.Errorf("calls %d", c.Calls)
	}
	for _, q := range s.Quests {
		if strings.Contains(string(q), "secret") {
			t.Fatal("the key must never be kept")
		}
	}
	list, err := s.List()
	if err != nil || list[249].ID != 250 || *list[0].RequiredLevel != 3 {
		t.Errorf("list %v %+v", err, list[0])
	}
}

func TestBudgetAndBadKey(t *testing.T) {
	srv := fake(t, 1000)
	defer srv.Close()
	c := client(srv.URL, "secret")
	c.MaxCalls = 3
	if _, err := c.Fetch(context.Background()); !errors.Is(err, ErrQuota) {
		t.Errorf("budget: %v", err)
	}
	if _, err := client(srv.URL, "wrong").Fetch(context.Background()); err == nil || !strings.Contains(err.Error(), "refused") {
		t.Errorf("bad key: %v", err)
	}
	if _, err := NewClient(" "); err == nil {
		t.Error("an empty key must be refused")
	}
}

func TestTemporaryFailureRetriesWithinBudget(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if calls == 1 {
			w.WriteHeader(503)
			return
		}
		fmt.Fprint(w, `{"data":{"build":"ok"}}`)
	}))
	defer srv.Close()
	c := client(srv.URL, "secret")
	if _, err := c.get(context.Background(), "meta", nil); err != nil || calls != 2 || c.Calls != 2 {
		t.Fatalf("calls=%d budget=%d err=%v", calls, c.Calls, err)
	}
	calls = 0
	c = client(srv.URL, "secret")
	c.MaxCalls = 1
	if _, err := c.get(context.Background(), "meta", nil); !errors.Is(err, ErrQuota) || calls != 1 {
		t.Fatalf("budget exceeded: calls=%d err=%v", calls, err)
	}
}

func TestQuotaNotRetried(t *testing.T) {
	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { calls++; w.WriteHeader(429) }))
	defer srv.Close()
	c := client(srv.URL, "secret")
	if _, err := c.get(context.Background(), "meta", nil); !errors.Is(err, ErrQuota) || calls != 1 {
		t.Fatalf("calls=%d err=%v", calls, err)
	}
}
