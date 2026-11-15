package budget

import (
	"bufio"
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func transactionFixture(t *testing.T) (string, admissionBinding) {
	t.Helper()
	dir, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(dir, 0700); err != nil {
		t.Fatal(err)
	}
	state := recordExample()
	data, err := encodeAdmissionRecord(state)
	if err != nil {
		t.Fatal(err)
	}
	for name, body := range map[string][]byte{"lock": {}, "state": data} {
		if err := os.WriteFile(filepath.Join(dir, name), body, 0600); err != nil {
			t.Fatal(err)
		}
	}
	return dir, admissionBinding{boot: state.boot, policy: state.policy, limit: state.ledger.limit}
}

func TestTransactionEntrySubprocess(t *testing.T) {
	dir := os.Getenv("AZAZEL_TRANSACTION_FIXTURE")
	if dir == "" {
		return
	}
	state := recordExample()
	tx, err := beginAdmissionTransaction(context.Background(), dir, admissionBinding{boot: state.boot, policy: state.policy, limit: state.ledger.limit})
	if err != nil {
		fmt.Println("REFUSED", err)
		os.Exit(12)
	}
	defer tx.close()
	if os.Getenv("AZAZEL_TRANSACTION_OBSERVE") == "1" {
		if err := tx.observe(context.Background(), state.owners[0]); err != nil {
			fmt.Println("REFUSED", err)
			os.Exit(13)
		}
	}
	fmt.Println("HELD")
	if os.Getenv("AZAZEL_TRANSACTION_WAIT") == "1" {
		_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
	}
}

func transactionChild(t *testing.T, dir string, observe bool) (*exec.Cmd, *bufio.Reader) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestTransactionEntrySubprocess$")
	cmd.Env = append(os.Environ(), "AZAZEL_TRANSACTION_FIXTURE="+dir, "AZAZEL_TRANSACTION_WAIT=1", fmt.Sprintf("AZAZEL_TRANSACTION_OBSERVE=%d", map[bool]int{true: 1}[observe]))
	out, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	in, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = in.Close(); _ = cmd.Process.Kill(); _ = cmd.Wait() })
	timer := time.AfterFunc(8*time.Second, func() { _ = cmd.Process.Kill() })
	t.Cleanup(func() { timer.Stop() })
	return cmd, bufio.NewReader(out)
}

func TestTransactionStableLockAndKilledOwner(t *testing.T) {
	dir, binding := transactionFixture(t)
	before, err := os.Stat(filepath.Join(dir, "lock"))
	if err != nil {
		t.Fatal(err)
	}
	child, out := transactionChild(t, dir, false)
	line, err := out.ReadString('\n')
	if err != nil || line != "HELD\n" {
		t.Fatalf("child: %q %v", line, err)
	}
	if tx, err := beginAdmissionTransaction(context.Background(), dir, binding); !errors.Is(err, errAdmissionBusy) || tx != nil {
		t.Fatalf("did not contend: %v", err)
	}
	if err := child.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = child.Wait()
	tx, err := beginAdmissionTransaction(context.Background(), dir, binding)
	if err != nil {
		t.Fatal(err)
	}
	if err := tx.close(); err != nil {
		t.Fatal(err)
	}
	if err := tx.close(); err == nil {
		t.Fatal("repeated close acknowledged")
	}
	after, err := os.Stat(filepath.Join(dir, "lock"))
	if err != nil || !os.SameFile(before, after) {
		t.Fatal("lock replaced")
	}
}

func TestTransactionKilledObservingOwnerFailsClosed(t *testing.T) {
	dir, binding := transactionFixture(t)
	before, _ := os.ReadFile(filepath.Join(dir, "state"))
	child, out := transactionChild(t, dir, true)
	line, err := out.ReadString('\n')
	if err != nil || line != "HELD\n" {
		t.Fatalf("child: %q %v", line, err)
	}
	if err := child.Process.Kill(); err != nil {
		t.Fatal(err)
	}
	_ = child.Wait()
	intentBytes, err := os.ReadFile(filepath.Join(dir, "intent"))
	if err != nil {
		t.Fatal(err)
	}
	intent, err := decodeAdmissionIntent(intentBytes)
	if err != nil || intent.phase != intentObserving || intent.predecessor != sha256.Sum256(before) {
		t.Fatalf("bad fence: %+v %v", intent, err)
	}
	if tx, err := beginAdmissionTransaction(context.Background(), dir, binding); !errors.Is(err, errAdmissionPending) || tx != nil {
		t.Fatalf("lost uncertainty: %v", err)
	}
	after, _ := os.ReadFile(filepath.Join(dir, "state"))
	if string(before) != string(after) {
		t.Fatal("state changed")
	}
}

