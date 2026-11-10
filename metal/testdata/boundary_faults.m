#import <Foundation/Foundation.h>
#import <Metal/Metal.h>
#import <objc/runtime.h>
#include <stdlib.h>
#include <string.h>

// Only the test subprocess loads this library. Production has no fault hook.
static BOOL unavailable(id self, SEL selector) { return NO; }
static id no_length_buffer(id self, SEL selector, NSUInteger size,
                           MTLResourceOptions options) { return nil; }
static id no_bytes_buffer(id self, SEL selector, const void *bytes,
                          NSUInteger size, MTLResourceOptions options) {
    return nil;
}
static NSError *command_error(id self, SEL selector) {
    return [NSError errorWithDomain:@"slayer-boundary-test" code:1
        userInfo:@{NSLocalizedDescriptionKey: @"injected command failure"}];
}

// A deliberately tiny device limit holds byte-count endpoints without a
// large allocation. This replaces only the reported capability in this child.
static NSUInteger small_buffer_limit(id self, SEL selector) { return 16; }

static IMP original_library;
static IMP original_function;
typedef id (*named_factory)(id, SEL, NSString *) NS_RETURNS_RETAINED;
typedef id (*library_factory)(id, SEL, NSString *, MTLCompileOptions *,
                              NSError **) NS_RETURNS_RETAINED;
static id without_smoke(id self, SEL selector, NSString *name)
    NS_RETURNS_RETAINED;
static id without_smoke(id self, SEL selector, NSString *name) {
    if ([name isEqualToString:@"smoke_light"]) { return nil; }
    return ((named_factory)original_function)(self, selector, name);
}
static id library_without_smoke(id self, SEL selector, NSString *source,
                                MTLCompileOptions *options, NSError **error)
    NS_RETURNS_RETAINED;
static id library_without_smoke(id self, SEL selector, NSString *source,
                                MTLCompileOptions *options, NSError **error) {
    id library = ((library_factory)original_library)(self, selector, source,
                                                   options, error);
    Class type = object_getClass(library);
    SEL function = @selector(newFunctionWithName:);
    Method method = class_getInstanceMethod(type, function);
    if (method != NULL && method_getImplementation(method) != (IMP)without_smoke) {
        original_function = method_getImplementation(method);
        class_replaceMethod(type, function, (IMP)without_smoke,
                            method_getTypeEncoding(method));
    }
    return library;
}

static id<MTLDevice> boundary_device(void) {
    const char *mode = getenv("SLAYER_METAL_BOUNDARY_FAULT");
    if (mode != NULL && strcmp(mode, "no-device") == 0) { return nil; }
    id<MTLDevice> device = MTLCreateSystemDefaultDevice();
    Class type = object_getClass(device);
    if (mode != NULL && strcmp(mode, "no-smoke-kernel") == 0) {
        SEL library = @selector(newLibraryWithSource:options:error:);
        Method method = class_getInstanceMethod(type, library);
        if (method_getImplementation(method) != (IMP)library_without_smoke) {
            original_library = method_getImplementation(method);
            class_replaceMethod(type, library, (IMP)library_without_smoke,
                                method_getTypeEncoding(method));
        }
    }
    if (mode != NULL && strcmp(mode, "no-raytracing") == 0) {
        class_replaceMethod(type, @selector(supportsRaytracing),
                            (IMP)unavailable, "B@:");
    }
if (mode != NULL && strcmp(mode, "small-buffer-limit") == 0) {
    SEL limit = @selector(maxBufferLength);
    class_replaceMethod(type, limit, (IMP)small_buffer_limit,
                        method_getTypeEncoding(class_getInstanceMethod(type, limit)));
}
    if (mode != NULL && strcmp(mode, "no-buffers") == 0) {
        SEL length = @selector(newBufferWithLength:options:);
        SEL bytes = @selector(newBufferWithBytes:length:options:);
        class_replaceMethod(type, length, (IMP)no_length_buffer,
                            method_getTypeEncoding(class_getInstanceMethod(type, length)));
        class_replaceMethod(type, bytes, (IMP)no_bytes_buffer,
                            method_getTypeEncoding(class_getInstanceMethod(type, bytes)));
    }
    if (mode != NULL && strcmp(mode, "command-error") == 0) {
        id<MTLCommandBuffer> commands = [[device newCommandQueue] commandBuffer];
        Class command_type = object_getClass(commands);
        SEL error = @selector(error);
        class_replaceMethod(command_type, error, (IMP)command_error,
                            method_getTypeEncoding(class_getInstanceMethod(command_type, error)));
    }
    return device;
}

__attribute__((used)) static struct {
    const void *replacement;
    const void *original;
} interpose[] __attribute__((section("__DATA,__interpose"))) = {
    {(const void *)boundary_device, (const void *)MTLCreateSystemDefaultDevice}
};
