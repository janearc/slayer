package telemetry

import (
	"encoding/binary"
	"math"

	"github.com/janearc/slayer/internal/observability"
	"golang.org/x/sys/unix"
)

type hostSource interface {
	sysctl(string) ([]byte, error)
	statfs(string) (unix.Statfs_t, error)
	vm() (vmPageStates, error)
}

type nativeHost struct{}

// sysctl reads a raw kernel value by name.
func (nativeHost) sysctl(name string) ([]byte, error) { return unix.SysctlRaw(name) }

// statfs reads the filesystem statistics for path.
func (nativeHost) statfs(path string) (unix.Statfs_t, error) {
	var stat unix.Statfs_t
	err := unix.Statfs(path, &stat)
	return stat, err
}

// hostSamples reads each instrument independently, including when pressure
// is elevated. The disk reading belongs to the queue filesystem.
func hostSamples(host hostSource, directory string) []observability.Sample {
	out := make([]observability.Sample, 0, 22)
	put := func(name string, value float64) { out = append(out, observability.Sample{Name: name, Value: value}) }
	memory, err := host.sysctl("hw.memsize")
	validMemory := err == nil && len(memory) == 8 && binary.NativeEndian.Uint64(memory) > 0
	put("azazel_host_physical_memory_total_available", flag(validMemory))
	if validMemory {
		memoryBytes := binary.NativeEndian.Uint64(memory)
		put("azazel_host_physical_memory_total_bytes", float64(memoryBytes))
	}
	pages, err := host.vm()
	states := []struct {
		name  string
		count uint64
	}{
		{"azazel_host_vm_free_bytes", pages.Free},
		{"azazel_host_vm_active_bytes", pages.Active},
		{"azazel_host_vm_inactive_bytes", pages.Inactive},
		{"azazel_host_vm_wired_bytes", pages.Wired},
		{"azazel_host_vm_speculative_bytes", pages.Speculative},
	}
	validPages := err == nil && pages.PageBytes > 0 && pages.Speculative <= pages.Free
	if validPages {
		for _, state := range states {
			if state.count > math.MaxUint64/pages.PageBytes {
				validPages = false
				break
			}
		}
	}
	put("azazel_host_vm_page_states_available", flag(validPages))
	if validPages {
		put("azazel_host_vm_page_size_bytes", float64(pages.PageBytes))
		for _, state := range states {
			put(state.name, float64(state.count*pages.PageBytes))
		}
	}
	validCompressor := validPages && pages.CompressorKnown && pages.Compressor <= math.MaxUint64/pages.PageBytes
	put("azazel_host_vm_compressor_available", flag(validCompressor))
	if validCompressor {
		put("azazel_host_vm_compressor_resident_bytes", float64(pages.Compressor*pages.PageBytes))
	}
	swap, err := host.sysctl("vm.swapusage")
	validSwap := err == nil && len(swap) == 32
	var total, free, used uint64
	if validSwap {
		total, free, used = binary.NativeEndian.Uint64(swap), binary.NativeEndian.Uint64(swap[8:]), binary.NativeEndian.Uint64(swap[16:])
		validSwap = free <= total && used == total-free
	}
	put("azazel_host_swap_available", flag(validSwap))
	if validSwap {
		put("azazel_host_swap_total_bytes", float64(total))
		put("azazel_host_swap_free_bytes", float64(free))
		put("azazel_host_swap_used_bytes", float64(used))
	}
	pressure, err := host.sysctl("kern.memorystatus_vm_pressure_level")
	validPressure := err == nil && len(pressure) == 4
	var level uint32
	if validPressure {
		level = binary.NativeEndian.Uint32(pressure)
		validPressure = level == 1 || level == 2 || level == 4
	}
	put("azazel_host_memory_pressure_available", flag(validPressure))
	if validPressure {
		put("azazel_host_memory_pressure_level", float64(level))
	}
	stat, err := host.statfs(directory)
	validDisk := err == nil && stat.Bsize > 0 && stat.Bavail <= stat.Bfree && stat.Bfree <= stat.Blocks && uint64(stat.Bavail) <= math.MaxUint64/uint64(stat.Bsize)
	put("azazel_host_disk_available", flag(validDisk))
	if validDisk {
		diskFreeBytes := uint64(stat.Bavail) * uint64(stat.Bsize)
		put("azazel_host_disk_free_bytes", float64(diskFreeBytes))
	}
	return out
}