func TestTransactionConcurrentObservation(t *testing.T) {
	dir, binding := transactionFixture(t)
	var wg sync.WaitGroup
	results := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tx, err := beginAdmissionTransaction(context.Background(), dir, binding)
			if err == nil {
				err = errors.Join(tx.observe(context.Background(), recordExample().owners[0]), tx.close())
			}
			results <- err
		}()
	}
	wg.Wait()
	close(results)
	succeeded := 0
	for err := range results {
		if err == nil {
			succeeded++
		} else if !errors.Is(err, errAdmissionBusy) && !errors.Is(err, errAdmissionPending) {
			t.Fatal(err)
		}
	}
	if succeeded != 1 {
		t.Fatalf("observations: %d", succeeded)
	}
}

func TestTransactionRefusesUnsafeEntry(t *testing.T) {
	cases := []string{"boot", "policy", "limit", "state-corrupt", "state-missing", "intent-corrupt", "intent-observing", "intent-prepared", "lock-missing", "lock-link", "state-link", "lock-hardlink", "directory-mode", "state-mode", "cancelled"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			dir, binding := transactionFixture(t)
			ctx := context.Background()
			switch name {
			case "boot":
				binding.boot = "other"
			case "policy":
				binding.policy.loadLimit++
			case "limit":
				binding.limit++
			case "state-corrupt":
				if err := os.WriteFile(filepath.Join(dir, "state"), []byte("bad"), 0600); err != nil {
					t.Fatal(err)
				}
			case "state-missing", "lock-missing":
				if err := os.Remove(filepath.Join(dir, strings.TrimSuffix(name, "-missing"))); err != nil {
					t.Fatal(err)
				}
			case "intent-corrupt":
				if err := os.WriteFile(filepath.Join(dir, "intent"), []byte("bad"), 0600); err != nil {
					t.Fatal(err)
				}
			case "intent-observing", "intent-prepared":
				state := recordExample()
				data, _ := encodeAdmissionRecord(state)
				intent := admissionIntent{version: 1, phase: intentObserving, boot: state.boot, owner: state.owners[0], predecessor: sha256.Sum256(data)}
				if name == "intent-prepared" {
					state.predecessor = intent.predecessor
					next, _ := encodeAdmissionRecord(state)
					intent.phase = intentPrepared
					intent.next = &state
					intent.stateDigest = sha256.Sum256(next)
				}
				encoded, err := encodeAdmissionIntent(intent)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "intent"), encoded, 0600); err != nil {
					t.Fatal(err)
				}
			case "lock-link", "state-link":
				entry := strings.TrimSuffix(name, "-link")
				if err := os.Rename(filepath.Join(dir, entry), filepath.Join(dir, "target")); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink("target", filepath.Join(dir, entry)); err != nil {
					t.Fatal(err)
				}
			case "lock-hardlink":
				if err := os.Link(filepath.Join(dir, "lock"), filepath.Join(dir, "alias")); err != nil {
					t.Fatal(err)
				}
			case "directory-mode":
				if err := os.Chmod(dir, 0755); err != nil {
					t.Fatal(err)
				}
			case "state-mode":
				if err := os.Chmod(filepath.Join(dir, "state"), 0644); err != nil {
					t.Fatal(err)
				}
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			}
			if tx, err := beginAdmissionTransaction(ctx, dir, binding); err == nil || tx != nil {
				if tx != nil {
					_ = tx.close()
				}
				t.Fatalf("unsafe entry accepted: %v", err)
			}
		})
	}
}

