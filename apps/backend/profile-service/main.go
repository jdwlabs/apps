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

	"github.com/jackc/pgx/v5/pgxpool"

	"libs/backend/shared/auth"
)

func main() {
	// JSON on stdout, so the log pipeline reads fields rather than parsing a
	// line format, matching what the JVM service ships today.
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))

	if err := run(); err != nil {
		slog.Error("the service could not start", "error", err)
		os.Exit(1)
	}
}

func run() error {
	config, err := configFromEnvironment()
	if err != nil {
		return err
	}

	verifier, err := auth.NewVerifier(auth.Config{
		SecretKeyBase64:           config.SecretKeyBase64,
		ExpectedIssuer:            config.ExpectedIssuer,
		ExpectedAudience:          config.ExpectedAudience,
		AllowAnyIssuerAndAudience: config.AllowAnyIssuerAndAudience,
	})
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	pool, err := openPool(ctx, config)
	if err != nil {
		return err
	}
	defer pool.Close()

	server, err := NewServer(ServerConfig{
		Store:    NewPostgresStore(pool),
		Verifier: verifier,
		CORS:     config.CORS,
	})
	if err != nil {
		return err
	}

	return serve(ctx, config, server.Handler())
}

func openPool(ctx context.Context, config Config) (*pgxpool.Pool, error) {
	poolConfig, err := pgxpool.ParseConfig(config.DatabaseDSN)
	if err != nil {
		return nil, err
	}
	poolConfig.MaxConns = config.MaxConnections
	poolConfig.MinConns = config.MinConnections

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, err
	}
	// Dialling here rather than on the first request turns a wrong password into
	// a failure to start, instead of a service that reports itself healthy and
	// answers every request with a 500.
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return pool, nil
}

func serve(ctx context.Context, config Config, handler http.Handler) error {
	server := &http.Server{
		Addr:              config.Address,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
	}

	failed := make(chan error, 1)
	go func() {
		slog.Info("listening", "address", server.Addr)
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			failed <- err
		}
	}()

	select {
	case err := <-failed:
		return err
	case <-ctx.Done():
	}

	slog.Info("shutting down")
	// Detached from the cancelled context, which is what makes the drain a
	// drain: shutting down on a context that is already done closes in-flight
	// connections immediately.
	shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx),
		time.Duration(config.ShutdownTimeoutSeconds)*time.Second)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		return err
	}
	slog.Info("shut down cleanly")
	return nil
}
