package budget

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"os"
	"time"

	"golang.org/x/sys/unix"
)

var errAdmissionPressure = errors.New("admission memory pressure is not normal")

type admissionHostAdapter interface {
	SysctlRaw(string) ([]byte, error)
	Statfs(int) (unix.Statfs_t, error)
	Now() time.Time
}

type nativeAdmissionHost struct{}

// SysctlRaw returns the raw bytes of the system value named key.
func (nativeAdmissionHost) SysctlRaw(key string) ([]byte, error) { return unix.SysctlRaw(key) }

// Statfs returns the file system statistics for the open file descriptor fd.
func (nativeAdmissionHost) Statfs(fd int) (unix.Statfs_t, error) {
	var stat unix.Statfs_t
	err := unix.Fstatfs(fd, &stat)
	return stat, err
}

// Now returns the time from the system clock.
func (nativeAdmissionHost) Now() time.Time { return time.Now() }

// admissionHostRead reads the system value key through adapter. A done
// context wins over a read error, whether it ended before or during the read;
// a read error is wrapped with the key.
func admissionHostRead(ctx context.Context, adapter admissionHostAdapter, key string) ([]byte, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	data, err := adapter.SysctlRaw(key)
	if canceled := ctx.Err(); canceled != nil {
		return nil, canceled
	}
	if err != nil {
		return nil, fmt.Errorf("admission host %s: %w", key, err)
	}
	return data, nil
}

// admissionBoot returns the boot session UUID that adapter reports for
// kern.bootsessionuuid. It must be 36 characters of hex digits and dashes in
// the usual places, then a zero byte, or the result is errAdmissionUnsafe.
func admissionBoot(ctx context.Context, adapter admissionHostAdapter) (string, error) {
	data, err := admissionHostRead(ctx, adapter, "kern.bootsessionuuid")
	if err != nil {
		return "", err
	}
	if len(data) != 37 || data[36] != 0 {
		return "", errAdmissionUnsafe
	}
	for i, c := range data[:36] {
		if i == 8 || i == 13 || i == 18 || i == 23 {
			if c != '-' {
				return "", errAdmissionUnsafe
			}
		} else if !(c >= '0' && c <= '9' || c >= 'A' && c <= 'F' || c >= 'a' && c <= 'f') {
			return "", errAdmissionUnsafe
		}
	}
	return string(data[:36]), nil
}

// nativeAdmissionBoot is admissionBoot on the real system. A nil ctx is
// refused.
func nativeAdmissionBoot(ctx context.Context) (string, error) {
	if ctx == nil {
		return "", errAdmissionUnsafe
	}
	return admissionBoot(ctx, nativeAdmissionHost{})
}

// measureAdmissionHost takes a host measurement from the real system. output
// must be an open directory; free disk space is read for the file system it
// is on.
func measureAdmissionHost(ctx context.Context, output *os.File, binding admissionBinding) (admissionMeasurement, error) {
	return sampleAdmissionHost(ctx, output, binding, nativeAdmissionHost{})
}

// sampleAdmissionHost reads load average, swap use, memory pressure and free
// disk space of output's file system through adapter, with the times the
// sample began and ended. It returns errAdmissionUnsafe for a bad argument, a
// boot that is not binding's or changes mid-sample, a value that does not
// parse, or a sample older than admissionSampleAge. Pressure 2 or 4 gives
// errAdmissionPressure.
func sampleAdmissionHost(ctx context.Context, output *os.File, binding admissionBinding, adapter admissionHostAdapter) (admissionMeasurement, error) {
	var empty admissionMeasurement
	if ctx == nil || output == nil || adapter == nil || binding.limit == 0 || binding.policy.assess(hostSample{}) == hostInvalidPolicy {
		return empty, errAdmissionUnsafe
	}
	if err := ctx.Err(); err != nil {
		return empty, err
	}
	taken := adapter.Now()
	if taken.IsZero() {
		return empty, errAdmissionUnsafe
	}
	info, err := output.Stat()
	if err != nil {
		return empty, err
	}
	if !info.IsDir() {
		return empty, errAdmissionUnsafe
	}
	boot, err := admissionBoot(ctx, adapter)
	if err != nil {
		return empty, err
	}
	if boot != binding.boot {
		return empty, errAdmissionUnsafe
	}
	load, err := admissionHostRead(ctx, adapter, "vm.loadavg")
	if err != nil {
		return empty, err
	}
	if len(load) != 24 {
		return empty, errAdmissionUnsafe
	}
	scale := int64(binary.NativeEndian.Uint64(load[16:]))
	if scale <= 0 {
		return empty, errAdmissionUnsafe
	}
	loadAverage := float64(binary.NativeEndian.Uint32(load)) / float64(scale)
	swap, err := admissionHostRead(ctx, adapter, "vm.swapusage")
	if err != nil {
		return empty, err
	}
	if len(swap) != 32 {
		return empty, errAdmissionUnsafe
	}
	total := binary.NativeEndian.Uint64(swap)
	available := binary.NativeEndian.Uint64(swap[8:])
	used := binary.NativeEndian.Uint64(swap[16:])
	page := binary.NativeEndian.Uint32(swap[24:])
	encrypted := binary.NativeEndian.Uint32(swap[28:])
	if used > total || available > total || used != total-available || page == 0 || page&(page-1) != 0 || encrypted > 1 {
		return empty, errAdmissionUnsafe
	}
	pressure, err := admissionHostRead(ctx, adapter, "kern.memorystatus_vm_pressure_level")
	if err != nil {
		return empty, err
	}
	if len(pressure) != 4 {
		return empty, errAdmissionUnsafe
	}
	level := binary.NativeEndian.Uint32(pressure)
	if level != 1 {
		if level == 2 || level == 4 {
			return empty, errAdmissionPressure
		}
		return empty, errAdmissionUnsafe
	}
	if err := ctx.Err(); err != nil {
		return empty, err
	}
	descriptor, err := output.SyscallConn()
	if err != nil {
		return empty, err
	}
	var stat unix.Statfs_t
	var statErr error
	err = descriptor.Control(func(fd uintptr) { stat, statErr = adapter.Statfs(int(fd)) })
	if canceled := ctx.Err(); canceled != nil {
		return empty, canceled
	}
	if err != nil || statErr != nil {
		return empty, errors.Join(err, statErr)
	}
	if stat.Bsize == 0 || stat.Bavail > stat.Bfree || stat.Bfree > stat.Blocks || stat.Bavail > math.MaxUint64/uint64(stat.Bsize) {
		return empty, errAdmissionUnsafe
	}
	finalBoot, err := admissionBoot(ctx, adapter)
	if err != nil {
		return empty, err
	}
	if finalBoot != boot {
		return empty, errAdmissionUnsafe
	}
	completed := adapter.Now()
	if completed.Before(taken) || completed.Sub(taken) > admissionSampleAge {
		return empty, errAdmissionUnsafe
	}
	if err := ctx.Err(); err != nil {
		return empty, err
	}
	return admissionMeasurement{
		binding: binding, taken: taken, completed: completed, pressure: level,
		sample: hostSample{load: loadAverage, swap: byteCount(used), freeDisk: byteCount(stat.Bavail * uint64(stat.Bsize)), loadKnown: true, swapKnown: true, diskKnown: true},
	}, nil
}
