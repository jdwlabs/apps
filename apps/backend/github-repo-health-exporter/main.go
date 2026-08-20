package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
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

// splitList parses a comma-separated env var, dropping blanks so a trailing
// comma or an all-whitespace value reads as "unset" rather than one empty
// item.
func splitList(v string) []string {
	var out []string
	for _, part := range strings.Split(v, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

// defaultRepos is the four-repo org this exporter's metric surface is
// defined over. Overridable via REPOS for local iteration against a fork,
// but production carries no reason to narrow it — the whole point is
// covering every repo at once.
var defaultRepos = []string{"apps", "platform", "infrastructure", "deployments"}

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(log)

	key, err := parseRSAPrivateKey([]byte(mustEnv("GITHUB_APP_PRIVATE_KEY")))
	if err != nil {
		log.Error("parsing GITHUB_APP_PRIVATE_KEY", "err", err)
		os.Exit(1)
	}
	gh := NewGitHubClient(
		env("GITHUB_API", "https://api.github.com"),
		mustEnv("GITHUB_APP_ID"),
		mustEnv("GITHUB_APP_INSTALLATION_ID"),
		key,
		&http.Client{Timeout: 20 * time.Second},
	)

	owner := env("GITHUB_ORG", "jdwlabs")
	repos := splitList(env("REPOS", ""))
	if len(repos) == 0 {
		repos = defaultRepos
	}

	gauges := newCIHealthGauges()
	interval := envDuration("REFRESH_INTERVAL", 15*time.Minute)

	ctx, stopRefresh := context.WithCancel(context.Background())
	go runRefreshLoop(ctx, gh, owner, repos, gauges, interval, log)

	srv := &http.Server{
		Addr:              ":" + env("PORT", "9090"),
		Handler:           newRouter(gauges),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Info("exporter listening", "addr", srv.Addr, "repos", repos, "interval", interval)
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

// runRefreshLoop polls every repo once at startup (so /metrics has data
// before the first interval elapses, rather than reading empty for up to 15
// minutes after a rollout) and then on interval until ctx is cancelled.
func runRefreshLoop(ctx context.Context, gh *GitHubClient, owner string, repos []string, gauges *ciHealthGauges, interval time.Duration, log *slog.Logger) {
	refreshAll(ctx, gh, owner, repos, gauges, log)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			refreshAll(ctx, gh, owner, repos, gauges, log)
		}
	}
}

// refreshAll fetches and recomputes stats per repo independently: one repo's
// API failure (rate limit, transient 5xx, a renamed repo) must not blank out
// the other three repos' already-known-good series. gauges.set only replaces
// the failed repo's own keys when it is actually called, so a failure here
// simply skips that repo for this cycle and its previous values age in place.
func refreshAll(ctx context.Context, gh *GitHubClient, owner string, repos []string, gauges *ciHealthGauges, log *slog.Logger) {
	touched := false
	for _, repo := range repos {
		runs, err := gh.ListMainRuns(ctx, owner+"/"+repo)
		if err != nil {
			gauges.refreshesFailed.Add(1)
			log.Error("refresh failed", "repo", repo, "err", err)
			continue
		}
		gauges.set(repo, computeCIHealth(runs))
		gauges.refreshesOK.Add(1)
		touched = true
	}
	if touched {
		gauges.lastRefreshUnix.Store(time.Now().Unix())
	}
}

func newRouter(gauges *ciHealthGauges) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mux.HandleFunc("/metrics", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; version=0.0.4")
		gauges.writeTo(w)
	})
	return mux
}
