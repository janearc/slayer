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
// own error says. It is NO when the limit passed first, with why saying
// what was waited for and for how long: the command is then still queued
// or running on the device, which may stay busy until it ends.
BOOL slayer_wait_command(id<MTLCommandBuffer> commands, double limit,
                         NSString *what, NSString **why);

// slayer_stuck is the sentence a caller says when it will not use a
// context or session again because a command outran its limit.
NSString *slayer_stuck(void);
#endif

// slayer_shared_library_count is how many distinct sources the cache
// holds. plain C, so the test can ask it through cgo.
long slayer_shared_library_count(void);

#endif
