package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
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
	hc := &http.Client{Timeout: 90 * time.Second}

	holmes := NewHolmesClient(env("HOLMES_URL", "http://holmes-holmes.ai-sre.svc.cluster.local"), hc)
	patchGen := NewPatchGenerator(
		env("LITELLM_URL", "http://litellm.ai-sre.svc.cluster.local:4000/v1"),
		mustEnv("LITELLM_KEY"),
		env("LITELLM_MODEL", "claude-sonnet"),
		0.75, hc,
	)
	discord := NewDiscordNotifier(mustEnv("DISCORD_WEBHOOK_URL"), hc)
	jira := NewJiraClient(mustEnv("JIRA_URL"), mustEnv("JIRA_USERNAME"), mustEnv("JIRA_API_TOKEN"), env("JIRA_PROJECT", "JDWLABS"), hc)
	github := NewGitHubClient(env("GITHUB_API", "https://api.github.com"), mustEnv("GITHUB_TOKEN"), hc)

	pipeline := NewPipeline(holmes, patchGen, jira, github, discord, log)

	srv := &http.Server{Addr: ":" + env("PORT", "8080"), Handler: newRouter(pipeline)}

	go func() {
		log.Info("relay listening", "addr", srv.Addr)
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Error("server error", "err", err)
			os.Exit(1)
		}
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	log.Info("shutting down")
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = srv.Shutdown(ctx)
}
