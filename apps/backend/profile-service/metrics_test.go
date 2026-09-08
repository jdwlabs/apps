package main

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
)

// injectedSegment is what a caller can put where a profile id goes — the router
// takes it as free text. Markup is what would matter if it reached a response,
// and the newline is what would matter if it reached the exposition format.
const injectedSegment = "<script>alert(1)\n"

func timedHandler(t *testing.T, handler http.Handler) (*Metrics, http.Handler) {
	t.Helper()
	router, err := NewRouter([]Route{
		{Method: http.MethodGet, Pattern: "/api/profiles/{profileId}/icon", Handler: handler},
	})
	if err != nil {
		t.Fatalf("NewRouter: %v", err)
	}
	metrics := NewMetrics()
	return metrics, metrics.Middleware(router, router)
}

func scrape(t *testing.T, metrics *Metrics) string {
	t.Helper()
	recorder := httptest.NewRecorder()
	metrics.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/actuator/prometheus", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("scrape = %d, want 200", recorder.Code)
	}
	return recorder.Body.String()
}

func TestAPathSegmentACallerControlsNeverReachesTheExportedSeries(t *testing.T) {
	// The uri label is the route pattern, not the path, so neither the series
	// count nor the scrape body is anyone else's to write.
	metrics, handler := timedHandler(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	escaped := url.PathEscape(injectedSegment)
	for _, path := range []string{
		"/api/profiles/" + escaped + "/icon",
		"/api/profiles/" + escaped + "/there-is-no-such-route",
	} {
		request := httptest.NewRequest(http.MethodGet, path, nil)
		if !strings.Contains(request.URL.Path, injectedSegment) {
			t.Fatalf("path = %q, want the segment to reach the router undecoded", request.URL.Path)
		}
		handler.ServeHTTP(httptest.NewRecorder(), request)
	}

	exported := scrape(t, metrics)

	if strings.Contains(exported, injectedSegment) || strings.Contains(exported, "<script>") {
		t.Error("a caller-supplied path segment was exported as a label value")
	}
	if !strings.Contains(exported, `uri="/api/profiles/{profileId}/icon"`) {
		t.Error("the matched request was not labelled by its route pattern")
	}
	if !strings.Contains(exported, `uri="`+unmatchedURI+`"`) {
		t.Error("the unmatched request was not collapsed onto one series")
	}
}

func TestTimingARequestLeavesTheResponseExactlyAsTheHandlerWroteIt(t *testing.T) {
	// The wrapper exists to read the status. Anything it did to the body would
	// be a second, untyped copy of a response on a path with no reason to see
	// one — which is how a reflected value becomes a finding.
	body := `{"displayName":"<script>alert(1)</script>"}`
	metrics, handler := timedHandler(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		if _, err := w.Write([]byte(body)); err != nil {
			t.Errorf("write: %v", err)
		}
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/profiles/abc/icon", nil))

	if recorder.Code != http.StatusCreated {
		t.Errorf("status = %d, want 201", recorder.Code)
	}
	if recorder.Body.String() != body {
		t.Errorf("body = %q, want the handler's own %q", recorder.Body.String(), body)
	}
	if recorder.Header().Get("Content-Type") != "application/json" {
		t.Errorf("content type = %q, want the one the handler set", recorder.Header().Get("Content-Type"))
	}
	if exported := scrape(t, metrics); !strings.Contains(exported, `status="201"`) {
		t.Error("the status the handler set was not the one recorded")
	}
}

func TestAHandlerThatWritesWithoutAStatusIsRecordedAsTheTwoHundredItSends(t *testing.T) {
	// This is what lets the wrapper leave Write alone: the status the standard
	// library sends for an unannounced body is the one the field already holds.
	metrics, handler := timedHandler(t, http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if _, err := w.Write([]byte(`[]`)); err != nil {
			t.Errorf("write: %v", err)
		}
	}))

	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/profiles/abc/icon", nil))

	if recorder.Code != http.StatusOK {
		t.Errorf("status = %d, want the implicit 200", recorder.Code)
	}
	exported := scrape(t, metrics)
	if !strings.Contains(exported, `status="200"`) || !strings.Contains(exported, `outcome="SUCCESS"`) {
		t.Error("an unannounced 200 was not recorded as one")
	}
}
