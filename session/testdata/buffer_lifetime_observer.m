#import <Foundation/Foundation.h>
#import <Metal/Metal.h>
#import <objc/runtime.h>
#include <dlfcn.h>
#include <limits.h>
#include <string.h>
#include <pthread.h>
#include <stdio.h>
#include <stdlib.h>

// This observer is loaded only by an isolated test child. It returns the
// original buffer and keeps no reference to it after returning. A marker records
// its release when the buffer dies, following Apple's association lifetime:
// https://developer.apple.com/library/archive/documentation/Cocoa/Conceptual/ObjectiveC/Chapters/ocAssociativeReferences.html
// No allocation answer, command result or kernel value is replaced.
static char marker_key, owner_key;
static IMP original_add_object;
static pthread_mutex_t log_lock = PTHREAD_MUTEX_INITIALIZER;
static pthread_mutex_t hook_lock = PTHREAD_MUTEX_INITIALIZER;
static unsigned long next_identifier;
static IMP original_new_buffer;

static void record_channel(const char *channel, const char *event, unsigned long identifier) {
    const char *base = getenv("SLAYER_BUFFER_LIFETIME_LOG");
    char path[PATH_MAX];
    if (base == NULL) { return; }
    snprintf(path, sizeof(path), "%s%s", base, channel);
    pthread_mutex_lock(&log_lock);
    FILE *out = fopen(path, "a");
    if (out != NULL) {
        fprintf(out, "%s %lu\n", event, identifier);
        fclose(out);
    }
    pthread_mutex_unlock(&log_lock);
}

@interface SlayerBufferLifetime : NSObject
@property(nonatomic) unsigned long identifier;
@property(nonatomic, copy) NSString *channel;
@end

@implementation SlayerBufferLifetime
- (void)dealloc {
    record_channel(_channel.UTF8String, "free", _identifier);
}
@end

// The method belongs to the new family. Preserve its retained-return ABI
// in both the original IMP call and the replacement function.
// https://clang.llvm.org/docs/AutomaticReferenceCounting.html
typedef id<MTLBuffer> NewBufferCall(id, SEL, NSUInteger,
                                  MTLResourceOptions) NS_RETURNS_RETAINED;
static id<MTLBuffer> observed_new_buffer(id self, SEL selector,
        NSUInteger length, MTLResourceOptions options) NS_RETURNS_RETAINED;

static id<MTLBuffer> observed_new_buffer(id self, SEL selector,
        NSUInteger length, MTLResourceOptions options) {
    id<MTLBuffer> buffer = ((NewBufferCall *)original_new_buffer)(
        self, selector, length, options);
    // The actual 8x8 fuel bed keeps two interleaved float32 state buffers.
    if (buffer != nil && length == 8 * 8 * 2 * sizeof(float)) {
        SlayerBufferLifetime *marker = [SlayerBufferLifetime new];
        pthread_mutex_lock(&log_lock);
        marker.identifier = ++next_identifier;
        pthread_mutex_unlock(&log_lock);
        marker.channel = @".buffers";
        record_channel(".buffers", "new", marker.identifier);
        objc_setAssociatedObject(buffer, &marker_key, marker,
                                 OBJC_ASSOCIATION_RETAIN_NONATOMIC);
    }
    return buffer;
}

// The session retains its array; a driver may independently retain buffers.
// Observe only the array called from the actual C session allocation path.
// Markers retain neither the array nor any buffer.
static void observed_add_object(id self, SEL selector, id object) {
    const void *caller = __builtin_return_address(0);
    ((void (*)(id, SEL, id))original_add_object)(self, selector, object);
    if (objc_getAssociatedObject(object, &marker_key) == nil) { return; }
    Dl_info info;
    const char *name = dladdr(caller, &info) && info.dli_sname != NULL
        ? info.dli_sname : "(unknown)";
    const char *base = getenv("SLAYER_BUFFER_LIFETIME_LOG");
    if (base != NULL) {
        char path[PATH_MAX];
        snprintf(path, sizeof(path), "%s.callers", base);
        pthread_mutex_lock(&log_lock);
        FILE *out = fopen(path, "a");
        if (out != NULL) { fprintf(out, "%s\n", name); fclose(out); }
        pthread_mutex_unlock(&log_lock);
    }
    if (strcmp(name, "slayer_session_buffer") != 0 &&
            strcmp(name, "_slayer_session_buffer") != 0) { return; }
    if (objc_getAssociatedObject(self, &owner_key) == nil) {
        SlayerBufferLifetime *marker = [SlayerBufferLifetime new];
        marker.channel = @"";
        pthread_mutex_lock(&log_lock);
        marker.identifier = ++next_identifier;
        pthread_mutex_unlock(&log_lock);
        record_channel("", "new", marker.identifier);
        objc_setAssociatedObject(self, &owner_key, marker,
                                 OBJC_ASSOCIATION_RETAIN_NONATOMIC);
    }
}

static id<MTLDevice> observed_device(void) {
    id<MTLDevice> device = MTLCreateSystemDefaultDevice();
    if (device == nil) { return nil; }
    pthread_mutex_lock(&hook_lock);
    Class type = object_getClass(device);
    SEL selector = @selector(newBufferWithLength:options:);
    Method method = class_getInstanceMethod(type, selector);
    if (method != NULL && method_getImplementation(method) !=
            (IMP)observed_new_buffer) {
        if (original_new_buffer == NULL) {
            original_new_buffer = method_getImplementation(method);
            class_replaceMethod(type, selector, (IMP)observed_new_buffer,
                                method_getTypeEncoding(method));
        } else {
            // Do not call another device class's IMP or hide missing data.
            record_channel("", "unsupported-device-class", 0);
        }
    }
    if (original_add_object == NULL) {
        Class list = object_getClass([NSMutableArray array]);
        SEL append = @selector(addObject:);
        Method add = class_getInstanceMethod(list, append);
        if (add != NULL) {
            original_add_object = method_getImplementation(add);
            class_replaceMethod(list, append, (IMP)observed_add_object,
                                method_getTypeEncoding(add));
        }
    }
    pthread_mutex_unlock(&hook_lock);
    return device;
}

__attribute__((used)) static struct {
    const void *replacement;
    const void *original;
} interpose[] __attribute__((section("__DATA,__interpose"))) = {
    {(const void *)observed_device, (const void *)MTLCreateSystemDefaultDevice}
};
