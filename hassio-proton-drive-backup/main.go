package main

import (
	"context"
	"errors"
	"hassio-proton-drive-backup/api/server"
	"hassio-proton-drive-backup/internal/config"
	"hassio-proton-drive-backup/internal/storage"
	"hassio-proton-drive-backup/internal/storage_backends"
	"hassio-proton-drive-backup/web"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

/* func getIP(r *http.Request) (string, error) {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "", err
	}

	netIP := net.ParseIP(ip)
	if netIP != nil {
		slog.Debug("Incoming request", "ip", netIP)
		return ip, nil
	}

	return "", fmt.Errorf("no valid ip found")
} */

func main() {
	// Initalize config
	configService := config.NewConfigService()
	config := configService.GetConfig()

	// Set LogLevel
	h := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: config.LogLevel})
	slog.SetDefault(slog.New(h))

	var err error
	var storageService storage.Service
	switch config.StorageBackend {
	case storage.STORJ:
		storageService, err = storage_backends.NewStorjService(configService)
		if err != nil {
			slog.Error("Could not initialize storage backend", "err", err, "storage backend", config.StorageBackend)
			os.Exit(1)
		}
	case storage.PROTON:
		storageService, err = storage_backends.NewProtonDriveService(configService)
		if err != nil {
			slog.Error("Could not initialize storage backend", "err", err, "storage backend", config.StorageBackend)
			os.Exit(1)
		}
	case storage.S3:
		storageService, err = storage_backends.NewS3Service(configService)
		if err != nil {
			slog.Error("Could not initialize storage backend", "err", err, "storage backend", config.StorageBackend)
			os.Exit(1)
		}
	default:
		slog.Error("unknown storage backend", "storage backend", config.StorageBackend)
		os.Exit(1)
	}

	api, err := server.NewServer(configService, storageService)
	if err != nil {
		slog.Error("Failed to initialize API", "error", err)
		os.Exit(1)
	}

	uiHandler := web.NewHandler(config)

	api.Router.Handle("/", uiHandler)

	// Define server
	server := http.Server{
		Addr:    ":8099",
		Handler: api.Router,
	}

	// Start http server
	go func() {
		slog.Info("Starting http server on port 8099")
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
		slog.Info("Stopped serving new connections.")
	}()

	/* 	// Define server
	   	extServer := http.Server{
	   		Addr:    ":9101",
	   		Handler: proxy.Router,
	   	}
	   	// Start http server
	   	go func() {
	   		slog.Info("Starting http server on port 9101")
	   		if err := extServer.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
	   			slog.Error("HTTP server error", "error", err)
	   			os.Exit(1)
	   		}
	   		slog.Info("Stopped serving new connections.")
	   	}() */

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	slog.Info("Initializing graceful shutdown")
	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("HTTP shutdown error", "error", err)
		os.Exit(1)
	}

	slog.Info("Graceful shutdown complete")
}
