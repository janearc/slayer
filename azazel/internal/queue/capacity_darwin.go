package queue

/*
#cgo LDFLAGS: -framework IOKit -framework CoreFoundation
#include <IOKit/IOKitLib.h>
#include <CoreFoundation/CoreFoundation.h>

static int azazel_gpu_busy(int *percent) {
    io_iterator_t devices = IO_OBJECT_NULL;
    if (IOServiceGetMatchingServices(kIOMainPortDefault,
        IOServiceMatching("AGXAccelerator"), &devices) != KERN_SUCCESS) return -1;
    io_object_t device;
    int found = 0;
    int valid = 0;
    while ((device = IOIteratorNext(devices)) != IO_OBJECT_NULL) {
        found++;
        CFTypeRef stats = IORegistryEntryCreateCFProperty(device,
            CFSTR("PerformanceStatistics"), kCFAllocatorDefault, 0);
        if (stats && CFGetTypeID(stats) == CFDictionaryGetTypeID()) {
            CFTypeRef value = CFDictionaryGetValue((CFDictionaryRef)stats,
                CFSTR("Device Utilization %"));
            if (value && CFGetTypeID(value) == CFNumberGetTypeID()) {
                valid = CFNumberGetValue((CFNumberRef)value, kCFNumberIntType, percent);
            }
        }
        if (stats) CFRelease(stats);
        IOObjectRelease(device);
    }
    IOObjectRelease(devices);
    return found == 1 && valid ? 0 : -1;
}
*/
import "C"

import (
	"context"

	"github.com/janearc/slayer/azazel/internal/budget"
)

// NativeCapacityProbe observes host pressure and the device-wide GPU counter.
type NativeCapacityProbe struct{}

// Read holds GPU starts on unknown utilization while allowing admitted CPU work.
func (NativeCapacityProbe) Read(ctx context.Context, directory string) (Capacity, error) {
	return readCapacity(ctx, directory, nativeCapacitySystem{})
}

type nativeCapacitySystem struct{}

// MetricsGPUDeviceUtilization reads the device-wide GPU utilization counter.
// It does not represent a particular process or make an admission decision.
func MetricsGPUDeviceUtilization() (int, bool) {
	percent, known := (nativeCapacitySystem{}).GPU()
	return percent, known && percent >= 0 && percent <= 100
}

// Host reads host pressure and disk observations for directory.
func (nativeCapacitySystem) Host(ctx context.Context, directory string) (budget.HostObservation, error) {
	return budget.ObserveHost(ctx, directory)
}

// GPU reads the device-wide GPU utilization counter. known is false when the
// counter cannot be read.
func (nativeCapacitySystem) GPU() (int, bool) {
	var percent C.int
	known := C.azazel_gpu_busy(&percent) == 0
	return int(percent), known
}
