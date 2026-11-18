// whether someone is using the machine while a band draws, read and never
// touched: the system's time since the last input, and whether the main
// display sleeps. the stops this records are the system's "impacting
// interactivity", so a band's receipt says what the interactivity was.

#import <Foundation/Foundation.h>
#import <IOKit/IOKitLib.h>
#import <CoreGraphics/CoreGraphics.h>
#include "bridge.h"

long long slayer_idle_nanoseconds(void) {
	io_service_t system = IOServiceGetMatchingService(
		kIOMainPortDefault, IOServiceMatching("IOHIDSystem"));
	if (system == IO_OBJECT_NULL) {
		return -1;
	}
	CFTypeRef idle = IORegistryEntryCreateCFProperty(
		system, CFSTR("HIDIdleTime"), kCFAllocatorDefault, 0);
	IOObjectRelease(system);
	if (idle == NULL) {
		return -1;
	}
	long long nanoseconds = -1;
	if (CFGetTypeID(idle) == CFNumberGetTypeID()) {
		CFNumberGetValue((CFNumberRef)idle, kCFNumberSInt64Type,
		                 &nanoseconds);
	}
	CFRelease(idle);
	return nanoseconds;
}

int slayer_display_asleep(void) {
	return CGDisplayIsAsleep(CGMainDisplayID()) ? 1 : 0;
}
