package budget

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"golang.org/x/sys/unix"
)

// fakeHost is a host adapter whose answers each test sets. A nil field gives
// the healthy answer used by sampleRaw.
type fakeHost struct {
	sysctl func(key string, call int) ([]byte, error)
	statfs func() (unix.Statfs_t, error)
	now    func(call int) time.Time
	calls  map[string]int
	clock  int
}

// SysctlRaw answers a read of key, counting reads per key.
func (host *fakeHost) SysctlRaw(key string) ([]byte, error) {
	if host.calls == nil {
		host.calls = map[string]int{}
	}
	host.calls[key]++
	if host.sysctl != nil {
		if data, err := host.sysctl(key, host.calls[key]); data != nil || err != nil {
			return data, err
		}
	}
	return sampleRaw(key), nil
}

// Statfs answers a file system query with a healthy disk unless set.
func (host *fakeHost) Statfs(int) (unix.Statfs_t, error) {
	if host.statfs != nil {
		return host.statfs()
	}
	return unix.Statfs_t{Bsize: 4096, Blocks: 100, Bfree: 30, Bavail: 20}, nil
}

// Now answers the clock, counting calls.
func (host *fakeHost) Now() time.Time {
	host.clock++
	if host.now != nil {
		return host.now(host.clock)
	}
	return time.Unix(1000, 0)
}

// fakeBinding is a binding that matches sampleBoot and accepts the healthy
// readings.
func fakeBinding() admissionBinding {
	return admissionBinding{boot: sampleBoot, policy: hostPolicy{loadLimit: 10, swapLimit: 4096, diskFloor: 1}, limit: 100}
}

// openFakeDir opens a temporary directory for sampling.
func openFakeDir(t *testing.T) *os.File {
	t.Helper()
	dir, err := os.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { dir.Close() })
	return dir
}

// TestSampleFakeHostFailures checks that a failure at each step of a sample
// is returned as it is, with no measurement.
func TestSampleFakeHostFailures(t *testing.T) {
	broken := errors.New("read failed")
	failKey := func(key string, call int) func(*fakeHost) {
		return func(host *fakeHost) {
			host.sysctl = func(k string, n int) ([]byte, error) {
				if k == key && n == call {
					return nil, broken
				}
				return nil, nil
			}
		}
	}
	cases := map[string]func(*fakeHost){
		"first boot read": failKey("kern.bootsessionuuid", 1),
		"load read":       failKey("vm.loadavg", 1),
		"swap read":       failKey("vm.swapusage", 1),
		"pressure read":   failKey("kern.memorystatus_vm_pressure_level", 1),
		"final boot read": failKey("kern.bootsessionuuid", 2),
		"disk query":      func(host *fakeHost) { host.statfs = func() (unix.Statfs_t, error) { return unix.Statfs_t{}, broken } },
	}
	for name, setup := range cases {
		host := &fakeHost{}
		setup(host)
		got, err := sampleAdmissionHost(context.Background(), openFakeDir(t), fakeBinding(), host)
		if !errors.Is(err, broken) || got != (admissionMeasurement{}) {
			t.Errorf("%s: %+v %v", name, got, err)
		}
	}
}

// TestSampleFakeHostBadArguments checks that each unusable argument is
// refused before the host is read.
func TestSampleFakeHostBadArguments(t *testing.T) {
	dir := openFakeDir(t)
	host := &fakeHost{}
	var noContext context.Context
	noLimit := fakeBinding()
	noLimit.limit = 0
	badPolicy := fakeBinding()
	badPolicy.policy.loadLimit = 0
	cases := map[string]func() (admissionMeasurement, error){
		"nil context": func() (admissionMeasurement, error) {
			return sampleAdmissionHost(noContext, dir, fakeBinding(), host)
		},
		"nil output": func() (admissionMeasurement, error) {
			return sampleAdmissionHost(context.Background(), nil, fakeBinding(), host)
		},
		"nil adapter": func() (admissionMeasurement, error) {
			return sampleAdmissionHost(context.Background(), dir, fakeBinding(), nil)
		},
		"no limit": func() (admissionMeasurement, error) {
			return sampleAdmissionHost(context.Background(), dir, noLimit, host)
		},
		"invalid policy": func() (admissionMeasurement, error) {
			return sampleAdmissionHost(context.Background(), dir, badPolicy, host)
		},
	}
	for name, call := range cases {
		if got, err := call(); !errors.Is(err, errAdmissionUnsafe) || got != (admissionMeasurement{}) {
			t.Errorf("%s: %+v %v", name, got, err)
		}
	}
	if len(host.calls) != 0 || host.clock != 0 {
		t.Errorf("host was read: %v, %d clock reads", host.calls, host.clock)
	}
}

// TestSampleFakeHostCancelledLate checks that a context that ends while the
// disk is queried, or while the closing time is read, is returned.
func TestSampleFakeHostCancelledLate(t *testing.T) {
	t.Run("during disk query", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		host := &fakeHost{statfs: func() (unix.Statfs_t, error) {
			cancel()
			return unix.Statfs_t{Bsize: 4096, Blocks: 100, Bfree: 30, Bavail: 20}, nil
		}}
		if _, err := sampleAdmissionHost(ctx, openFakeDir(t), fakeBinding(), host); !errors.Is(err, context.Canceled) {
			t.Errorf("got %v", err)
		}
	})
	t.Run("during closing time", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		host := &fakeHost{now: func(call int) time.Time {
			if call == 2 {
				cancel()
			}
			return time.Unix(1000, 0)
		}}
		if _, err := sampleAdmissionHost(ctx, openFakeDir(t), fakeBinding(), host); !errors.Is(err, context.Canceled) {
			t.Errorf("got %v", err)
		}
	})
}

// TestAdmissionHostReadCancelledFirst checks that a context that is done
// before the read stops it without asking the host.
func TestAdmissionHostReadCancelledFirst(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	host := &fakeHost{}
	if _, err := admissionHostRead(ctx, host, "vm.loadavg"); !errors.Is(err, context.Canceled) || len(host.calls) != 0 {
		t.Errorf("got %v after %v calls", err, host.calls)
	}
}

// TestAdmissionBootRefusesBadDigit checks that a letter outside hex is
// refused in the boot id, while a good one is returned whole.
func TestAdmissionBootRefusesBadDigit(t *testing.T) {
	bad := []byte(sampleBoot)
	bad[0] = 'G'
	host := &fakeHost{sysctl: func(string, int) ([]byte, error) { return append(bad, 0), nil }}
	if _, err := admissionBoot(context.Background(), host); !errors.Is(err, errAdmissionUnsafe) {
		t.Errorf("bad digit: %v", err)
	}
	if got, err := admissionBoot(context.Background(), &fakeHost{}); err != nil || got != sampleBoot {
		t.Errorf("good boot: %q %v", got, err)
	}
}

// TestNativeAdmissionBootRefusesNilContext checks that a missing context is
// refused before the real system is read.
func TestNativeAdmissionBootRefusesNilContext(t *testing.T) {
	var noContext context.Context
	if got, err := nativeAdmissionBoot(noContext); got != "" || !errors.Is(err, errAdmissionUnsafe) {
		t.Errorf("got %q, %v", got, err)
	}
}
