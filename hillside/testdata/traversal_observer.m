#import <Foundation/Foundation.h>
#import <Metal/Metal.h>
#import <objc/runtime.h>
#include <pthread.h>
#include <stdio.h>
#include <stdlib.h>

// Only an isolated test child loads this observer. It records the flags
// actually passed to Metal's compiler and returns the original library.
// It changes no shader source, options, result, error or command.
static IMP original_new_library;
static pthread_mutex_t observer_lock = PTHREAD_MUTEX_INITIALIZER;

// Preserve the new method family's ownership convention in both calls.
// https://clang.llvm.org/docs/AutomaticReferenceCounting.html
typedef id<MTLLibrary> NewLibraryCall(id, SEL, NSString *,
        MTLCompileOptions *, NSError *__autoreleasing *) NS_RETURNS_RETAINED;
static id<MTLLibrary> observed_new_library(id self, SEL selector,
        NSString *source, MTLCompileOptions *options,
        NSError *__autoreleasing *error) NS_RETURNS_RETAINED;

static id<MTLLibrary> observed_new_library(id self, SEL selector,
        NSString *source, MTLCompileOptions *options,
        NSError *__autoreleasing *error) {
    id<MTLLibrary> library = ((NewLibraryCall *)original_new_library)(
        self, selector, source, options, error);
    const char *path = getenv("SLAYER_TRAVERSAL_OBSERVER_LOG");
    if (library != nil && path != NULL &&
            [source containsString:@"kernel void trace("]) {
        int primary = [source containsString:
            @"#define SLAYER_NATIVE_MESHES 1\n"];
        int shadows = [source containsString:
            @"#define SLAYER_NATIVE_SHADOWS 1\n"];
        pthread_mutex_lock(&observer_lock);
        FILE *out = fopen(path, "a");
        if (out != NULL) {
            fprintf(out, "%d %d\n", primary, shadows);
            fclose(out);
        }
        pthread_mutex_unlock(&observer_lock);
    }
    return library;
}

static id<MTLDevice> observed_device(void) {
    id<MTLDevice> device = MTLCreateSystemDefaultDevice();
    if (device == nil) { return nil; }
    pthread_mutex_lock(&observer_lock);
    Class type = object_getClass(device);
    SEL selector = @selector(newLibraryWithSource:options:error:);
    Method method = class_getInstanceMethod(type, selector);
    if (method != NULL && method_getImplementation(method) !=
            (IMP)observed_new_library && original_new_library == NULL) {
        original_new_library = method_getImplementation(method);
        class_replaceMethod(type, selector, (IMP)observed_new_library,
                            method_getTypeEncoding(method));
    }
    pthread_mutex_unlock(&observer_lock);
    return device;
}

__attribute__((used)) static struct {
    const void *replacement;
    const void *original;
} interpose[] __attribute__((section("__DATA,__interpose"))) = {
    {(const void *)observed_device, (const void *)MTLCreateSystemDefaultDevice}
};
