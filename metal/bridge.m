// the Objective-C side of the bridge: a device, a queue and one pipeline,
// held in a context Go only ever sees as a pointer. built with ARC; the
// context owns its objects through retained bridges and gives them back
// in slayer_metal_close.

#import <Foundation/Foundation.h>
#import <Metal/Metal.h>
#include <string.h>
#include "bridge.h"

typedef struct {
	void *device;
	void *queue;
	void *pipeline;
	double gpu;
	char name[128];
} context;

// shared storage: on apple silicon the gpu and the cpu read the same
// memory, so nothing is copied across a bus
static const MTLResourceOptions shared = MTLResourceStorageModeShared;

// say writes why something failed into err, cut to fit
static void say(char *err, int errlen, NSString *what) {
	if (err == NULL || errlen <= 0) {
		return;
	}
	strncpy(err, [what UTF8String], errlen - 1);
	err[errlen - 1] = 0;
}

// failed says what failed and the system's own words for why
static void failed(char *err, int errlen, NSString *what, NSError *why) {
	NSString *both = [NSString stringWithFormat:@"%@: %@", what,
	                  why.localizedDescription];
	say(err, errlen, both);
}

// pipeline compiles a kernel from source on a device, or says why not
static id<MTLComputePipelineState> pipeline(id<MTLDevice> device,
                                            const char *source,
                                            const char *kernel,
                                            char *err, int errlen) {
	NSError *why = nil;
	NSString *text = [NSString stringWithUTF8String:source];
	MTLCompileOptions *options = [MTLCompileOptions new];
	id<MTLLibrary> library = [device newLibraryWithSource:text
	                                              options:options
	                                                error:&why];
	if (library == nil) {
		failed(err, errlen, @"compile", why);
		return nil;
	}
	NSString *name = [NSString stringWithUTF8String:kernel];
	id<MTLFunction> function = [library newFunctionWithName:name];
	if (function == nil) {
		NSString *missing =
			[NSString stringWithFormat:@"no kernel named %@", name];
		say(err, errlen, missing);
		return nil;
	}
	id<MTLComputePipelineState> made =
		[device newComputePipelineStateWithFunction:function
		                                      error:&why];
	if (made == nil) {
		failed(err, errlen, @"pipeline", why);
	}
	return made;
}

void *slayer_metal_open(const char *source, const char *kernel, int *nodevice,
                        char *err, int errlen) {
	@autoreleasepool {
		*nodevice = 0;
		id<MTLDevice> device = MTLCreateSystemDefaultDevice();
		if (device == nil) {
			*nodevice = 1;
			say(err, errlen, @"no metal device");
			return NULL;
		}
		id<MTLComputePipelineState> made =
			pipeline(device, source, kernel, err, errlen);
		if (made == nil) {
			return NULL;
		}
		id<MTLCommandQueue> queue = [device newCommandQueue];
		if (queue == nil) {
			say(err, errlen, @"could not make a command queue");
			return NULL;
		}
		context *c = calloc(1, sizeof(context));
		if (c == NULL) {
			say(err, errlen, @"out of memory for the context");
			return NULL;
		}
		c->device = (__bridge_retained void *)device;
		c->queue = (__bridge_retained void *)queue;
		c->pipeline = (__bridge_retained void *)made;
		strncpy(c->name, [device.name UTF8String], sizeof(c->name) - 1);
		return c;
	}
}

const char *slayer_metal_name(void *ctx) {
	return ((context *)ctx)->name;
}

// run encodes one dispatch of the context's pipeline over a grid, with
// the buffers bound in order and the origin after them, and waits for
// it, saying why it failed if it did
static int run(context *c, NSArray<id<MTLBuffer>> *buffers, MTLSize grid,
               const void *origin, int originlen, char *err, int errlen) {
	id<MTLCommandQueue> queue = (__bridge id<MTLCommandQueue>)c->queue;
	id<MTLComputePipelineState> made =
		(__bridge id<MTLComputePipelineState>)c->pipeline;
	id<MTLCommandBuffer> commands = [queue commandBuffer];
	id<MTLComputeCommandEncoder> encoder = [commands computeCommandEncoder];
	if (encoder == nil) {
		say(err, errlen, @"could not make a command");
		return 1;
	}
	[encoder setComputePipelineState:made];
	for (NSUInteger index = 0; index < buffers.count; index++) {
		[encoder setBuffer:buffers[index] offset:0 atIndex:index];
	}
	if (origin != NULL) {
		[encoder setBytes:origin length:(NSUInteger)originlen
		          atIndex:buffers.count];
	}
	NSUInteger across = made.threadExecutionWidth;
	NSUInteger down = made.maxTotalThreadsPerThreadgroup / across;
	if (grid.height == 1) {
		across = made.maxTotalThreadsPerThreadgroup;
		down = 1;
	}
	[encoder dispatchThreads:grid
	   threadsPerThreadgroup:MTLSizeMake(across, down, 1)];
	[encoder endEncoding];
	[commands commit];
	[commands waitUntilCompleted];
	if (commands.error != nil) {
		failed(err, errlen, @"run", commands.error);
		return 1;
	}
	c->gpu += commands.GPUEndTime - commands.GPUStartTime;
	return 0;
}

