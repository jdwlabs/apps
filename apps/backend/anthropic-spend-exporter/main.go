package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
	"time"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
	}
	return fallback
}

func mustEnv(key string) string {
	v := os.Getenv(key)
	if v == "" {
		slog.Error("missing required env var", "key", key)
		os.Exit(1)
	}
	return v
}

func parseCycleDay(s string) (int, error) {
	day, err := strconv.Atoi(s)
	if err != nil || day < 1 || day > 31 {
		return 0, fmt.Errorf("BILLING_CYCLE_DAY must be a whole number from 1 to 31, got %q", s)
	}
	return day, nil
}

func parseBudget(s string) (float64, error) {
	usd, err := strconv.ParseFloat(s, 64)
	if err != nil || math.IsNaN(usd) || math.IsInf(usd, 0) || usd <= 0 {
		return 0, fmt.Errorf("CREDIT_BUDGET_USD must be a positive number, got %q", s)
	}
	return usd, nil
}

// refresher owns everything one polling cycle touches. Only the refresh
// loop's goroutine uses it, so names needs no lock.
type refresher struct {
	client   *AnthropicClient
	view     *spendView
	cycleDay int
	names    map[string]string
	now      func() time.Time
	log      *slog.Logger
}

// workspaceLabel turns the cost report's workspace ID into what the
// dashboard shows. An ID with no known name is exported as the ID, so spend
// in a workspace created since the last successful listing is still
// attributed to something a person can look up.
func (r *refresher) workspaceLabel(id string) string {
	if id == "" {
		return "default"
	}
	if name := r.names[id]; name != "" {
		return name
	}
	return id
}

func (r *refresher) refreshOnce(ctx context.Context) {
	start, end := cycleWindow(r.now(), r.cycleDay)

	// Names are decoration: losing them must not cost the dollar figures, so
	// a failed listing keeps the previous map and the refresh carries on.
	if names, err := r.client.WorkspaceNames(ctx); err != nil {
		r.log.Warn("listing workspaces failed; keeping previous names", "err", err)
	} else {
		r.names = names
	}

	byID, err := r.client.CycleCosts(ctx, start)
	if err != nil {
		r.view.refreshesFailed.Add(1)
		r.log.Error("refresh failed", "err", err)
		return
	}

	costs := make(map[costSeries]float64, len(byID))
	for series, usd := range byID {
		series.workspace = r.workspaceLabel(series.workspace)
		// += because two workspaces can share a display name.
		costs[series] += usd
	}
	r.view.replace(costs, start, end)
	r.view.refreshesOK.Add(1)
	r.view.lastRefreshUnix.Store(r.now().Unix())
}

// run refreshes once at startup, so /metrics has data before the first
// interval elapses, and then on interval until ctx is cancelled.
func (r *refresher) run(ctx context.Context, interval time.Duration) {
	r.refreshOnce(ctx)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			r.refreshOnce(ctx)
		}
	}
}

func newRouter(view *spendView) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		view.writeTo(w)
	})
	return mux
}

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(log)

	cycleDay, err := parseCycleDay(mustEnv("BILLING_CYCLE_DAY"))
	if err != nil {
		log.Error("invalid configuration", "err", err)
		os.Exit(1)
	}
	budget, err := parseBudget(env("CREDIT_BUDGET_USD", "100"))
	if err != nil {
		log.Error("invalid configuration", "err", err)
		os.Exit(1)
	}

	view := newSpendView(budget)
	r := &refresher{
		client: NewAnthropicClient(
			env("ANTHROPIC_API", "https://api.anthropic.com"),
			mustEnv("ANTHROPIC_ADMIN_KEY"),
			&http.Client{Timeout: 30 * time.Second},
		),
		view:     view,
		cycleDay: cycleDay,
		names:    map[string]string{},
		now:      time.Now,
		log:      log,
	}
	interval := envDuration("REFRESH_INTERVAL", 5*time.Minute)

	ctx, stopRefresh := context.WithCancel(context.Background())
	go r.run(ctx, interval)

	srv := &http.Server{
		Addr:              ":" + env("PORT", "9090"),
		Handler:           newRouter(view),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Info("exporter listening", "addr", srv.Addr, "cycle_day", cycleDay, "budget_usd", budget, "interval", interval)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Info("shutting down")

	stopRefresh()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(shutdownCtx)
	log.Info("shutdown complete")
}
