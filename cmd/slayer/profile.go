package main

import (
	"fmt"
	"io"
	"os"
	"runtime"
	"runtime/pprof"
	"time"
)

// profileVariable names the file a run writes Go's CPU profile to, for
// `go tool pprof`; unset, nothing is profiled. A stack sample from
// outside loses the callers of a hot leaf, and a draw's time is a
// question of which callers pay.
const profileVariable = "SLAYER_CPUPROFILE"

// startProfile starts the CPU profile the environment asks for and
// returns what stops it and writes it out; a profile that cannot start is
// said, and the run goes on unprofiled.
func startProfile(wrong io.Writer) func() {
	path := os.Getenv(profileVariable)
	if path == "" {
		return func() {}
	}
	out, err := os.Create(path)
	if err != nil {
		fmt.Fprintf(wrong, "slayer: %s: %v\n", profileVariable, err)
		return func() {}
	}
	if err := pprof.StartCPUProfile(out); err != nil {
		fmt.Fprintf(wrong, "slayer: %s: %v\n", profileVariable, err)
		out.Close()
		return func() {}
	}
	return func() {
		pprof.StopCPUProfile()
		if err := out.Close(); err != nil {
			fmt.Fprintf(wrong, "slayer: %s: %v\n", profileVariable, err)
		}
	}
}

// heapVariable names the file a run writes Go's heap profile to at the
// largest the heap has been; unset, nothing is written. A draw's memory
// is at its most once staging is done and painting begins, not at its
// end, so the profile is written again each time the heap is found
// larger than before. What the process holds beyond the heap's peak is
// held outside Go: the device's buffers, and the native bridge's.
const heapVariable = "SLAYER_HEAPPROFILE"

// heapLook is how often the heap is looked at. picked: often enough to
// catch a peak that lasts a stage, seldom enough to cost nothing.
const heapLook = 10 * time.Second

// startHeapProfile watches the heap where the environment asks, writing
// its profile at each new peak, and returns what stops the watch and
// writes it once more if the heap ended at its peak.
func startHeapProfile(wrong io.Writer) func() {
	path := os.Getenv(heapVariable)
	if path == "" {
		return func() {}
	}
	done := make(chan struct{})
	finished := make(chan struct{})
	var most uint64
	look := func() {
		var stats runtime.MemStats
		runtime.ReadMemStats(&stats)
		if stats.HeapInuse <= most {
			return
		}
		most = stats.HeapInuse
		if err := writeHeap(path); err != nil {
			fmt.Fprintf(wrong, "slayer: %s: %v\n", heapVariable, err)
		}
	}
	go func() {
		defer close(finished)
		tick := time.NewTicker(heapLook)
		defer tick.Stop()
		for {
			select {
			case <-tick.C:
				look()
			case <-done:
				look()
				return
			}
		}
	}()
	return func() {
		close(done)
		<-finished
		fmt.Fprintf(wrong, "slayer: %s: the heap's peak in use was %.1f GB\n",
			heapVariable, float64(most)/1e9)
	}
}

// writeHeap writes the heap profile as it is now to a file.
func writeHeap(path string) error {
	out, err := os.Create(path)
	if err != nil {
		return err
	}
	if err := pprof.Lookup("heap").WriteTo(out, 0); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}