func TestTransactionObservationCancellationAndLifetime(t *testing.T) {
	dir, binding := transactionFixture(t)
	tx, err := beginAdmissionTransaction(context.Background(), dir, binding)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.close()
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := tx.observe(ctx, recordExample().owners[0]); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "intent")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("cancelled observation wrote intent")
	}
	if err := tx.observe(context.Background(), reservationOwner{name: "unknown", incarnation: "unknown"}); err == nil {
		t.Fatal("unknown owner accepted")
	}
	if err := tx.observe(context.Background(), recordExample().owners[0]); err != nil {
		t.Fatal(err)
	}
	if err := tx.observe(context.Background(), recordExample().owners[0]); err == nil {
		t.Fatal("repeat observation accepted")
	}
	if err := tx.close(); err != nil {
		t.Fatal(err)
	}
	if err := tx.observe(context.Background(), recordExample().owners[0]); err == nil {
		t.Fatal("closed transaction wrote")
	}
}

func TestTransactionConcurrentProcesses(t *testing.T) {
	dir, _ := transactionFixture(t)
	var wg sync.WaitGroup
	results := make(chan string, 6)
	for i := 0; i < 6; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, os.Args[0], "-test.run=^TestTransactionEntrySubprocess$")
			cmd.Env = append(os.Environ(), "AZAZEL_TRANSACTION_FIXTURE="+dir, "AZAZEL_TRANSACTION_WAIT=0", "AZAZEL_TRANSACTION_OBSERVE=1")
			output, err := cmd.CombinedOutput()
			if err == nil && strings.HasPrefix(string(output), "HELD\n") {
				results <- "held"
				return
			}
			if err != nil && strings.HasPrefix(string(output), "REFUSED ") {
				results <- "refused"
				return
			}
			results <- fmt.Sprintf("unexpected child: %q %v", output, err)
		}()
	}
	wg.Wait()
	close(results)
	held := 0
	for result := range results {
		if result == "held" {
			held++
		} else if result != "refused" {
			t.Fatal(result)
		}
	}
	if held != 1 {
		t.Fatalf("successful observing processes: %d", held)
	}
}

func TestTransactionRejectsReplacedLockAndLinkedDirectory(t *testing.T) {
	dir, binding := transactionFixture(t)
	alias := filepath.Join(dir, "linked")
	if err := os.Symlink(dir, alias); err != nil {
		t.Fatal(err)
	}
	if tx, err := beginAdmissionTransaction(context.Background(), alias, binding); err == nil || tx != nil {
		t.Fatal("followed directory link")
	}
	tx, err := beginAdmissionTransaction(context.Background(), dir, binding)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.close()
	if err := os.Rename(filepath.Join(dir, "lock"), filepath.Join(dir, "old-lock")); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "lock"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := tx.observe(context.Background(), recordExample().owners[0]); err == nil {
		t.Fatal("accepted replaced lock")
	}
	if _, err := os.Stat(filepath.Join(dir, "intent")); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("wrote through replaced lock")
	}
}

type transactionCancelAfterCheck struct {
	context.Context
	cancel  context.CancelFunc
	checked bool
}

func (ctx *transactionCancelAfterCheck) Err() error {
	err := ctx.Context.Err()
	if !ctx.checked {
		ctx.checked = true
		ctx.cancel()
	}
	return err
}

func TestTransactionCancellationAfterInitialCheckWritesNothing(t *testing.T) {
	dir, binding := transactionFixture(t)
	tx, err := beginAdmissionTransaction(context.Background(), dir, binding)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.close()
	base, cancel := context.WithCancel(context.Background())
	defer cancel()
	ctx := &transactionCancelAfterCheck{Context: base, cancel: cancel}
	if err := tx.observe(ctx, recordExample().owners[0]); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, "intent")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cancelled before create left intent: %v", err)
	}
	if tx.observed {
		t.Fatal("cancelled before create consumed observation")
	}
	if err := tx.observe(context.Background(), recordExample().owners[0]); err != nil {
		t.Fatalf("retry after pre-create cancellation: %v", err)
	}
}
