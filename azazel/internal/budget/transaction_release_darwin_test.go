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
	"reflect"
	"strings"
	"testing"
	"time"
)

var releasePublishPoints = []string{"prepared-write", "prepared-sync", "prepared-close", "prepared-rename", "prepared-directory", "state-write", "state-sync", "state-close", "state-rename", "state-directory", "retiring-write", "retiring-sync", "retiring-close", "retiring-directory", "intent-remove", "retired-directory", "retiring-remove", "complete-directory"}

func releaseRequest(tx *admissionTransaction, binding admissionBinding, id reservationIdentity) transactionReleaseRequest {
	release := reservationRelease{id: id, owner: tx.owner}
	return transactionReleaseRequest{binding: binding, release: release, proof: reservationQuiescenceProof{binding: binding, release: release, predecessor: tx.digest, state: releaseWorkAndDescendantsQuiescent}}
}

func transactionReleaseFixture(t *testing.T, full bool) (*admissionTransaction, transactionReleaseRequest) {
	t.Helper()
	dir, binding := transactionFixture(t)
	state := recordExample()
	if full {
		state.ledger.limit = admissionEntryLimit
		state.ledger.reservations = nil
		for i := 0; i < admissionEntryLimit; i++ {
			held := reservationIdentity(fmt.Sprintf("held-%d", i))
			state.ledger.reservations = append(state.ledger.reservations, ownedMemoryReservation{id: held, owner: state.owners[0], amount: 1})
		}
		binding.limit = state.ledger.limit
	}
	data, err := encodeAdmissionRecord(state)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "state"), data, 0600); err != nil {
		t.Fatal(err)
	}
	tx, err := beginAdmissionTransaction(context.Background(), dir, binding)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if !tx.closed {
			_ = tx.close()
		}
	})
	held := tx.state.ledger.reservations[0]
	if err := tx.observe(context.Background(), held.owner); err != nil {
		t.Fatal(err)
	}
	return tx, releaseRequest(tx, binding, held.id)
}

func releaseReadState(t *testing.T, dir string) ([]byte, admissionRecord) {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "state"))
	if err != nil {
		t.Fatal(err)
	}
	state, err := decodeAdmissionRecord(data)
	if err != nil {
		t.Fatal(err)
	}
	return data, state
}

func TestReleaseDurableAndIdempotent(t *testing.T) {
	for _, full := range []bool{false, true} {
		t.Run(fmt.Sprint(full), func(t *testing.T) {
			tx, request := transactionReleaseFixture(t, full)
			dir := tx.directory.Name()
			before, predecessor := releaseReadState(t, dir)
			original := request
			used, _ := predecessor.ledger.validatedUsage()
			amount := predecessor.ledger.reservations[0].amount
			result, err := tx.release(context.Background(), request)
			if err != nil || result.disposition != releaseProposed {
				t.Fatalf("release: %+v %v", result, err)
			}
			data, state := releaseReadState(t, dir)
			afterUsed, valid := state.ledger.validatedUsage()
			if !valid || afterUsed != used-amount || len(state.ledger.reservations) != len(predecessor.ledger.reservations)-1 || state.predecessor != sha256.Sum256(before) || result.stateDigest != sha256.Sum256(data) {
				t.Fatalf("incorrect release: %+v", state)
			}
			if request != original || state.stop != predecessor.stop || state.policy != predecessor.policy || !reflect.DeepEqual(state.owners, predecessor.owners) {
				t.Fatal("mutated request or unrelated state")
			}
			old, err := encodeAdmissionRecord(tx.state)
			if err != nil || string(old) != string(before) {
				t.Fatal("mutated predecessor")
			}
			if got, err := tx.release(context.Background(), request); err == nil || got != (transactionReleaseOutcome{}) {
				t.Fatalf("same-handle retry: %+v %v", got, err)
			}
			admission := transactionAdmissionRequest{binding: request.binding, reservation: ownedMemoryReservation{id: "replacement", owner: request.release.owner, amount: 1}}
			if got, err := tx.admit(context.Background(), admission, transactionMeasurement(admission)); err == nil || !reflect.DeepEqual(got, transactionAdmissionOutcome{}) {
				t.Fatalf("release handle admitted work: %+v %v", got, err)
			}
			if err := tx.close(); err != nil {
				t.Fatal(err)
			}
			next, err := beginAdmissionTransaction(context.Background(), dir, request.binding)
			if err != nil {
				t.Fatal(err)
			}
			defer next.close()
			if err := next.observe(context.Background(), request.release.owner); err != nil {
				t.Fatal(err)
			}
			got, err := next.release(context.Background(), releaseRequest(next, request.binding, request.release.id))
			if err != nil || got.disposition != releaseNotFound {
				t.Fatalf("idempotent absence: %+v %v", got, err)
			}
			_, again := releaseReadState(t, dir)
			againUsed, _ := again.ledger.validatedUsage()
			if againUsed != afterUsed || !reflect.DeepEqual(again.ledger, state.ledger) {
				t.Fatal("retry released more capacity")
			}
		})
	}
}

