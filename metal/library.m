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
#include "bridge.h"
#include <stdatomic.h>
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

// stopped is whether waits are stopped: set from Go, when the context a
// drawing stops on is done, and read by every wait at each look. a wait
// ends within look microseconds of it, however long the command runs.
static atomic_int stopped;

void slayer_stop_waits(int stop) {
	atomic_store(&stopped, stop != 0);
}

BOOL slayer_waits_stopped(void) {
	return atomic_load(&stopped) != 0;
}

BOOL slayer_wait_command(id<MTLCommandBuffer> commands, double limit,
                         NSString *what, NSString **why) {
	NSDate *begun = [NSDate date];
	while (commands.status != MTLCommandBufferStatusCompleted &&
	       commands.status != MTLCommandBufferStatusError) {
		if (slayer_waits_stopped()) {
			*why = [NSString stringWithFormat:
				@"%@ was stopped before it completed; the command may "
				@"still be running on the device, which may stay busy, "
				@"so do not reuse this one", what];
			return NO;
		}
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

// the stop as it was measured on a stopped band of the city, 2026-11-16:
// MTLCommandBufferErrorDomain code 1, "Impacting Interactivity
// (0000000e:kIOGPUCommandBufferCallbackErrorImpactingInteractivity)",
// over an underlying IOGPUCommandQueueErrorDomain code 14, its encoder
// faulted. metal's own code is only "internal"; the underlying one names it.
static NSString *const displayStopDomain = @"IOGPUCommandQueueErrorDomain";

// displayStopCode is kIOGPUCommandBufferCallbackErrorImpactingInteractivity.
static const NSInteger displayStopCode = 0xe;

BOOL slayer_stopped_for_display(NSError *why) {
	NSError *under = why.userInfo[NSUnderlyingErrorKey];
	return why != nil &&
	       [why.domain isEqualToString:MTLCommandBufferErrorDomain] &&
	       [under.domain isEqualToString:displayStopDomain] &&
	       under.code == displayStopCode;
}

// encoder_states is each encoder's execution state, from a command made
// to keep them, which is what says whether the kernel ran, faulted or
// was never reached when the command stopped
static NSString *encoder_states(NSError *why) {
	NSArray *infos = why.userInfo[MTLCommandBufferEncoderInfoErrorKey];
	NSMutableArray<NSString *> *states = [NSMutableArray array];
	for (id<MTLCommandBufferEncoderInfo> info in infos) {
		[states addObject:[NSString stringWithFormat:@"%@ state %ld",
		                   info.label, (long)info.errorState]];
	}
	return [states componentsJoinedByString:@", "];
}

NSString *slayer_command_failure(NSString *what, NSError *why) {
	NSString *said = [NSString stringWithFormat:@"%@: %@ (%@ code %ld)",
	                  what, why.localizedDescription, why.domain,
	                  (long)why.code];
	if (!slayer_keeps_encoder_states()) {
		return said;
	}
	return [NSString stringWithFormat:@"%@; encoders: [%@]; user info: %@",
	        said, encoder_states(why), why.userInfo];
}

BOOL slayer_keeps_encoder_states(void) {
	static BOOL kept;
	static dispatch_once_t once;
	dispatch_once(&once, ^{
		kept = getenv("SLAYER_BAND_TIMES") != NULL;
	});
	return kept;
}
