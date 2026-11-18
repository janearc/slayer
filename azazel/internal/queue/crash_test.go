package queue

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// ledgerHelper exposes real transaction boundaries to a separate kill controller.
func ledgerHelper() int {
	if len(os.Args) != 5 {
		return 90
	}
	store := At(os.Args[1])
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if os.Args[2] == "uncommitted" {
		err := store.transact(ctx, false, func(snapshot *Snapshot) error {
			snapshot.Jobs = nil
			fmt.Fprintln(os.Stdout, "inside transaction")
			<-ctx.Done()
			return ctx.Err()
		})
		if err != nil {
			return 91
		}
		return 92
	}
	request := Request{Agent: "crash-test", Key: os.Args[2], Command: "/usr/bin/true",
		Directory: os.Args[3], Args: []string{}, Lane: "gpu", TimeoutSeconds: 10}
	job, err := store.Submit(ctx, request)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 93
	}
	if os.Args[4] == "claim" {
		job, err = store.claim(ctx, "killed-worker")
		if err != nil {
			return 94
		}
	}
	fmt.Fprintln(os.Stdout, job.Sequence)
	<-ctx.Done()
	return 95
}

// ledgerProcess starts a bounded fixture and returns its acknowledgement stream.
func ledgerProcess(t *testing.T, store Store, key, mode string) (*exec.Cmd, *bufio.Reader) {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(t.Context(), 20*time.Second)
	t.Cleanup(cancel)
	command := exec.CommandContext(ctx, executable, store.root, key, filepath.Dir(store.root), mode)
	command.Env = append(os.Environ(), "AZAZEL_QUEUE_HELPER=ledger", "GOMAXPROCS=1")
	command.Stderr = os.Stderr
	output, err := command.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err = command.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { command.Process.Kill(); command.Wait() })
	return command, bufio.NewReader(output)
}

// TestAcknowledgedConcurrentSubmissionsSurviveKilledClients checks real processes.
func TestAcknowledgedConcurrentSubmissionsSurviveKilledClients(t *testing.T) {
	store, _ := fixture(t)
	var commands []*exec.Cmd
	var outputs []*bufio.Reader
	for index := 0; index < 4; index++ {
		command, output := ledgerProcess(t, store, fmt.Sprint(index), "submit")
		commands = append(commands, command)
		outputs = append(outputs, output)
	}
	for index, command := range commands {
		if line, err := outputs[index].ReadString('\n'); err != nil || line == "" {
			t.Fatal(line, err)
		}
		if err := command.Process.Kill(); err != nil {
			t.Fatal(err)
		}
		command.Wait()
	}
	snapshot, err := At(store.root).Status(t.Context())
	if err != nil || len(snapshot.Jobs) != 4 {
		t.Fatal(snapshot, err)
	}
	seen := map[string]bool{}
	for index, job := range snapshot.Jobs {
		if job.Sequence != uint64(index+1) || job.State != Queued || seen[job.Request.Key] {
			t.Fatal(job)
		}
		seen[job.Request.Key] = true
	}
}

// TestKilledUncommittedWriterCannotEraseAcknowledgedJobs checks rollback after death.
func TestKilledUncommittedWriterCannotEraseAcknowledgedJobs(t *testing.T) {
	store, request := fixture(t)
	accepted, err := store.Submit(t.Context(), request)
	if err != nil {
		t.Fatal(err)
	}
	command, output := ledgerProcess(t, store, "uncommitted", "submit")
	if line, err := output.ReadString('\n'); err != nil || line != "inside transaction\n" {
		t.Fatal(line, err)
	}
	if err := command.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	command.Wait()
	snapshot, err := At(store.root).Status(t.Context())
	if err != nil || len(snapshot.Jobs) != 1 || snapshot.Jobs[0].Request.Key != accepted.Request.Key {
		t.Fatal(snapshot, err)
	}
	duplicate, err := store.Submit(t.Context(), request)
	if err != nil || duplicate.Sequence != accepted.Sequence {
		t.Fatal(duplicate, err)
	}
}

// TestKilledWorkerDoesNotAuthorizeAnotherAttempt preserves the uncertain claim.
func TestKilledWorkerDoesNotAuthorizeAnotherAttempt(t *testing.T) {
	store, request := fixture(t)
	command, output := ledgerProcess(t, store, "claimed", "claim")
	if line, err := output.ReadString('\n'); err != nil || line != "1\n" {
		t.Fatal(line, err)
	}
	if err := command.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	command.Wait()
	if _, err := store.Submit(t.Context(), request); err != nil {
		t.Fatal(err)
	}
	if _, err := store.claim(t.Context(), "replacement"); err != ErrActive {
		t.Fatal(err)
	}
	snapshot, err := store.Status(t.Context())
	if err != nil || len(snapshot.Jobs) != 2 || snapshot.Jobs[0].State != Running || snapshot.Jobs[1].State != Queued {
		t.Fatal(snapshot, err)
	}
}