func TestReleaseRefusesUncertainInputs(t *testing.T) {
	for _, name := range []string{"wrong-owner", "stale-owner", "held-by-other", "empty-id", "invalid-id", "binding", "no-proof", "proof-owner", "proof-id", "proof-predecessor", "proof-binding", "state", "intent", "unobserved", "cancelled", "nil-context", "lock"} {
		t.Run(name, func(t *testing.T) {
			tx, request := transactionReleaseFixture(t, false)
			dir := tx.directory.Name()
			ctx := context.Background()
			switch name {
			case "wrong-owner":
				request.release.owner.name = "other"
			case "stale-owner":
				request.release.owner.incarnation = "stale"
			case "held-by-other":
				tx.owner = tx.state.owners[0]
				request = releaseRequest(tx, request.binding, request.release.id)
				intent, err := encodeAdmissionIntent(admissionIntent{version: admissionRecordVersion, phase: intentObserving, boot: tx.state.boot, owner: tx.owner, predecessor: tx.digest})
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "intent"), intent, 0600); err != nil {
					t.Fatal(err)
				}
			case "empty-id":
				request.release.id = ""
			case "invalid-id":
				request.release.id = "bad/id"
			case "binding":
				request.binding.boot = "other"
			case "no-proof":
				request.proof = reservationQuiescenceProof{}
			case "proof-owner":
				request.proof.release.owner.incarnation = "stale"
			case "proof-id":
				request.proof.release.id = "other"
			case "proof-predecessor":
				request.proof.predecessor = sha256.Sum256([]byte("other"))
			case "proof-binding":
				request.proof.binding.limit++
			case "state":
				_, state := releaseReadState(t, dir)
				state.stop.generation++
				data, err := encodeAdmissionRecord(state)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "state"), data, 0600); err != nil {
					t.Fatal(err)
				}
			case "intent":
				if err := os.WriteFile(filepath.Join(dir, "intent"), []byte("bad"), 0600); err != nil {
					t.Fatal(err)
				}
			case "unobserved":
				tx.ready = false
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "nil-context":
				ctx = nil
			case "lock":
				if err := os.Rename(filepath.Join(dir, "lock"), filepath.Join(dir, "old-lock")); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "lock"), nil, 0600); err != nil {
					t.Fatal(err)
				}
			}
			before, _ := releaseReadState(t, dir)
			got, err := tx.release(ctx, request)
			if err == nil || got != (transactionReleaseOutcome{}) {
				t.Fatalf("unsafe release: %+v %v", got, err)
			}
			after, _ := releaseReadState(t, dir)
			if string(before) != string(after) {
				t.Fatal("failure released capacity")
			}
			if _, err := os.Stat(filepath.Join(dir, "intent")); err != nil {
				t.Fatal("lost fence")
			}
		})
	}
}

