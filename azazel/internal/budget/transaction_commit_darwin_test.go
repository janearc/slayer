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

func transactionCommitFixture(t *testing.T) (*admissionTransaction, transactionAdmissionRequest) {
	t.Helper()
	dir, binding := transactionFixture(t)
	data, err := os.ReadFile(filepath.Join(dir, "state"))
	if err != nil {
		t.Fatal(err)
	}
	state, err := decodeAdmissionRecord(data)
	if err != nil {
		t.Fatal(err)
	}
	state.stop = stopState{generation: 3}
	data, err = encodeAdmissionRecord(state)
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
	owner := state.owners[0]
	if err := tx.observe(context.Background(), owner); err != nil {
		t.Fatal(err)
	}
	return tx, transactionAdmissionRequest{binding: binding, reservation: ownedMemoryReservation{id: "new", owner: owner, amount: 9}}
}

func transactionMeasurement(request transactionAdmissionRequest) func(context.Context) (admissionMeasurement, error) {
	return func(context.Context) (admissionMeasurement, error) {
		now := time.Now()
		return admissionMeasurement{binding: request.binding, taken: now, completed: now, pressure: 1, sample: hostSample{loadKnown: true, swapKnown: true, diskKnown: true, freeDisk: 30}}, nil
	}
}

func TestTransactionCommitSubprocess(t *testing.T) {
	dir := os.Getenv("AZAZEL_COMMIT_FIXTURE")
	if dir == "" {
		return
	}
	state := recordExample()
	binding := admissionBinding{boot: state.boot, policy: state.policy, limit: state.ledger.limit}
	tx, err := beginAdmissionTransaction(context.Background(), dir, binding)
	if err != nil {
		fmt.Println("REFUSED", err)
		os.Exit(14)
	}
	defer tx.close()
	if err := tx.observe(context.Background(), state.owners[0]); err != nil {
		fmt.Println("REFUSED", err)
		os.Exit(15)
	}
	point := os.Getenv("AZAZEL_COMMIT_POINT")
	tx.publish = func(at string) error {
		if at == point {
			fmt.Println("PUBLISH", at)
			_, _ = bufio.NewReader(os.Stdin).ReadString('\n')
		}
		return nil
	}
	request := transactionAdmissionRequest{binding: binding, reservation: ownedMemoryReservation{id: "new", owner: state.owners[0], amount: 9}}
	result, err := tx.admit(context.Background(), request, transactionMeasurement(request))
	if err != nil {
		fmt.Println("REFUSED", err)
		os.Exit(16)
	}
	fmt.Println("OUTCOME", result.disposition)
}

func TestTransactionKilledPublishPoints(t *testing.T) {
	for _, point := range []string{"prepared-write", "prepared-sync", "prepared-close", "prepared-rename", "prepared-directory", "state-write", "state-sync", "state-close", "state-rename", "state-directory", "retiring-write", "retiring-sync", "retiring-close", "retiring-directory", "intent-remove", "retired-directory", "retiring-remove", "complete-directory"} {
		t.Run(point, func(t *testing.T) {
			tx, request := transactionCommitFixture(t)
			dir := tx.directory.Name()
			if err := tx.close(); err != nil {
				t.Fatal(err)
			}
			if err := os.Remove(filepath.Join(dir, "intent")); err != nil {
				t.Fatal(err)
			}
			cmd := exec.Command(os.Args[0], "-test.run=^TestTransactionCommitSubprocess$")
			cmd.Env = append(os.Environ(), "AZAZEL_COMMIT_FIXTURE="+dir, "AZAZEL_COMMIT_POINT="+point)
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
			if other, err := beginAdmissionTransaction(context.Background(), dir, request.binding); other != nil || !errors.Is(err, errAdmissionBusy) {
				t.Fatalf("publish lock lost: %v", err)
			}
			if err := cmd.Process.Kill(); err != nil {
				t.Fatal(err)
			}
			_ = cmd.Wait()
			other, err := beginAdmissionTransaction(context.Background(), dir, request.binding)
			if point == "complete-directory" {
				if err != nil {
					t.Fatal(err)
				}
				if err := other.close(); err != nil {
					t.Fatal(err)
				}
			} else if other != nil || !errors.Is(err, errAdmissionPending) {
				t.Fatalf("uncertainty lost: %v", err)
			}
		})
	}
}

