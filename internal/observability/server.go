package observability

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"net/http/pprof"
	"os"
	"sync"
	"sync/atomic"
	"time"
)

var consumers atomic.Int64
var active atomic.Bool

// Enabled reports whether any endpoint or summary reporter consumes observations.
func Enabled() bool { return consumers.Load() > 0 }

// Start binds only configured endpoints. Profiling is restricted to loopback.
// The returned function closes any listeners; it is safe when disabled.
func Start(app string) (func(), error) {
	metricsAddress := os.Getenv("SLAYER_METRICS_ADDR")
	profileAddress := os.Getenv("SLAYER_PPROF_ADDR")
	if metricsAddress == "" && profileAddress == "" {
		return func() {}, nil
	}
	if !active.CompareAndSwap(false, true) {
		return nil, fmt.Errorf("observability already started")
	}
	type endpoint struct {
		listener net.Listener
		server   *http.Server
	}
	var endpoints []endpoint
	var once sync.Once
	stop := func() {
		once.Do(func() {
			consumers.Add(-1)
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			for _, endpoint := range endpoints {
				_ = endpoint.server.Shutdown(ctx)
			}
			active.Store(false)
		})
	}
	open := func(address string, handler http.Handler) error {
		listener, err := net.Listen("tcp", address)
		if err != nil {
			return err
		}
		server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
		endpoints = append(endpoints, endpoint{listener: listener, server: server})
		return nil
	}
	if address := metricsAddress; address != "" {
		if err := open(address, Handler()); err != nil {
			active.Store(false)
			return nil, fmt.Errorf("metrics listener: %w", err)
		}
	}
	if address := profileAddress; address != "" {
		host, _, err := net.SplitHostPort(address)
		if err != nil || !(host == "localhost" || net.ParseIP(host) != nil && net.ParseIP(host).IsLoopback()) {
			for _, endpoint := range endpoints {
				_ = endpoint.listener.Close()
			}
			active.Store(false)
			return nil, fmt.Errorf("profiling listener must use a loopback address")
		}
		mux := http.NewServeMux()
		mux.HandleFunc("/debug/pprof/", pprof.Index)
		mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
		mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
		mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
		mux.HandleFunc("/debug/pprof/trace", pprof.Trace)
		if err := open(address, mux); err != nil {
			for _, endpoint := range endpoints {
				_ = endpoint.listener.Close()
			}
			active.Store(false)
			return nil, fmt.Errorf("profiling listener: %w", err)
		}
	}
	if len(endpoints) > 0 {
		Set(app+"_observability_up", 1)
		consumers.Add(1)
	}
	for _, item := range endpoints {
		go func(e endpoint) { _ = e.server.Serve(e.listener) }(item)
	}
	return stop, nil
}
