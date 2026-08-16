package roomworker

import (
	"context"
	"log"
	"net/http"
	"time"
)

func (w *Worker) startHealthServer(ctx context.Context) {
	mux := http.NewServeMux()
	mux.HandleFunc("/livez", func(rw http.ResponseWriter, _ *http.Request) {
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write([]byte("ok"))
	})
	mux.HandleFunc("/readyz", func(rw http.ResponseWriter, r *http.Request) {
		if !w.ready.Load() {
			http.Error(rw, "worker not ready", http.StatusServiceUnavailable)
			return
		}
		checkCtx, cancel := context.WithTimeout(r.Context(), time.Second)
		defer cancel()
		if err := w.redis.Ping(checkCtx).Err(); err != nil {
			http.Error(rw, "redis unavailable", http.StatusServiceUnavailable)
			return
		}
		if err := w.store.Ping(checkCtx); err != nil {
			http.Error(rw, "database unavailable", http.StatusServiceUnavailable)
			return
		}
		rw.WriteHeader(http.StatusOK)
		_, _ = rw.Write([]byte("ok"))
	})

	server := &http.Server{
		Addr:              w.cfg.HealthAddr,
		Handler:           mux,
		ReadHeaderTimeout: 3 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdownCtx)
	}()

	go func() {
		log.Printf("room worker %d health server listening on %s", w.cfg.WorkerID, w.cfg.HealthAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("roomworker: health server: %v", err)
		}
	}()
}