func TestTransactionSampleMetadata(t *testing.T) {
	for _, scenario := range []string{"missing-completed", "completed-future", "completed-before", "pressure-unknown", "pressure-warning", "cancel-during"} {
		t.Run(scenario, func(t *testing.T) {
			tx, request := transactionCommitFixture(t)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			base := transactionMeasurement(request)
			_, err := tx.admit(ctx, request, func(ctx context.Context) (admissionMeasurement, error) {
				measured, err := base(ctx)
				switch scenario {
				case "missing-completed":
					measured.completed = time.Time{}
				case "completed-future":
					measured.completed = measured.taken.Add(time.Hour)
				case "completed-before":
					measured.completed = measured.taken.Add(-time.Nanosecond)
				case "pressure-unknown":
					measured.pressure = 0
				case "pressure-warning":
					measured.pressure = 2
				case "cancel-during":
					cancel()
				}
				return measured, err
			})
			if err == nil {
				t.Fatal("accepted unsafe sample metadata")
			}
		})
	}
}

func TestTransactionDurableAdmission(t *testing.T) {
	tx, request := transactionCommitFixture(t)
	before, err := encodeAdmissionRecord(tx.state)
	if err != nil {
		t.Fatal(err)
	}
	original := request
	result, err := tx.admit(context.Background(), request, transactionMeasurement(request))
	if err != nil || result.disposition != admissionProposed || result.reservation != reservationProposed {
		t.Fatalf("result: %+v %v", result, err)
	}
	if request != original {
		t.Fatal("mutated request")
	}
	if _, err := os.Stat(filepath.Join(tx.directory.Name(), "intent")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("intent not retired: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(tx.directory.Name(), "state"))
	if err != nil {
		t.Fatal(err)
	}
	state, err := decodeAdmissionRecord(data)
	if err != nil || state.predecessor != sha256.Sum256(before) || len(state.ledger.reservations) != 3 {
		t.Fatalf("state: %+v %v", state, err)
	}
	unchanged, encodeErr := encodeAdmissionRecord(tx.state)
	if encodeErr != nil || string(unchanged) != string(before) {
		t.Fatal("mutated predecessor")
	}
	if again, err := tx.admit(context.Background(), request, transactionMeasurement(request)); err == nil || !reflect.DeepEqual(again, transactionAdmissionOutcome{}) {
		t.Fatalf("repeat success: %+v %v", again, err)
	}
}

func TestTransactionPublishFaults(t *testing.T) {
	points := []string{"prepared-write", "prepared-sync", "prepared-close", "prepared-rename", "prepared-directory", "state-write", "state-sync", "state-close", "state-rename", "state-directory", "retiring-write", "retiring-sync", "retiring-close", "retiring-directory", "intent-remove", "retired-directory", "retiring-remove", "complete-directory"}
	for _, point := range points {
		t.Run(point, func(t *testing.T) {
			tx, request := transactionCommitFixture(t)
			injected := errors.New("publish fault")
			seen := false
			tx.publish = func(at string) error {
				if at == point {
					seen = true
					return injected
				}
				return nil
			}
			result, err := tx.admit(context.Background(), request, transactionMeasurement(request))
			if !seen || !errors.Is(err, injected) || !reflect.DeepEqual(result, transactionAdmissionOutcome{}) {
				t.Fatalf("hidden success: %t %+v %v", seen, result, err)
			}
			dir := tx.directory.Name()
			if _, err := os.Stat(filepath.Join(dir, "intent")); err != nil {
				t.Fatalf("lost uncertainty: %v", err)
			}
			if err := tx.close(); err != nil {
				t.Fatal(err)
			}
			if next, err := beginAdmissionTransaction(context.Background(), dir, request.binding); next != nil || !errors.Is(err, errAdmissionPending) {
				t.Fatalf("reopened uncertain transition: %v", err)
			}
		})
	}
}

func TestTransactionAdmissionRefusesUncertainInputs(t *testing.T) {
	for _, name := range []string{"unknown", "stale", "future", "boot", "policy", "owner", "intent", "state", "cancelled", "measurement-error", "unobserved"} {
		t.Run(name, func(t *testing.T) {
			tx, request := transactionCommitFixture(t)
			measure := transactionMeasurement(request)
			ctx := context.Background()
			switch name {
			case "unknown", "stale", "future", "boot", "policy":
				base := measure
				measure = func(ctx context.Context) (admissionMeasurement, error) {
					m, err := base(ctx)
					switch name {
					case "unknown":
						m.sample.loadKnown = false
					case "stale":
						m.taken = time.Now().Add(-time.Hour)
					case "future":
						m.taken = time.Now().Add(time.Hour)
					case "boot":
						m.binding.boot = "other"
					case "policy":
						m.binding.policy.loadLimit++
					}
					return m, err
				}
			case "owner":
				request.reservation.owner = tx.state.owners[0]
				if request.reservation.owner == tx.owner {
					request.reservation.owner = tx.state.owners[1]
				}
			case "intent":
				intent := admissionIntent{version: 1, phase: intentObserving, boot: tx.state.boot, owner: tx.state.owners[0], predecessor: tx.digest}
				if intent.owner == tx.owner {
					intent.owner = tx.state.owners[1]
				}
				data, err := encodeAdmissionIntent(intent)
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(tx.directory.Name(), "intent"), data, 0600); err != nil {
					t.Fatal(err)
				}
			case "state":
				if err := os.WriteFile(filepath.Join(tx.directory.Name(), "state"), []byte("bad"), 0600); err != nil {
					t.Fatal(err)
				}
			case "cancelled":
				var cancel context.CancelFunc
				ctx, cancel = context.WithCancel(ctx)
				cancel()
			case "measurement-error":
				measure = func(context.Context) (admissionMeasurement, error) {
					return admissionMeasurement{}, errors.New("measurement failed")
				}
			case "unobserved":
				tx.ready = false
			}
			before, err := os.ReadFile(filepath.Join(tx.directory.Name(), "state"))
			if err != nil {
				t.Fatal(err)
			}
			result, err := tx.admit(ctx, request, measure)
			if err == nil || !reflect.DeepEqual(result, transactionAdmissionOutcome{}) {
				t.Fatalf("uncertain success: %+v %v", result, err)
			}
			after, err := os.ReadFile(filepath.Join(tx.directory.Name(), "state"))
			if err != nil || string(before) != string(after) {
				t.Fatal("mutated state")
			}
			if _, err := os.Stat(filepath.Join(tx.directory.Name(), "intent")); err != nil {
				t.Fatal("lost fence")
			}
		})
	}
}

func TestTransactionAdmissionHoldsAndStop(t *testing.T) {
	for _, name := range []string{"disk", "load", "capacity", "duplicate", "stopped"} {
		t.Run(name, func(t *testing.T) {
			tx, request := transactionCommitFixture(t)
			measure := transactionMeasurement(request)
			want := admissionReservationHeld
			switch name {
			case "disk", "load":
				base := measure
				measure = func(ctx context.Context) (admissionMeasurement, error) {
					m, err := base(ctx)
					if name == "disk" {
						m.sample.freeDisk = 1
					} else {
						m.sample.load = 10
					}
					return m, err
				}
				want = admissionHostHeld
				if name == "disk" {
					want = admissionStopped
				}
			case "capacity":
				request.reservation.amount = 100
			case "duplicate":
				request.reservation.id = tx.state.ledger.reservations[0].id
			case "stopped":
				tx.state.stop.stopped = true
				data, err := encodeAdmissionRecord(tx.state)
				if err != nil {
					t.Fatal(err)
				}
				tx.digest = sha256.Sum256(data)
				if err := os.WriteFile(filepath.Join(tx.directory.Name(), "state"), data, 0600); err != nil {
					t.Fatal(err)
				}
				intent, err := encodeAdmissionIntent(admissionIntent{version: 1, phase: intentObserving, boot: tx.state.boot, owner: tx.owner, predecessor: tx.digest})
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(tx.directory.Name(), "intent"), intent, 0600); err != nil {
					t.Fatal(err)
				}
				want = admissionStopped
			}
			result, err := tx.admit(context.Background(), request, measure)
			if err != nil || result.disposition != want {
				t.Fatalf("hold: %+v %v", result, err)
			}
			data, err := os.ReadFile(filepath.Join(tx.directory.Name(), "state"))
			if err != nil {
				t.Fatal(err)
			}
			state, err := decodeAdmissionRecord(data)
			if err != nil || len(state.ledger.reservations) != 2 || name == "disk" && (!state.stop.stopped || state.stop.generation != 4) {
				t.Fatalf("hold state: %+v %v", state, err)
			}
		})
	}
}