// buffer is a shared buffer holding a copy of length bytes. a metal
// buffer cannot be empty, so an empty one is one zero byte the kernel is
// told nothing of.
static id<MTLBuffer> buffer(id<MTLDevice> device, const void *bytes,
                            NSUInteger length) {
	static const char zero = 0;
	if (bytes == NULL || length == 0) {
		bytes = &zero;
		length = 1;
	}
	return [device newBufferWithBytes:bytes length:length options:shared];
}

int slayer_metal_points(void *ctx, const float *points, int n,
                        const void *params, int paramlen,
                        float *out, char *err, int errlen) {
	@autoreleasepool {
		context *c = ctx;
		id<MTLDevice> device = (__bridge id<MTLDevice>)c->device;
		if (n <= 0) {
			return 0;
		}
		NSUInteger size = (NSUInteger)n * sizeof(float);
		NSUInteger given = (NSUInteger)paramlen;
		id<MTLBuffer> in = buffer(device, points, size * 3);
		id<MTLBuffer> found = [device newBufferWithLength:size
		                                          options:shared];
		id<MTLBuffer> asked = buffer(device, params, given);
		if (in == nil || found == nil || asked == nil) {
			say(err, errlen, @"could not make the buffers");
			return 1;
		}
		MTLSize grid = MTLSizeMake((NSUInteger)n, 1, 1);
		NSArray<id<MTLBuffer>> *buffers = @[in, found, asked];
		if (run(c, buffers, grid, NULL, 0, err, errlen) != 0) {
			return 1;
		}
		memcpy(out, found.contents, size);
		return 0;
	}
}

// inputs_of makes each input a shared buffer, once for the whole
// picture
static NSMutableArray<id<MTLBuffer>> *inputs_of(id<MTLDevice> device,
                                                const void **inputs,
                                                const long *lengths,
                                                int count) {
	NSMutableArray<id<MTLBuffer>> *made = [NSMutableArray array];
	for (int index = 0; index < count; index++) {
		long length = lengths[index] < 0 ? 0 : lengths[index];
		id<MTLBuffer> one =
			buffer(device, inputs[index], (NSUInteger)length);
		if (one == nil) {
			return nil;
		}
		[made addObject:one];
	}
	return made;
}

// draw_band runs one band of rows from first, and copies its rows of
// row bytes each to out
static int draw_band(context *c, NSArray<id<MTLBuffer>> *buffers,
                     int width, int first, int down, char *out, size_t row,
                     char *err, int errlen) {
	unsigned int origin[2] = {0, (unsigned int)first};
	MTLSize grid = MTLSizeMake((NSUInteger)width, (NSUInteger)down, 1);
	if (run(c, buffers, grid, origin, sizeof(origin), err, errlen) != 0) {
		return 1;
	}
	memcpy(out, buffers.lastObject.contents, row * (size_t)down);
	return 0;
}

// draw_bands draws the whole picture a band of rows at a time into out
static int draw_bands(context *c, NSArray<id<MTLBuffer>> *buffers,
                      int width, int height, int rows, char *out,
                      size_t row, char *err, int errlen) {
	for (int first = 0; first < height; first += rows) {
		int down = MIN(rows, height - first);
		char *at = out + row * (size_t)first;
		if (draw_band(c, buffers, width, first, down, at, row, err,
		              errlen) != 0) {
			return 1;
		}
	}
	return 0;
}

int slayer_metal_grid(void *ctx, int width, int height, int rows, int per,
                      const void **inputs, const long *lengths, int count,
                      float *out, char *err, int errlen) {
	@autoreleasepool {
		context *c = ctx;
		id<MTLDevice> device = (__bridge id<MTLDevice>)c->device;
		NSMutableArray<id<MTLBuffer>> *buffers =
			inputs_of(device, inputs, lengths, count);
		if (buffers == nil) {
			say(err, errlen, @"could not make an input buffer");
			return 1;
		}
		size_t row = (size_t)width * (size_t)per * sizeof(float);
		size_t size = row * (size_t)rows;
		id<MTLBuffer> band =
			[device newBufferWithLength:size options:shared];
		if (band == nil) {
			say(err, errlen, @"could not make the band's buffer");
			return 1;
		}
		[buffers addObject:band];
		char *picture = (char *)out;
		return draw_bands(c, buffers, width, height, rows, picture, row,
		                  err, errlen);
	}
}

double slayer_metal_gpu(void *ctx) {
	return ((context *)ctx)->gpu;
}

void slayer_metal_close(void *ctx) {
	if (ctx == NULL) {
		return;
	}
	context *c = ctx;
	(void)(__bridge_transfer id)c->pipeline;
	(void)(__bridge_transfer id)c->queue;
	(void)(__bridge_transfer id)c->device;
	free(c);
}
