package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/janearc/slayer/internal/commandtest"
)

var lightboxCommand string

// TestMain retains one real lightbox command for its startup refusal tests.
func TestMain(m *testing.M) {
	directory, err := os.MkdirTemp("", "slayer-lightbox-command-")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	lightboxCommand = filepath.Join(directory, "lightbox")
	if err = commandtest.Build(".", ".", lightboxCommand, testing.CoverMode()); err != nil {
		os.RemoveAll(directory)
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	code := m.Run()
	os.RemoveAll(directory)
	os.Exit(code)
}

func TestHealthFilesAndMissingRequestsKeepTheirStatusAndLogPath(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "image.txt"), []byte("actual gallery file\n"), 0600); err != nil {
		t.Fatal(err)
	}
	var log bytes.Buffer
	handler := routes(root, slog.New(slog.NewJSONHandler(&log, nil)))
	for _, test := range []struct {
		path   string
		status int
		body   string
	}{{"/health", 200, "ok\n"}, {"/image.txt", 200, "actual gallery file\n"}, {"/missing", 404, "404 page not found\n"}} {
		request := httptest.NewRequest(http.MethodGet, test.path, nil)
		reply := httptest.NewRecorder()
		handler.ServeHTTP(reply, request)
		if reply.Code != test.status || reply.Body.String() != test.body {
			t.Fatalf("%s: %d %q", test.path, reply.Code, reply.Body.String())
		}
		var record struct{ Msg, Path, Took string }
		if err := json.Unmarshal(bytes.TrimSpace(log.Bytes()), &record); err != nil {
			t.Fatal(err)
		}
		if record.Msg != "served" || record.Path != test.path {
			t.Fatalf("wrong log %+v", record)
		}
		if elapsed, err := time.ParseDuration(record.Took); err != nil || elapsed < 0 {
			t.Fatalf("bad duration %q", record.Took)
		}
		log.Reset()
	}
}

func TestRealServerStartupReportsMissingDirectoryAndBadPort(t *testing.T) {
	for _, test := range []struct {
		args []string
		want string
	}{
		{[]string{"-help"}, "-dir"},
		{[]string{"-dir", filepath.Join(t.TempDir(), "missing")}, "no folder to serve"},
		{[]string{"-dir", t.TempDir(), "-at", "127.0.0.1:bad-port"}, "bad-port"},
	} {
		ctx, cancel := context.WithTimeout(t.Context(), 10*time.Second)
		out, err := exec.CommandContext(ctx, lightboxCommand, test.args...).CombinedOutput()
		cancel()
		help := test.args[0] == "-help"
		if (err == nil) != help || !strings.Contains(string(out), test.want) {
			t.Fatalf("%v: %s", err, out)
		}
		if !help && !strings.Contains(string(out), `"level":"ERROR"`) {
			t.Fatalf("startup error not logged: %s", out)
		}
	}
}
