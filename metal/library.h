// the shared shader cache: one compiled library per distinct source a
// process asks for. objective-c only; bridge.h stays plain C for cgo.

#ifndef SLAYER_METAL_LIBRARY_H
#define SLAYER_METAL_LIBRARY_H

#ifdef __OBJC__
#import <Metal/Metal.h>

// slayer_shared_library is the library for a source on a device,
// compiled the first time the pair is asked for and remembered after,
// so a process compiles each distinct source once however many kernels
// and sessions open it. nil on a failed compile, with why saying why;
// failures are not remembered, so a caller may try again.
id<MTLLibrary> slayer_shared_library(id<MTLDevice> device,
                                     const char *source, NSError **why);

// slayer_wait_command waits for a committed command to finish, for at
// most limit seconds. It is YES when the command finished, whatever its
// own error says. It is NO when the limit passed first, or waits were
// stopped, with why saying what was waited for and for how long: the
// command is then still queued or running on the device, which may stay
// busy until it ends.
BOOL slayer_wait_command(id<MTLCommandBuffer> commands, double limit,
                         NSString *what, NSString **why);

// slayer_waits_stopped is whether waits are stopped now, as
// slayer_stop_waits (bridge.h) set them. A wait for a command ends at
// its next look when they are, without the command ending: Metal cannot
// take a running command back, so the device is left to finish or lose
// it, and the caller says it will not use the context or session again.
BOOL slayer_waits_stopped(void);

// slayer_stuck is the sentence a caller says when it will not use a
// context or session again because a command outran its limit.
NSString *slayer_stuck(void);

// slayer_stopped_for_display is whether a command's error is the system
// stopping it so the display could present: on Apple Silicon the window
// server draws on the same gpu. Such a command was stopped cleanly; the
// device is fine and the same work may be sent again in smaller parts.
BOOL slayer_stopped_for_display(NSError *why);

// slayer_command_failure is what failed and the system's words for why,
// with the error's domain and code, which say which case it was.
NSString *slayer_command_failure(NSString *what, NSError *why);

// slayer_keeps_encoder_states is whether commands are made to keep each
// encoder's execution state for a failure to say: when SLAYER_BAND_TIMES
// asks for the bands' receipts, and not otherwise.
BOOL slayer_keeps_encoder_states(void);
#endif

// slayer_shared_library_count is how many distinct sources the cache
// holds. plain C, so the test can ask it through cgo.
long slayer_shared_library_count(void);

#endif
