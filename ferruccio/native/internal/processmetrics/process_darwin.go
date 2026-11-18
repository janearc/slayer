//go:build darwin && !cgo && (arm64 || amd64)

// Package processmetrics provides Ferruccio's optional no-cgo memory sampler.
package processmetrics

import (
	"os"
	"runtime"
	"sync"
	"unsafe"

	"github.com/go-webgpu/goffi/ffi"
	"github.com/go-webgpu/goffi/types"
	"github.com/janearc/slayer/internal/observability"
)

// rusageV4 is Apple's public sys/resource.h rusage_info_v4. The full size
// matters because proc_pid_rusage writes the complete versioned structure.
// The test checks size and field offsets against the installed SDK headers.
type rusageV4 struct {
	UUID           [16]byte
	BeforeResident [6]uint64
	Resident       uint64
	Footprint      uint64
	BeforePeak     [20]uint64
	Peak           uint64
	AfterPeak      [6]uint64
}

type nativeReader struct {
	once     sync.Once
	library  unsafe.Pointer
	function unsafe.Pointer
	call     types.CallInterface
	ready    bool
}

// These values describe the public 64-bit Darwin rusage_info_v4 ABI and are
// independently checked against the SDK in TestRusageLayoutMatchesAppleSDK.
func supportedLayout() bool {
	var info rusageV4
	return unsafe.Sizeof(info) == 296 && unsafe.Offsetof(info.Resident) == 64 &&
		unsafe.Offsetof(info.Footprint) == 72 && unsafe.Offsetof(info.Peak) == 240
}

// initialize loads libSystem and the proc_pid_rusage call, once for the
// reader. It leaves the reader not ready when the layout or a load fails.
func (reader *nativeReader) initialize() {
	if !supportedLayout() {
		return
	}
	library, err := ffi.LoadLibrary("libSystem.B.dylib")
	if err != nil {
		return
	}
	function, err := ffi.GetSymbol(library, "proc_pid_rusage")
	if err != nil {
		_ = ffi.FreeLibrary(library)
		return
	}
	arguments := []*types.TypeDescriptor{types.SInt32TypeDescriptor, types.SInt32TypeDescriptor, types.PointerTypeDescriptor}
	if err := ffi.PrepareCallInterface(&reader.call, types.DefaultCall, types.SInt32TypeDescriptor, arguments); err != nil {
		_ = ffi.FreeLibrary(library)
		return
	}
	// Retain the library for the process lifetime: simultaneous scrapes can
	// safely share the immutable call interface and its function pointer.
	reader.library, reader.function, reader.ready = library, function, true
}

// read calls proc_pid_rusage for this process and flavor. It returns false
// when the call is not ready or fails.
func (reader *nativeReader) read(flavor int32) (rusageV4, bool) {
	reader.once.Do(reader.initialize)
	var info rusageV4
	if !reader.ready {
		return info, false
	}
	pid := int32(os.Getpid())
	buffer := unsafe.Pointer(&info)
	var result int32
	var pinned runtime.Pinner
	pinned.Pin(&info)
	defer pinned.Unpin()
	_, err := ffi.CallFunction(&reader.call, reader.function, unsafe.Pointer(&result),
		[]unsafe.Pointer{unsafe.Pointer(&pid), unsafe.Pointer(&flavor), unsafe.Pointer(&buffer)})
	runtime.KeepAlive(&info)
	if err != nil || result != 0 {
		return rusageV4{}, false
	}
	return info, true
}

// sample turns a rusage read into process memory figures. The lifetime
// footprint peak is reported only when the read succeeds.
func sample(read func(int32) (rusageV4, bool)) observability.ProcessMemory {
	info, ok := read(4) // RUSAGE_INFO_V4 includes the lifetime footprint peak.
	peakAvailable := ok
	if !ok {
		info, ok = read(0) // RUSAGE_INFO_V0 supplies current values only.
	}
	if !ok {
		return observability.ProcessMemory{}
	}
	currentOK := info.Footprint > 0
	peakOK := peakAvailable && currentOK && info.Peak >= info.Footprint
	return observability.ProcessMemory{
		ResidentBytes: info.Resident, ResidentAvailable: info.Resident > 0,
		FootprintBytes: info.Footprint, FootprintAvailable: currentOK,
		PeakFootprintBytes: info.Peak, PeakFootprintAvailable: peakOK,
	}
}

// Register installs a lazy sampler. The dylib is loaded on the first scrape
// or configured summary, never merely by starting the application.
func Register() func() {
	reader := &nativeReader{}
	return observability.RegisterProcessMemorySampler(func() observability.ProcessMemory {
		return sample(reader.read)
	})
}
