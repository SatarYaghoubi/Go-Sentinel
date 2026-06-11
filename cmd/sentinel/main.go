package main

import (
	"context"
	"errors"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/SatarYaghoubi/sentinel/internal/api"
	"github.com/SatarYaghoubi/sentinel/internal/ratelimit"
	"github.com/SatarYaghoubi/sentinel/internal/scanner"
	"github.com/SatarYaghoubi/sentinel/internal/store"
)

func main() {
	addr := flag.String("addr", ":8080", "address to listen on")
	workers := flag.Int("workers", 4, "number of scanner worker goroutines")
	rate := flag.Float64("rate", 5, "allowed requests per second per client")
	burst := flag.Int("burst", 10, "burst size per client")
	flag.Parse()

	sc := scanner.New(scanner.DefaultRules(), *workers)
	st := store.New()
	limiter := ratelimit.New(*rate, *burst)

	// Periodically evict idle rate-limit buckets.
	stopCleanup := make(chan struct{})
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				limiter.Cleanup(10 * time.Minute)
			case <-stopCleanup:
				return
			}
		}
	}()

	handler := api.NewHandler(sc, st)
	router := api.NewRouter(handler, limiter)

	srv := &http.Server{
		Addr:         *addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Start the server in the background.
	go func() {
		log.Printf("sentinel listening on %s (workers=%d)", *addr, *workers)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server error: %v", err)
		}
	}()

	// Wait for an interrupt signal, then shut down gracefully.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Printf("graceful shutdown failed: %v", err)
	}
	close(stopCleanup)
	sc.Shutdown()
	log.Println("stopped")
}