func TestReleaseAbsent(t *testing.T) {
	tx, request := transactionReleaseFixture(t, false)
	request = releaseRequest(tx, request.binding, "absent")
	_, before := releaseReadState(t, tx.directory.Name())
	got, err := tx.release(context.Background(), request)
	if err != nil || got.disposition != releaseNotFound {
		t.Fatalf("absence: %+v %v", got, err)
	}
	_, after := releaseReadState(t, tx.directory.Name())
	if !reflect.DeepEqual(before.ledger, after.ledger) {
		t.Fatal("absent release changed capacity")
	}
}

func TestReleasePendingIntentsHoldExactCapacity(t *testing.T) {
	for _, name := range []string{"intent", "intent-prepared", "state-next", "retiring"} {
		t.Run(name, func(t *testing.T) {
			dir, binding := transactionFixture(t)
			before, state := releaseReadState(t, dir)
			intent, err := encodeAdmissionIntent(admissionIntent{version: admissionRecordVersion, phase: intentObserving, boot: binding.boot, owner: state.owners[0], predecessor: sha256.Sum256(before)})
			if err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(filepath.Join(dir, name), intent, 0600); err != nil {
				t.Fatal(err)
			}
			if tx, err := beginAdmissionTransaction(context.Background(), dir, binding); tx != nil || !errors.Is(err, errAdmissionPending) {
				t.Fatalf("pending release entry: %v", err)
			}
			after, _ := releaseReadState(t, dir)
			if string(before) != string(after) {
				t.Fatal("pending intent changed capacity")
			}
		})
	}
}

func TestReleasePublishFaults(t *testing.T) {
	for _, point := range releasePublishPoints {
		t.Run(point, func(t *testing.T) {
			tx, request := transactionReleaseFixture(t, true)
			dir := tx.directory.Name()
			before, state := releaseReadState(t, dir)
			injected := errors.New("release publication fault")
			tx.publish = func(at string) error {
				if at == point {
					return injected
				}
				return nil
			}
			got, err := tx.release(context.Background(), request)
			if !errors.Is(err, injected) || got != (transactionReleaseOutcome{}) {
				t.Fatalf("hidden success: %+v %v", got, err)
			}
			after, current := releaseReadState(t, dir)
			if point == "prepared-write" || point == "prepared-sync" || point == "prepared-close" || point == "prepared-rename" || point == "prepared-directory" || point == "state-write" || point == "state-sync" || point == "state-close" || point == "state-rename" {
				if string(before) != string(after) {
					t.Fatal("pre-replacement failure released capacity")
				}
			} else {
				used, _ := current.ledger.validatedUsage()
				if used != state.ledger.limit-1 {
					t.Fatal("wrong amount staged")
				}
			}
			if err := tx.close(); err != nil {
				t.Fatal(err)
			}
			if next, err := beginAdmissionTransaction(context.Background(), dir, request.binding); next != nil || !errors.Is(err, errAdmissionPending) {
				t.Fatalf("failed release made capacity available: %v", err)
			}
		})
	}
}

func TestReleaseCancelledAndReplacedLockDuringPublication(t *testing.T) {
	for _, name := range []string{"cancel", "lock"} {
		t.Run(name, func(t *testing.T) {
			tx, request := transactionReleaseFixture(t, false)
			dir := tx.directory.Name()
			before, _ := releaseReadState(t, dir)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			tx.publish = func(at string) error {
				if at == "state-write" {
					if name == "cancel" {
						cancel()
					} else {
						if err := os.Rename(filepath.Join(dir, "lock"), filepath.Join(dir, "old-lock")); err != nil {
							return err
						}
						return os.WriteFile(filepath.Join(dir, "lock"), nil, 0600)
					}
				}
				return nil
			}
			got, err := tx.release(ctx, request)
			if err == nil || got != (transactionReleaseOutcome{}) {
				t.Fatalf("unsafe publication: %+v %v", got, err)
			}
			after, _ := releaseReadState(t, dir)
			if string(before) != string(after) {
				t.Fatal("unsafe publication released capacity")
			}
		})
	}
}

