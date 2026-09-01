package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/oz-fatma/kontrata/backend/internal/config"
)

// version derleme sırasında ldflags ile gömülür; verilmezse "dev".
var version = "dev"

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("yapılandırma yüklenemedi: %v", err)
	}
	log.Printf("yapılandırma yüklendi %s sürüm=%s", cfg.String(), version)

	srv := newServer(cfg)

	go func() {
		log.Printf("sunucu dinliyor addr=:%d", cfg.Port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("sunucu durdu: %v", err)
		}
	}()

	wait := make(chan os.Signal, 1)
	signal.Notify(wait, syscall.SIGINT, syscall.SIGTERM)
	sig := <-wait
	log.Printf("kapanış sinyali alındı signal=%s", sig)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		log.Fatalf("düzgün kapanış başarısız: %v", err)
	}
	log.Printf("sunucu durdu")
}

func newServer(cfg config.Config) *http.Server {
	r := chi.NewRouter()
	r.Use(recoverPanic)
	r.Use(logRequest)
	r.Get("/healthz", handleHealthz)

	return &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Port),
		Handler:           r,
		ReadTimeout:       10 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

type healthResponse struct {
	Status  string `json:"status"`
	Version string `json:"version"`
}

func handleHealthz(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	resp := healthResponse{Status: "ok", Version: version}
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		log.Printf("healthz yanıtı yazılamadı: %v", err)
	}
}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	w.status = code
	w.ResponseWriter.WriteHeader(code)
}

func logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		// Yalnızca metot, yol, durum ve süre. Gövde, sorgu ve başlık yok.
		log.Printf("%s %s %d %s", r.Method, r.URL.Path, sw.status, time.Since(start))
	})
}

func recoverPanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			log.Printf("panic kurtarıldı: %v", rec)
			http.Error(w, "iç sunucu hatası", http.StatusInternalServerError)
		}()
		next.ServeHTTP(w, r)
	})
}
