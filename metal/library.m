// the shared shader cache. compiling a library from source costs tens
// of milliseconds; opening the same kernels for every frame, test or
// per-scene variant paid it every time. the cache holds the compiled
// library against its whole source text, so distinct sources can never
// collide, and against the device it was compiled for, so a machine
// with more than one gpu never crosses them.
//
// the cache only grows: a process compiles as many distinct sources as
// it has distinct kernels and per-scene variants, a handful, and each
// entry is the source string and the compiled library it already paid
// for.

#import "library.h"
#include <unistd.h>

static NSMutableDictionary<NSString *, id<MTLLibrary>> *kept;
static NSLock *guard;

// slayer_shared_library_count is how many distinct sources the cache
// holds, for the test that shows a second open compiles nothing.
long slayer_shared_library_count(void) {
	if (guard == NULL) {
		return 0;
	}
	[guard lock];
	long count = (long)kept.count;
	[guard unlock];
	return count;
}

id<MTLLibrary> slayer_shared_library(id<MTLDevice> device,
                                     const char *source, NSError **why) {
	static dispatch_once_t once;
	dispatch_once(&once, ^{
		kept = [NSMutableDictionary dictionary];
		guard = [NSLock new];
	});
	NSString *text = [NSString stringWithUTF8String:source];
	NSString *name = [NSString stringWithFormat:@"%llu\n%@",
	                  device.registryID, text];
	[guard lock];
	id<MTLLibrary> found = kept[name];
	[guard unlock];
	if (found != nil) {
		return found;
	}
	id<MTLLibrary> made = [device newLibraryWithSource:text
	                                           options:[MTLCompileOptions new]
	                                             error:why];
	if (made != nil) {
		[guard lock];
		// a race compiles twice and keeps one; both are correct
		kept[name] = made;
		[guard unlock];
	}
	return made;
}

// the pause between looks at a command, in microseconds: short beside the
// milliseconds a band takes, long enough that waiting costs no cpu to speak of
static const useconds_t look = 200;

BOOL slayer_wait_command(id<MTLCommandBuffer> commands, double limit,
                         NSString *what, NSString **why) {
	NSDate *begun = [NSDate date];
	while (commands.status != MTLCommandBufferStatusCompleted &&
	       commands.status != MTLCommandBufferStatusError) {
		if (-[begun timeIntervalSinceNow] >= limit) {
			*why = [NSString stringWithFormat:
				@"%@ did not complete in %g s; the command may still be "
				@"running on the device, which may stay busy, so do not "
				@"reuse this one", what, limit];
			return NO;
		}
		usleep(look);
	}
	// the command is over, so this returns at once; it leaves its status,
	// error and timestamps final, as every wait on a command has
	[commands waitUntilCompleted];
	return YES;
}

NSString *slayer_stuck(void) {
	return @"a command timed out earlier and may still be running on the "
	       @"device; do not reuse this one";
}