func TestReleaseSubprocess(t *testing.T) {
	dir := os.Getenv("AZAZEL_RELEASE_FIXTURE")
	if dir == "" {
		return
	}
	state := recordExample()
	binding := admissionBinding{boot: state.boot, policy: state.policy, limit: state.ledger.limit}
	tx, err := beginAdmissionTransaction(context.Background(), dir, binding)
	if err != nil {
		fmt.Println("REFUSED", err)
		os.Exit(21)
	}
	defer tx.close()
	if err := tx.observe(context.Background(), state.owners[0]); err != nil {
		fmt.Println("REFUSED", err)
		os.Exit(22)
	}
	point := os.Getenv("AZAZEL_RELEASE_POINT")
	tx.publish = func(at string) error {
		if at == point {
			fmt.Println("PUBLISH", at)
			_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
		}
		return nil
	}
	got, err := tx.release(context.Background(), releaseRequest(tx, binding, "a"))
	if err != nil {
		fmt.Println("REFUSED", err)
		os.Exit(23)
	}
	fmt.Println("OUTCOME", got.disposition)
}

func TestReleaseKilledPublishPoints(t *testing.T) {
	for _, point := range releasePublishPoints {
		t.Run(point, func(t *testing.T) {
			dir, binding := transactionFixture(t)
			before, _ := releaseReadState(t, dir)
			cmd := exec.Command(os.Args[0], "-test.run=^TestReleaseSubprocess$")
			cmd.Env = append(os.Environ(), "AZAZEL_RELEASE_FIXTURE="+dir, "AZAZEL_RELEASE_POINT="+point)
			stdout, err := cmd.StdoutPipe()
			if err != nil {
				t.Fatal(err)
			}
			stdin, err := cmd.StdinPipe()
			if err != nil {
				t.Fatal(err)
			}
			if err := cmd.Start(); err != nil {
				t.Fatal(err)
			}
			t.Cleanup(func() { _ = stdin.Close(); _ = cmd.Process.Kill(); _ = cmd.Wait() })
			timer := time.AfterFunc(8*time.Second, func() { _ = cmd.Process.Kill() })
			defer timer.Stop()
			line, err := bufio.NewReader(stdout).ReadString('\n')
			if err != nil || strings.TrimSpace(line) != "PUBLISH "+point {
				t.Fatalf("child: %q %v", line, err)
			}
			if other, err := beginAdmissionTransaction(context.Background(), dir, binding); other != nil || !errors.Is(err, errAdmissionBusy) {
				t.Fatalf("release lost lock: %v", err)
			}
			if err := cmd.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			_ = cmd.Wait()
			other, err := beginAdmissionTransaction(context.Background(), dir, binding)
			if point == "complete-directory" {
				if err != nil {
					t.Fatal(err)
				}
				if err := other.close(); err != nil {
					t.Fatal(err)
				}
				_, state := releaseReadState(t, dir)
				used, _ := state.ledger.validatedUsage()
				if used != 7 {
					t.Fatal("acknowledgement boundary has incorrect release")
				}
			} else if other != nil || !errors.Is(err, errAdmissionPending) {
				t.Fatalf("killed release made capacity available: %v", err)
			}
			after, current := releaseReadState(t, dir)
			switch point {
			case "prepared-write", "prepared-sync", "prepared-close", "prepared-rename", "prepared-directory", "state-write", "state-sync", "state-close", "state-rename":
				if string(before) != string(after) {
					t.Fatal("killed before replacement changed capacity")
				}
			default:
				used, _ := current.ledger.validatedUsage()
				if used != 7 {
					t.Fatal("killed release staged wrong capacity")
				}
			}
		})
	}
}
