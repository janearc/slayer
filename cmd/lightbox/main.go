// Command lightbox serves slayer's smokes page and its files to a browser
// on this machine: smokes/index.html, results.jsonl and the pictures.
//
//	lightbox [-at 127.0.0.1:8765] [-dir smokes]
//
// It listens on this machine alone; nothing else can reach it. /health
// says it is up. Every request is logged as one json line.
package main

import (
	"flag"
	"log/slog"
	"net/http"
	"os"
	"time"
)

// main serves the folder until it is stopped
func main() {
	at := flag.String("at", "127.0.0.1:8765", "where to listen")
	dir := flag.String("dir", "smokes", "the folder to serve")
	flag.Parse()
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	if _, err := os.Stat(*dir); err != nil {
		logger.Error("no folder to serve", "dir", *dir, "err", err)
		os.Exit(1)
	}
	server := &http.Server{Addr: *at, Handler: routes(*dir, logger),
		ReadHeaderTimeout: 5 * time.Second}
	logger.Info("serving", "at", *at, "dir", *dir)
	if err := server.ListenAndServe(); err != nil {
		logger.Error("stopped", "err", err)
		os.Exit(1)
	}
}

// routes is the folder's files, and /health, each request logged
func routes(dir string, logger *slog.Logger) http.Handler {
	files := http.FileServer(http.Dir(dir))
	mux := http.NewServeMux()
	mux.HandleFunc("/health",
		func(writer http.ResponseWriter, _ *http.Request) {
			writer.Write([]byte("ok\n"))
		})
	mux.Handle("/", files)
	return logged(mux, logger)
}

// logged writes one line for every request: the path and how long it took
func logged(next http.Handler, logger *slog.Logger) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter,
		request *http.Request) {
		start := time.Now()
		next.ServeHTTP(writer, request)
		logger.Info("served", "path", request.URL.Path,
			"took", time.Since(start).String())
	})
}
