package observability

import (
	"sync"
	"testing"
)

func TestOptionalProcessMemorySamplerRegistration(t *testing.T) {
	if got := registeredProcessMemory(); got.ResidentAvailable {
		t.Fatal("unregistered sampler reported a value")
	}
	stop := RegisterProcessMemorySampler(func() ProcessMemory {
		return ProcessMemory{ResidentBytes: 123, ResidentAvailable: true}
	})
	other := RegisterProcessMemorySampler(func() ProcessMemory {
		return ProcessMemory{ResidentBytes: 999, ResidentAvailable: true}
	})
	other()
	if got := registeredProcessMemory(); !got.ResidentAvailable || got.ResidentBytes != 123 {
		t.Fatalf("duplicate registration displaced sampler: %+v", got)
	}
	stop()
	stop()
	if got := registeredProcessMemory(); got.ResidentAvailable {
		t.Fatalf("unregistered sampler remained active: %+v", got)
	}
	stop = RegisterProcessMemorySampler(func() ProcessMemory { panic("diagnostic failure") })
	defer stop()
	if got := registeredProcessMemory(); got.ResidentAvailable || got.FootprintAvailable {
		t.Fatalf("failed sampler invented readings: %+v", got)
	}
}

func TestProcessMemorySamplerConcurrentReadsAndUnregister(t *testing.T) {
	var workers sync.WaitGroup
	for range 4 {
		workers.Add(1)
		go func() {
			defer workers.Done()
			for range 500 {
				_ = registeredProcessMemory()
			}
		}()
	}
	for range 100 {
		stop := RegisterProcessMemorySampler(func() ProcessMemory {
			return ProcessMemory{ResidentBytes: 123, ResidentAvailable: true}
		})
		stop()
	}
	workers.Wait()
}
