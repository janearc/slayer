package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/janearc/slayer/azazel/internal/queue"
)

// TestDryRunNeverInvokesAQueueMutation protects the discoverable Pzm contract.
func TestDryRunNeverInvokesAQueueMutation(t *testing.T) {
	for _, name := range []string{"init", "submit", "cancel", "pause", "resume", "tune", "work", "work-gpu", "recover", "execute"} {
		t.Run(name, func(t *testing.T) {
			directory := t.TempDir()
			root := filepath.Join(directory, "queue")
			store := queue.At(root)
			request := queue.Request{Agent: "test", Key: "dry-run", Command: "/usr/bin/true", Args: []string{}, Directory: directory, Lane: "gpu", TimeoutSeconds: 10}
			data, err := json.Marshal(request)
			if err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(directory, "request.json")
			if err = os.WriteFile(path, data, 0600); err != nil {
				t.Fatal(err)
			}
			var before []byte
			if name != "init" {
				if err = store.Init(t.Context(), queue.Config{Game: "/usr/bin/true"}); err != nil {
					t.Fatal(err)
				}
				if _, err = store.Submit(t.Context(), request); err != nil {
					t.Fatal(err)
				}
				before, err = os.ReadFile(filepath.Join(root, "queue.db"))
				if err != nil {
					t.Fatal(err)
				}
			}
			args := []string{name, "--queue", root, "--dry-run"}
			switch name {
			case "init":
				args = append(args, "--game", "/usr/bin/true")
			case "submit":
				args = append(args, path)
			case "cancel":
				args = append(args, "--agent", "test", "1")
			case "pause":
				args = append(args, "--reason", "preview")
			case "recover", "execute":
				args = append(args, "--worker", "preview", "1")
			}
			var out, wrong bytes.Buffer
			code := queueTool(t.Context()).Run(args, &out, &wrong)
			if code != 0 || !strings.Contains(out.String(), `"dry_run":true`) {
				t.Fatal(code, out.String(), wrong.String())
			}
			if name == "init" {
				if _, err = os.Stat(root); !os.IsNotExist(err) {
					t.Fatal("dry run created queue", err)
				}
			} else {
				after, err := os.ReadFile(filepath.Join(root, "queue.db"))
				if err != nil || !bytes.Equal(before, after) {
					t.Fatal("dry run wrote database", err)
				}
				if _, err = os.Stat(filepath.Join(root, "worker.lock")); !os.IsNotExist(err) {
					t.Fatal("dry run dispatched", err)
				}
			}
		})
	}
}
