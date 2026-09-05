package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
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

func envInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			return n
		}
	}
	return fallback
}

// splitList parses a comma-separated env var, dropping blanks so a trailing
// comma or an all-whitespace value reads as "unset" rather than one empty item.
func splitList(v string) []string {
	var out []string
	for _, part := range strings.Split(v, ",") {
		if part = strings.TrimSpace(part); part != "" {
			out = append(out, part)
		}
	}
	return out
}

// envDuration reads a Go duration ("6h", "45m"). An unparseable or
// non-positive value falls back rather than disabling the behaviour it
// configures — a typo must not silently turn auto-close into never-close.
func envDuration(key string, fallback time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil && d > 0 {
			return d
		}
		slog.Warn("ignoring unusable duration", "key", key, "value", v, "using", fallback.String())
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

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(log)

	// Output posts (Discord/Jira/GitHub) get a tight per-request cap. LLM-bound
	// calls (Holmes investigation, patch generation) legitimately run for
	// minutes on slow backends, so their client carries no timeout of its own —
	// the dispatcher's per-alert context (INVESTIGATION_TIMEOUT_SECONDS) is
	// their only deadline.
	hc := &http.Client{Timeout: 90 * time.Second}
	llmc := &http.Client{}

	holmes := NewHolmesClient(
		env("HOLMES_URL", "http://platform-holmes-holmes.ai-sre.svc.cluster.local"),
		env("LITELLM_MODEL", "claude-sonnet"),
		llmc,
	)
	// Names the repositories a remediation may target and bounds where it may
	// land — the same list drives both, so the model is told the destination
	// instead of guessing it and the boundary check still refuses anything
	// else. Unset disables the PR arm rather than trusting a model-named repo.
	allowedRepos := splitList(os.Getenv("GITHUB_REPO_ALLOWLIST"))
	if len(allowedRepos) == 0 {
		log.Warn("GITHUB_REPO_ALLOWLIST unset; remediation PRs are disabled")
	}
	patchGen := NewPatchGenerator(
		env("LITELLM_URL", "http://platform-litellm.ai-sre.svc.cluster.local:4000/v1"),
		mustEnv("LITELLM_KEY"),
		env("LITELLM_MODEL", "claude-sonnet"),
		0.75, allowedRepos, log, llmc,
	)
	jiraURL := mustEnv("JIRA_URL")
	discord := NewDiscordNotifier(mustEnv("DISCORD_WEBHOOK_URL"), jiraURL, hc)
	// "Task" not "Bug": the target project's type scheme has no Bug type and
	// Jira rejects creates with an unknown type (400).
	// JIRA_DONE_STATUS names the status an automatic close aims for; the Done
	// category alone also covers "Won't Do" and "Duplicate".
	jira := NewJiraClient(jiraURL, mustEnv("JIRA_USERNAME"), mustEnv("JIRA_API_TOKEN"), env("JIRA_PROJECT", "JDWLABS"), env("JIRA_ISSUE_TYPE", "Task"), hc,
		withDoneStatus(env("JIRA_DONE_STATUS", defaultDoneStatus)))
	githubAPI := env("GITHUB_API", "https://api.github.com")
	githubTokens, err := newGitHubTokenSource(githubAPI, hc)
	if err != nil {
		log.Error("github token source", "err", err)
		os.Exit(1)
	}
	// Bounds which files inside an allowed repository a patch may touch, as
	// path.Match globs. The model is told the layout, but it has no view of
	// the tree and has invented directories that nothing reconciles; the
	// write is refused unless the path is watched and already exists. Unset
	// disables the PR arm like the repo allowlist does.
	allowedPaths := splitList(os.Getenv("GITHUB_PATH_ALLOWLIST"))
	if len(allowedRepos) > 0 && len(allowedPaths) == 0 {
		log.Warn("GITHUB_PATH_ALLOWLIST unset; remediation PRs are disabled")
	}
	// Known exceptions to the allowlist above: a path an allow glob matches
	// but that is not actually reconciled (a dormant release, typically).
	// Kept in step with the platform repo's tools/orphaned-manifest-
	// allowlist.yaml by a CI assertion there; empty is fine, it just means no
	// exceptions are known yet.
	deniedPaths := splitList(os.Getenv("GITHUB_PATH_DENYLIST"))
	github := NewGitHubClient(githubAPI, githubTokens, allowedRepos, allowedPaths, deniedPaths, hc)

	// How long an alert must stay resolved before the relay closes the ticket
	// it opened for it.
	closeGrace := envDuration("RESOLVED_CLOSE_GRACE", defaultCloseGrace)
	pipeline := NewPipeline(holmes, patchGen, jira, github, discord, log, withCloseGrace(closeGrace))

	workers := envInt("MAX_CONCURRENT", 4)
	// The queue only has to keep workers fed between HTTP accept and pickup;
	// it cannot buy freshness, because depth costs latency at the drain rate
	// (workers divided by investigation time, which runs into minutes). Depth
	// beyond a couple of worker-rounds just converts a refusal the sender
	// would retry into an investigation delivered long after the incident it
	// describes. Overflow is held by the sender instead, which is durable and
	// bounded; this buffer is not.
	queueSize := envInt("QUEUE_SIZE", 2*workers)
	perAlertTO := time.Duration(envInt("INVESTIGATION_TIMEOUT_SECONDS", 240)) * time.Second
	disp := newDispatcher(pipeline, workers, queueSize, perAlertTO, pipeline.Counters(), log)

	webhookToken := os.Getenv("WEBHOOK_TOKEN")
	if webhookToken == "" {
		// ClusterIP limits exposure, but the downstream is paid/side-effecting;
		// an authenticated webhook is the intended posture.
		log.Warn("WEBHOOK_TOKEN unset; /webhook accepts any in-cluster caller")
	}

	srv := &http.Server{
		Addr:              ":" + env("PORT", "8080"),
		Handler:           newRouter(disp, pipeline.Counters(), webhookToken),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	// Pending closes are evaluated on a sweep rather than a per-ticket timer:
	// the tracked state is in-process only, so a restart drops pending closes
	// instead of leaving a timer to fire against state that no longer holds.
	sweepEvery := envDuration("RESOLVED_SWEEP_INTERVAL", 10*time.Minute)
	sweepCtx, stopSweeping := context.WithCancel(context.Background())
	defer stopSweeping()
	go func() {
		t := time.NewTicker(sweepEvery)
		defer t.Stop()
		for {
			select {
			case <-sweepCtx.Done():
				return
			case <-t.C:
				// One tick's work must finish before the next one is due;
				// without a deadline a wedged Jira stalls the sweep forever.
				tickCtx, tickCancel := context.WithTimeout(sweepCtx, sweepEvery)
				pipeline.SweepResolved(tickCtx)
				tickCancel()
			}
		}
	}()

	go func() {
		log.Info("relay listening", "addr", srv.Addr, "workers", workers, "queue", queueSize,
			"close_grace", closeGrace.String(), "sweep_every", sweepEvery.String())
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Info("shutting down")

	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	// Stop accepting new webhooks first, then drain in-flight investigations so
	// a rolling update does not silently discard work already accepted.
	_ = srv.Shutdown(ctx)
	stopSweeping()
	disp.shutdown(ctx)
	log.Info("shutdown complete")
}
