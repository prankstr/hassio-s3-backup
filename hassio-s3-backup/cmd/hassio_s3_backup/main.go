package main

import (
	"context"
	"errors"
	"hassio-proton-drive-backup/internal/backup"
	"hassio-proton-drive-backup/internal/config"
	"hassio-proton-drive-backup/internal/s3"
	"hassio-proton-drive-backup/webui"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	// Initalize config
	cs := config.NewConfigService()
	conf := cs.GetConfig()

	// Set LogLevel
	h := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{Level: conf.LogLevel})
	slog.SetDefault(slog.New(h))

	// Initalize S3
	s3, err := s3.NewClient(cs)
	if err != nil {
		slog.Error("failed to initialize S3 client", "error", err)
		os.Exit(1)
	}

	// Initialize the backup service
	bs := backup.NewService(s3, cs)

	// Initialize mux and register routes
	mux := http.NewServeMux()
	backup.RegisterBackupRoutes(mux, bs)
	config.RegisterConfigRoutes(mux, cs)

	// Setup UI route and handler
	uiHandler := webui.NewHandler(conf)
	mux.Handle("/", uiHandler)

	// Define and start HTTP server
	server := http.Server{
		Addr:    ":8099",
		Handler: mux,
	}

	go func() {
		slog.Info("starting HTTP server", "address", server.Addr)
		if err := server.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			slog.Error("HTTP server error", "error", err)
			os.Exit(1)
		}
		slog.Info("stopped serving new connections.")
	}()

	// TODO: Decide if the external server is needed
	// Could be used for metrics or direct access to the UI
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

	// Graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan

	slog.Info("initializing graceful shutdown")
	shutdownCtx, shutdownRelease := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownRelease()

	if err := server.Shutdown(shutdownCtx); err != nil {
		slog.Error("http shutdown error", "error", err)
		os.Exit(1)
	}

	slog.Info("graceful shutdown complete")
}
