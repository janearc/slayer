package observability

import "sync"

// ProcessMemory is one optional, current-process native memory observation.
// Availability is independent for RSS, current footprint and lifetime peak.
type ProcessMemory struct {
	ResidentBytes, FootprintBytes, PeakFootprintBytes             uint64
	ResidentAvailable, FootprintAvailable, PeakFootprintAvailable bool
}

var processMemoryProvider struct {
	sync.RWMutex
	read   func() ProcessMemory
	serial uint64
}

// RegisterProcessMemorySampler installs one optional sampler. A duplicate or
// nil registration does nothing. The returned function unregisters its own
// provider and is safe to call more than once. The provider is called only
// during a metrics scrape or an explicitly configured summary.
func RegisterProcessMemorySampler(read func() ProcessMemory) func() {
	processMemoryProvider.Lock()
	if read == nil || processMemoryProvider.read != nil {
		processMemoryProvider.Unlock()
		return func() {}
	}
	processMemoryProvider.serial++
	serial := processMemoryProvider.serial
	processMemoryProvider.read = read
	processMemoryProvider.Unlock()
	var once sync.Once
	return func() {
		once.Do(func() {
			processMemoryProvider.Lock()
			if processMemoryProvider.serial == serial {
				processMemoryProvider.read = nil
			}
			processMemoryProvider.Unlock()
		})
	}
}

// The sampler is optional diagnostics: even a faulty provider cannot bring
// down the renderer or summary goroutine.
func registeredProcessMemory() (sample ProcessMemory) {
	processMemoryProvider.RLock()
	read := processMemoryProvider.read
	processMemoryProvider.RUnlock()
	if read == nil {
		return sample
	}
	defer func() {
		if recover() != nil {
			sample = ProcessMemory{}
		}
	}()
	return read()
}
