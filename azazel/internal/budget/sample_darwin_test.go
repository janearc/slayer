package budget

import (
	"context"
	"encoding/binary"
	"errors"
	"math"
	"os"
	"path/filepath"
	"testing"
	"time"

	"go.uber.org/mock/gomock"
	"golang.org/x/sys/unix"
)

const sampleBoot = "01234567-89AB-CDEF-0123-456789ABCDEF"

func sampleRaw(key string) []byte {
	switch key {
	case "kern.bootsessionuuid":
		return append([]byte(sampleBoot), 0)
	case "vm.loadavg":
		data := make([]byte, 24)
		binary.NativeEndian.PutUint32(data, 100)
		binary.NativeEndian.PutUint32(data[4:], 200)
		binary.NativeEndian.PutUint32(data[8:], 300)
		binary.NativeEndian.PutUint64(data[16:], 100)
		return data
	case "vm.swapusage":
		data := make([]byte, 32)
		binary.NativeEndian.PutUint64(data, 4096)
		binary.NativeEndian.PutUint64(data[8:], 3072)
		binary.NativeEndian.PutUint64(data[16:], 1024)
		binary.NativeEndian.PutUint32(data[24:], 4096)
		return data
	case "kern.memorystatus_vm_pressure_level":
		data := make([]byte, 4)
		binary.NativeEndian.PutUint32(data, 1)
		return data
	}
	panic(key)
}

