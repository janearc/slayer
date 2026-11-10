#import <Foundation/Foundation.h>
#import <Metal/Metal.h>
#import <objc/runtime.h>
#include <stdio.h>
#include <stdlib.h>

// Only the isolated test process loads this observer. It preserves the
// real wait, timestamps, status and error; no timing value is invented.
static IMP original_wait;

static void observed_wait(id self, SEL selector) {
    ((void (*)(id, SEL))original_wait)(self, selector);
    const char *path = getenv("SLAYER_METAL_TIMING_LOG");
    if (path == NULL) { return; }
    id<MTLCommandBuffer> command = self;
    FILE *out = fopen(path, "a");
    if (out != NULL) {
        fprintf(out, "%.17g %.17g\n",
                command.GPUStartTime, command.GPUEndTime);
        fclose(out);
    }
}

static id<MTLDevice> observed_device(void) {
    id<MTLDevice> device = MTLCreateSystemDefaultDevice();
    if (device == nil) { return nil; }
    id<MTLCommandBuffer> command = [[device newCommandQueue] commandBuffer];
    Class type = object_getClass(command);
    SEL selector = @selector(waitUntilCompleted);
    Method method = class_getInstanceMethod(type, selector);
    if (method != NULL && method_getImplementation(method) != (IMP)observed_wait) {
        original_wait = method_getImplementation(method);
        class_replaceMethod(type, selector, (IMP)observed_wait,
                            method_getTypeEncoding(method));
    }
    return device;
}

__attribute__((used)) static struct {
    const void *replacement;
    const void *original;
} interpose[] __attribute__((section("__DATA,__interpose"))) = {
    {(const void *)observed_device, (const void *)MTLCreateSystemDefaultDevice}
};
