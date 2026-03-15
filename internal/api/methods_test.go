package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

func TestSessionCreateMapping(t *testing.T) {
	testMethodPathBody(t, func(c *Client, ctx context.Context) (any, error) {
		return c.SessionCreate(ctx, "k", map[string]any{"session": map[string]any{"initial_url": "https://example.com"}})
	}, http.MethodPost, "/v1/sessions", func(_ *http.Request, body map[string]any) {
		session := body["session"].(map[string]any)
		if session["initial_url"] != "https://example.com" {
			t.Fatalf("unexpected body: %#v", body)
		}
	})
}

func TestSessionListMapping(t *testing.T) {
	testMethodPathBody(t, func(c *Client, ctx context.Context) (any, error) {
		q := url.Values{}
		q.Set("page", "2")
		q.Set("limit", "20")
		return c.SessionList(ctx, "k", q)
	}, http.MethodGet, "/v1/sessions", func(r *http.Request, _ map[string]any) {
		if got := r.URL.Query().Get("page"); got != "2" {
			t.Fatalf("unexpected page query: %s", got)
		}
		if got := r.URL.Query().Get("limit"); got != "20" {
			t.Fatalf("unexpected limit query: %s", got)
		}
	})
}

func TestAgentRunMapping(t *testing.T) {
	testMethodPathBody(t, func(c *Client, ctx context.Context) (any, error) {
		return c.AgentRun(ctx, "k", "sess-1", map[string]any{"prompt": "hello"})
	}, http.MethodPost, "/v1/tools/perform-web-task", func(r *http.Request, body map[string]any) {
		if got := r.URL.Query().Get("sessionId"); got != "sess-1" {
			t.Fatalf("unexpected sessionId: %s", got)
		}
		if body["prompt"] != "hello" {
			t.Fatalf("unexpected prompt: %#v", body)
		}
	})
}

func TestTaskRunMapping(t *testing.T) {
	testMethodPathBody(t, func(c *Client, ctx context.Context) (any, error) {
		return c.TaskRun(ctx, "k", "task-1", map[string]any{"input_params": map[string]string{"A": "B"}})
	}, http.MethodPost, "/v2/tasks/task-1/run", func(_ *http.Request, body map[string]any) {
		input := body["input_params"].(map[string]any)
		if input["A"] != "B" {
			t.Fatalf("unexpected input_params: %#v", body)
		}
	})
}

func TestIdentityCredentialsMapping(t *testing.T) {
	testMethodPathBody(t, func(c *Client, ctx context.Context) (any, error) {
		return c.IdentityCredentials(ctx, "k", "ident-1")
	}, http.MethodGet, "/v1/identities/ident-1/credentials", func(_ *http.Request, _ map[string]any) {})
}

func TestApplicationListMapping(t *testing.T) {
	testMethodPathBody(t, func(c *Client, ctx context.Context) (any, error) {
		q := url.Values{}
		q.Set("search", "netsweet.co")
		return c.ApplicationList(ctx, "k", q)
	}, http.MethodGet, "/v1/applications", func(r *http.Request, _ map[string]any) {
		if got := r.URL.Query().Get("search"); got != "netsweet.co" {
			t.Fatalf("unexpected search query: %s", got)
		}
	})
}

func TestApplicationListIdentitiesMapping(t *testing.T) {
	testMethodPathBody(t, func(c *Client, ctx context.Context) (any, error) {
		q := url.Values{}
		q.Set("page", "2")
		q.Set("limit", "10")
		return c.ApplicationListIdentities(ctx, "k", "app-1", q)
	}, http.MethodGet, "/v1/applications/app-1/identities", func(r *http.Request, _ map[string]any) {
		if got := r.URL.Query().Get("page"); got != "2" {
			t.Fatalf("unexpected page query: %s", got)
		}
		if got := r.URL.Query().Get("limit"); got != "10" {
			t.Fatalf("unexpected limit query: %s", got)
		}
	})
}

func TestApplicationCreateMapping(t *testing.T) {
	testMethodPathBody(t, func(c *Client, ctx context.Context) (any, error) {
		return c.ApplicationCreate(ctx, "k", map[string]any{"source": "https://netsweet.co"})
	}, http.MethodPost, "/v1/applications", func(_ *http.Request, body map[string]any) {
		if got := body["source"]; got != "https://netsweet.co" {
			t.Fatalf("unexpected source: %#v", got)
		}
	})
}

func TestApplicationListAuthFlowsMapping(t *testing.T) {
	testMethodPathBody(t, func(c *Client, ctx context.Context) (any, error) {
		return c.ApplicationListAuthFlows(ctx, "k", "app-1")
	}, http.MethodGet, "/v1/applications/app-1/auth-flows", func(_ *http.Request, _ map[string]any) {})
}

func TestApplicationCreateTokenMapping(t *testing.T) {
	testMethodPathBody(t, func(c *Client, ctx context.Context) (any, error) {
		return c.ApplicationCreateToken(ctx, "k", "app-1", map[string]any{})
	}, http.MethodPost, "/v1/applications/app-1/tokens", func(_ *http.Request, _ map[string]any) {})
}

func testMethodPathBody(t *testing.T, run func(c *Client, ctx context.Context) (any, error), wantMethod, wantPath string, assertFn func(r *http.Request, body map[string]any)) {
	t.Helper()

	var gotMethod, gotPath, gotHeader string
	var gotQuery string
	var gotBody map[string]any

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod = r.Method
		gotPath = r.URL.Path
		gotQuery = r.URL.RawQuery
		gotHeader = r.Header.Get("anchor-api-key")
		if strings.Contains(r.Header.Get("content-type"), "application/json") {
			_ = json.NewDecoder(r.Body).Decode(&gotBody)
		}
		w.Header().Set("content-type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer srv.Close()

	client := New(Options{BaseURL: srv.URL})
	_, err := run(client, context.Background())
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}

	if gotMethod != wantMethod {
		t.Fatalf("unexpected method: got %s want %s", gotMethod, wantMethod)
	}
	if gotPath != wantPath {
		t.Fatalf("unexpected path: got %s want %s (query=%s)", gotPath, wantPath, gotQuery)
	}
	if gotHeader != "k" {
		t.Fatalf("missing API key header")
	}
	assertFn(&http.Request{URL: &url.URL{RawQuery: gotQuery}}, gotBody)
}