func TestSampleNativeAdapter(t *testing.T) {
	for _, scenario := range []string{"ready", "boot", "boot-size", "boot-format", "changed-boot", "load-size", "load-scale", "load-high-bit-first", "load-high-bit-second", "load-high-bit-third", "swap-size", "swap-overflow", "swap-page", "pressure-unknown", "pressure-size", "pressure-warning", "pressure-critical", "disk-zero", "disk-overflow", "disk-inconsistent", "sysctl-error", "disk-error", "cancelled", "cancel-during", "stale", "age-boundary", "zero-time", "future", "closed", "regular", "renamed"} {
		t.Run(scenario, func(t *testing.T) {
			dir := t.TempDir()
			output, err := os.Open(dir)
			if err != nil {
				t.Fatal(err)
			}
			defer output.Close()
			if scenario == "closed" {
				_ = output.Close()
			}
			if scenario == "regular" {
				f, err := os.Create(filepath.Join(dir, "file"))
				if err != nil {
					t.Fatal(err)
				}
				defer f.Close()
				output = f
			}
			if scenario == "renamed" {
				if err := os.Rename(dir, dir+"-moved"); err != nil {
					t.Fatal(err)
				}
				defer os.RemoveAll(dir + "-moved")
			}
			adapter := NewMockadmissionHostAdapter(gomock.NewController(t))
			now := time.Now()
			clockCalls := 0
			adapter.EXPECT().Now().DoAndReturn(func() time.Time {
				clockCalls++
				if scenario == "zero-time" {
					return time.Time{}
				}
				if clockCalls > 1 && scenario == "age-boundary" {
					return now.Add(admissionSampleAge)
				}
				if clockCalls > 1 && scenario == "stale" {
					return now.Add(admissionSampleAge + time.Nanosecond)
				}
				if clockCalls > 1 && scenario == "future" {
					return now.Add(-time.Nanosecond)
				}
				return now
			}).AnyTimes()
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			if scenario == "cancelled" {
				cancel()
			}
			bootCalls := 0
			adapter.EXPECT().SysctlRaw(gomock.Any()).DoAndReturn(func(key string) ([]byte, error) {
				data := sampleRaw(key)
				if scenario == "cancel-during" {
					cancel()
				}
				if scenario == "sysctl-error" {
					return nil, unix.EIO
				}
				switch key {
				case "kern.bootsessionuuid":
					bootCalls++
					if scenario == "boot-size" {
						return data[:36], nil
					}
					if scenario == "boot-format" {
						data[8] = '_'
					}
					if scenario == "boot" || scenario == "changed-boot" && bootCalls > 1 {
						data[0] = 'F'
					}
				case "vm.loadavg":
					if scenario == "load-size" {
						return data[:15], nil
					}
					if scenario == "load-scale" {
						binary.NativeEndian.PutUint64(data[16:], 0)
					}
					if scenario == "load-high-bit-first" {
						binary.NativeEndian.PutUint32(data, math.MaxUint32)
					}
					if scenario == "load-high-bit-second" {
						binary.NativeEndian.PutUint32(data[4:], 1<<31)
					}
					if scenario == "load-high-bit-third" {
						binary.NativeEndian.PutUint32(data[8:], math.MaxUint32)
					}
				case "vm.swapusage":
					if scenario == "swap-size" {
						return data[:31], nil
					}
					if scenario == "swap-overflow" {
						binary.NativeEndian.PutUint64(data[16:], math.MaxUint64)
					}
					if scenario == "swap-page" {
						binary.NativeEndian.PutUint32(data[24:], 0)
					}
				case "kern.memorystatus_vm_pressure_level":
					if scenario == "pressure-size" {
						return data[:3], nil
					}
					if scenario == "pressure-unknown" {
						binary.NativeEndian.PutUint32(data, 3)
					}
					if scenario == "pressure-warning" {
						binary.NativeEndian.PutUint32(data, 2)
					}
					if scenario == "pressure-critical" {
						binary.NativeEndian.PutUint32(data, 4)
					}
				}
				return data, nil
			}).AnyTimes()
			adapter.EXPECT().Statfs(int(output.Fd())).DoAndReturn(func(int) (unix.Statfs_t, error) {
				stat := unix.Statfs_t{Bsize: 4096, Blocks: 100, Bfree: 30, Bavail: 20}
				if scenario == "disk-error" {
					return stat, unix.EIO
				}
				if scenario == "disk-zero" {
					stat.Bsize = 0
				}
				if scenario == "disk-overflow" {
					stat.Blocks = math.MaxUint64
					stat.Bfree = math.MaxUint64
					stat.Bavail = math.MaxUint64
				}
				if scenario == "disk-inconsistent" {
					stat.Bavail = 101
				}
				return stat, nil
			}).AnyTimes()
			binding := admissionBinding{boot: sampleBoot, policy: hostPolicy{loadLimit: 10, swapLimit: 4096, diskFloor: 1}, limit: 100}
			measured, err := sampleAdmissionHost(ctx, output, binding, adapter)
			good := scenario == "ready" || scenario == "renamed" || scenario == "age-boundary" || scenario == "load-high-bit-first" || scenario == "load-high-bit-second" || scenario == "load-high-bit-third"
			expectedLoad, expectedDecision := 1.0, hostReady
			if scenario == "load-high-bit-first" {
				expectedLoad, expectedDecision = float64(uint32(math.MaxUint32))/100, hostLoadHigh
			}
			if good {
				if err != nil || measured.binding != binding || measured.taken != now || measured.completed != now.Add(func() time.Duration {
					if scenario == "age-boundary" {
						return admissionSampleAge
					}
					return 0
				}()) || measured.pressure != 1 || measured.sample.load != expectedLoad || measured.sample.swap != 1024 || measured.sample.freeDisk != 81920 || binding.policy.assess(measured.sample) != expectedDecision {
					t.Fatalf("measurement: %+v %v", measured, err)
				}
			} else if err == nil || measured != (admissionMeasurement{}) {
				t.Fatalf("unsafe sample: %+v %v", measured, err)
			}
			if scenario == "cancelled" || scenario == "cancel-during" {
				if !errors.Is(err, context.Canceled) {
					t.Fatalf("cancellation lost: %v", err)
				}
			}
		})
	}
}

func TestSampleDarwinIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	boot, err := nativeAdmissionBoot(ctx)
	if err != nil {
		t.Fatal(err)
	}
	output, err := os.Open(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer output.Close()
	binding := admissionBinding{boot: boot, policy: hostPolicy{loadLimit: 10000, swapLimit: math.MaxUint64, diskFloor: 1}, limit: 1}
	measured, err := measureAdmissionHost(ctx, output, binding)
	if errors.Is(err, errAdmissionPressure) {
		t.Log("native pressure holds admission")
		return
	}
	if err != nil {
		for _, key := range []string{"vm.loadavg", "vm.swapusage", "kern.memorystatus_vm_pressure_level"} {
			data, readErr := unix.SysctlRaw(key)
			t.Logf("%s: %x %v", key, data, readErr)
		}
		t.Fatal(err)
	}
	if measured.binding.boot != boot || !measured.sample.loadKnown || !measured.sample.swapKnown || !measured.sample.diskKnown || measured.completed.Before(measured.taken) || measured.pressure != 1 {
		t.Fatalf("native sample: %+v", measured)
	}
}
