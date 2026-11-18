//go:build darwin && cgo

package metal

/*
#include <stdint.h>
int slayer_metal_memory_snapshot(uint64_t *allocated, uint64_t *recommended);
*/
import "C"

// MemorySnapshot reads the default Metal device. It opens no command queue;
// callers should invoke it only after a real GPU draw has selected that
// device. The allocation reading is a snapshot, not a peak or process RSS.
func MemorySnapshot() (allocated, recommended uint64, available bool) {
	var current, working C.uint64_t
	available = C.slayer_metal_memory_snapshot(&current, &working) != 0
	return uint64(current), uint64(working), available
}
