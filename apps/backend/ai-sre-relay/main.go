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
	patchGen := NewPatchGenerator(
		env("LITELLM_URL", "http://platform-litellm.ai-sre.svc.cluster.local:4000/v1"),
		mustEnv("LITELLM_KEY"),
		env("LITELLM_MODEL", "claude-sonnet"),
		0.75, llmc,
	)
	jiraURL := mustEnv("JIRA_URL")
	discord := NewDiscordNotifier(mustEnv("DISCORD_WEBHOOK_URL"), jiraURL, hc)
	// "Task" not "Bug": the target project's type scheme has no Bug type and
	// Jira rejects creates with an unknown type (400).
	jira := NewJiraClient(jiraURL, mustEnv("JIRA_USERNAME"), mustEnv("JIRA_API_TOKEN"), env("JIRA_PROJECT", "JDWLABS"), env("JIRA_ISSUE_TYPE", "Task"), hc)
	// Bounds where a model-authored patch may be written. Unset disables the
	// PR arm rather than trusting the repo the model names.
	allowedRepos := splitList(os.Getenv("GITHUB_REPO_ALLOWLIST"))
	if len(allowedRepos) == 0 {
		log.Warn("GITHUB_REPO_ALLOWLIST unset; remediation PRs are disabled")
	}
	github := NewGitHubClient(env("GITHUB_API", "https://api.github.com"), mustEnv("GITHUB_TOKEN"), allowedRepos, hc)

	pipeline := NewPipeline(holmes, patchGen, jira, github, discord, log)

	workers := envInt("MAX_CONCURRENT", 4)
	queueSize := envInt("QUEUE_SIZE", 64)
	perAlertTO := time.Duration(envInt("INVESTIGATION_TIMEOUT_SECONDS", 240)) * time.Second
	disp := newDispatcher(pipeline, workers, queueSize, perAlertTO, log)

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

	go func() {
		log.Info("relay listening", "addr", srv.Addr, "workers", workers, "queue", queueSize)
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
	disp.shutdown(ctx)
	log.Info("shutdown complete")
}
