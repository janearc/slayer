// A device snapshot taken only after an instrumented GPU glimpse. It does
// not open a command queue or alter the renderer's retained context.
#import <Foundation/Foundation.h>
#import <Metal/Metal.h>
#include <stdint.h>

int slayer_metal_memory_snapshot(uint64_t *allocated, uint64_t *recommended) {
	@autoreleasepool {
		id<MTLDevice> device = MTLCreateSystemDefaultDevice();
		if (device == nil) return 0;
		*allocated = (uint64_t)device.currentAllocatedSize;
		*recommended = device.recommendedMaxWorkingSetSize;
		return 1;
	}
}
