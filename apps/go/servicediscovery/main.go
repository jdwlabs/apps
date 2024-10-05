package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/rs/cors"
	"libs/go/shared/util"
	"log"
	"log/slog"
	"net/http"
	"os"
	"syscall"
	"time"
)

var config = map[string]string{}

type envGetter interface {
	getEnvOrDefault(key, fallback string) string
}

type realEnv struct{}

func (e realEnv) getEnvOrDefault(key, fallback string) string {
	return util.GetEnvOrDefault(key, fallback)
}

func rootHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, err := fmt.Fprintf(w, "Service Discovery is up and running")
	if err != nil {
		return
	}
}

func healthHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	_, err := fmt.Fprintf(w, "Service is healthy")
	if err != nil {
		return
	}
}

func configHandler(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	err := json.NewEncoder(w).Encode(config)
	if err != nil {
		return
	}
}

func versionHandler(envGetter envGetter) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		version := envGetter.getEnvOrDefault("SD_VERSION", "local")
		w.Header().Set("Content-Type", "application/json")
		_, err := fmt.Fprintf(w, `{"version": "%s"}`, version)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = fmt.Fprintf(w, `{"error": "Unable to fetch version"}`)
		}
	}
}

func initServer(envGetter envGetter) *http.Server {
	port := envGetter.getEnvOrDefault("SD_PORT", "9000")
	router := http.NewServeMux()
	registerRoutes(router, envGetter)

	return &http.Server{
		Addr:    ":" + port,
		Handler: util.Logging(cors.Default().Handler(router)),
	}
}

func loadConfig(envGetter envGetter) {
	configPath := envGetter.getEnvOrDefault("SD_CONFIG_PATH", "config.json")
	file, err := os.Open(configPath)
	if err != nil {
		log.Printf("Error loading config file at %s: %+v. Using default config.", configPath, err)
		return
	}
	defer func(file *os.File) {
		err := file.Close()
		if err != nil {
			log.Printf("Error closing config file: %+v", err)
		}
	}(file)

	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		log.Printf("Error decoding config: %+v. Using default config.", err)
	}
}

func registerRoutes(router *http.ServeMux, envGetter envGetter) {
	router.HandleFunc("GET /", rootHandler)
	router.HandleFunc("GET /health", healthHandler)
	router.HandleFunc("GET /config", configHandler)
	router.HandleFunc("GET /version", versionHandler(envGetter))
}

func main() {
	loadConfig(realEnv{})
	server := initServer(realEnv{})

	go func() {
		slog.Info("Starting server on port " + server.Addr)
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("HTTP server error: %+v", err)
		}
		slog.Info("Stopped serving new connections")
	}()

	stopChan := make(chan os.Signal, syscall.SIGTERM)
	<-stopChan
	slog.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Server shutdown failed: %+v", err)
	}
	slog.Info("Server shutdown successfully")
}
