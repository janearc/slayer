package telemetry

import (
	"encoding/binary"
	"errors"
	"math"
	"testing"

	"golang.org/x/sys/unix"
)

type fakeHost struct {
	values  map[string][]byte
	disk    unix.Statfs_t
	diskErr error
	pages   vmPageStates
	vmErr   error
}

func (h fakeHost) sysctl(name string) ([]byte, error) {
	if value, ok := h.values[name]; ok {
		return value, nil
	}
	return nil, errors.New("unavailable")
}
func (h fakeHost) statfs(string) (unix.Statfs_t, error) { return h.disk, h.diskErr }
func (h fakeHost) vm() (vmPageStates, error)            { return h.pages, h.vmErr }

func TestHostSamplesSurviveElevatedPressureAndPartialFailure(t *testing.T) {
	swap := make([]byte, 32)
	binary.NativeEndian.PutUint64(swap[0:], 1000)
	binary.NativeEndian.PutUint64(swap[8:], 300)
	binary.NativeEndian.PutUint64(swap[16:], 700)
	pressure := make([]byte, 4)
	binary.NativeEndian.PutUint32(pressure, 2)
	host := fakeHost{values: map[string][]byte{"vm.swapusage": swap, "kern.memorystatus_vm_pressure_level": pressure}, diskErr: errors.New("disk missing")}
	got := values(hostSamples(host, "/output"))
	for name, want := range map[string]float64{
		"azazel_host_swap_available": 1, "azazel_host_swap_used_bytes": 700,
		"azazel_host_swap_free_bytes": 300, "azazel_host_swap_total_bytes": 1000,
		"azazel_host_memory_pressure_available": 1, "azazel_host_memory_pressure_level": 2,
		"azazel_host_disk_available": 0,
	} {
		if got[name] != want {
			t.Errorf("%s = %v, want %v", name, got[name], want)
		}
	}
	if _, ok := got["azazel_host_disk_free_bytes"]; ok {
		t.Fatal("invented free disk", got)
	}
}

func TestHostSamplesRejectMalformedAndReadDiskIndependently(t *testing.T) {
	host := fakeHost{values: map[string][]byte{"vm.swapusage": {1}, "kern.memorystatus_vm_pressure_level": {7, 0, 0, 0}}, disk: unix.Statfs_t{Bsize: 4096, Blocks: 20, Bfree: 10, Bavail: 8}}
	got := values(hostSamples(host, "/output"))
	if got["azazel_host_swap_available"] != 0 || got["azazel_host_memory_pressure_available"] != 0 || got["azazel_host_disk_available"] != 1 || got["azazel_host_disk_free_bytes"] != 8*4096 {
		t.Fatal(got)
	}
	if _, ok := got["azazel_host_swap_used_bytes"]; ok {
		t.Fatal("invented swap", got)
	}
}

func TestHostMemoryTotalsAndPageStatesAreIndependent(t *testing.T) {
	total := make([]byte, 8)
	binary.NativeEndian.PutUint64(total, 64*1024)
	pressure := make([]byte, 4)
	binary.NativeEndian.PutUint32(pressure, 2)
	host := fakeHost{
		values: map[string][]byte{"hw.memsize": total, "kern.memorystatus_vm_pressure_level": pressure},
		pages: vmPageStates{PageBytes: 4096, Free: 4, Active: 3,
			Inactive: 2, Wired: 1, Speculative: 1, Compressor: 5, CompressorKnown: true},
		diskErr: errors.New("disk unavailable"),
	}
	got := values(hostSamples(host, "/unused"))
	for name, want := range map[string]float64{
		"azazel_host_physical_memory_total_available": 1,
		"azazel_host_physical_memory_total_bytes":     64 * 1024,
		"azazel_host_vm_page_states_available":        1,
		"azazel_host_vm_page_size_bytes":              4096,
		"azazel_host_vm_free_bytes":                   4 * 4096,
		"azazel_host_vm_active_bytes":                 3 * 4096,
		"azazel_host_vm_inactive_bytes":               2 * 4096,
		"azazel_host_vm_wired_bytes":                  4096,
		"azazel_host_vm_speculative_bytes":            4096,
		"azazel_host_vm_compressor_available":         1,
		"azazel_host_vm_compressor_resident_bytes":    5 * 4096,
	} {
		if got[name] != want {
			t.Errorf("%s = %v, want %v", name, got[name], want)
		}
	}
	if got["azazel_host_memory_pressure_available"] != 1 || got["azazel_host_memory_pressure_level"] != 2 || got["azazel_host_swap_available"] != 0 {
		t.Fatal("invented unrelated observations", got)
	}
}

func TestOlderHostCanReportPageStatesWithoutCompressor(t *testing.T) {
	host := fakeHost{pages: vmPageStates{PageBytes: 4096, Free: 4, Active: 3, Speculative: 1}}
	got := values(hostSamples(host, "/unused"))
	if got["azazel_host_vm_page_states_available"] != 1 || got["azazel_host_vm_compressor_available"] != 0 || got["azazel_host_vm_free_bytes"] != 4*4096 {
		t.Fatal(got)
	}
	if _, ok := got["azazel_host_vm_compressor_resident_bytes"]; ok {
		t.Fatal("invented compressor reading", got)
	}
}

func TestCompressorOverflowDoesNotHideOtherPageStates(t *testing.T) {
	host := fakeHost{pages: vmPageStates{PageBytes: 4096, Free: 4, Active: 3,
		Compressor: math.MaxUint64, CompressorKnown: true}}
	got := values(hostSamples(host, "/unused"))
	if got["azazel_host_vm_page_states_available"] != 1 || got["azazel_host_vm_compressor_available"] != 0 || got["azazel_host_vm_free_bytes"] != 4*4096 {
		t.Fatal(got)
	}
	if _, ok := got["azazel_host_vm_compressor_resident_bytes"]; ok {
		t.Fatal("overflowed compressor reading", got)
	}
}

func TestHostMemoryRejectsMalformedOrOverflowWithoutPartialValues(t *testing.T) {
	for _, host := range []fakeHost{
		{values: map[string][]byte{"hw.memsize": {1}}, pages: vmPageStates{PageBytes: 4096, Free: math.MaxUint64}},
		{values: map[string][]byte{"hw.memsize": make([]byte, 8)}, pages: vmPageStates{}},
		{pages: vmPageStates{PageBytes: 4096, Free: 1, Speculative: 2}},
		{vmErr: errors.New("unavailable")},
	} {
		got := values(hostSamples(host, "/unused"))
		if got["azazel_host_physical_memory_total_available"] != 0 || got["azazel_host_vm_page_states_available"] != 0 {
			t.Fatal("accepted invalid readings", got)
		}
		for _, name := range []string{"azazel_host_physical_memory_total_bytes", "azazel_host_vm_free_bytes", "azazel_host_vm_wired_bytes"} {
			if _, ok := got[name]; ok {
				t.Errorf("invented %s: %v", name, got)
			}
		}
	}
}

func TestNativeVMPageObservationHasValidHostPageSize(t *testing.T) {
	pages, err := (nativeHost{}).vm()
	if err != nil {
		t.Fatal(err)
	}
	// The host page size can differ from the process page size under Rosetta.
	if pages.PageBytes == 0 || pages.PageBytes&(pages.PageBytes-1) != 0 || pages.Speculative > pages.Free {
		t.Fatalf("unexpected native VM observation: %+v", pages)
	}
}
