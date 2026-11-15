// the Objective-C side of the bridge: a device, a queue and one pipeline,
// held in a context Go only ever sees as a pointer. built with ARC; the
// context owns its objects through retained bridges and gives them back
// in slayer_metal_close.

#import <Foundation/Foundation.h>
#import <Metal/Metal.h>
#include <string.h>
#include "bridge.h"
#include "library.h"

typedef struct {
	void *device;
	void *queue;
	void *pipeline;
	void *library;
	void *pipelines;
	void *accelerations;
	void *declared; // kernel name to the bytes each buffer index must hold
	int acceleration_root;
	double gpu;
	double limit;   // seconds one command may take before the wait gives up
	int stuck;      // set when a command outran its limit; the context is done
	char kernel[64]; // the selected entry point, named in a timeout's error
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

// declared_sizes is how many bytes the kernel declares for each of its
// buffer arguments, by the argument's index, from the pipeline's reflection
static NSDictionary *declared_sizes(MTLComputePipelineReflection *reflection) {
	NSMutableDictionary *sizes = [NSMutableDictionary dictionary];
	for (id<MTLBinding> binding in reflection.bindings) {
		if (binding.type == MTLBindingTypeBuffer) {
			id<MTLBufferBinding> buffer = (id<MTLBufferBinding>)binding;
			sizes[@(buffer.index)] = @(buffer.bufferDataSize);
		}
	}
	return sizes;
}

// pipeline compiles a kernel from source on a device, or says why not
static id<MTLComputePipelineState> pipeline(id<MTLDevice> device,
                                            const char *source,
                                            const char *kernel,
                                            char *err, int errlen,
                                            id<MTLLibrary> __autoreleasing *kept,
                                            NSDictionary **sizes) {
	NSError *why = nil;
	id<MTLLibrary> library = slayer_shared_library(device, source, &why);
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
	MTLComputePipelineReflection *reflection = nil;
	id<MTLComputePipelineState> made =
		[device newComputePipelineStateWithFunction:function
		                                    options:MTLPipelineOptionBindingInfo
		                                 reflection:&reflection
		                                      error:&why];
	if (made == nil) {
		failed(err, errlen, @"pipeline", why);
	}
	if (made != nil) {
		*kept = library;
		*sizes = declared_sizes(reflection);
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
		id<MTLLibrary> compiled = nil;
		NSDictionary *sizes = nil;
		id<MTLComputePipelineState> made =
			pipeline(device, source, kernel, err, errlen, &compiled, &sizes);
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
  c->library = (__bridge_retained void *)compiled;
  NSMutableDictionary *pipelines = [NSMutableDictionary dictionary];
  pipelines[[NSString stringWithUTF8String:kernel]] = made;
  c->pipelines = (__bridge_retained void *)pipelines;
  NSMutableDictionary *declared = [NSMutableDictionary dictionary];
  declared[[NSString stringWithUTF8String:kernel]] = sizes;
  c->declared = (__bridge_retained void *)declared;
		strncpy(c->name, [device.name UTF8String], sizeof(c->name) - 1);
		strncpy(c->kernel, kernel, sizeof(c->kernel) - 1);
		return c;
	}
}

// Entry points share the device, queue, compiled library and acceleration tree.
int slayer_metal_function(void *ctx, const char *kernel, char *err, int errlen) {
 @autoreleasepool {
  context *c = ctx;
  NSString *name = [NSString stringWithUTF8String:kernel];
  NSMutableDictionary *pipelines = (__bridge NSMutableDictionary *)c->pipelines;
  id<MTLComputePipelineState> made = pipelines[name];
  if (made == nil) {
   id<MTLLibrary> library = (__bridge id<MTLLibrary>)c->library;
   id<MTLFunction> function = [library newFunctionWithName:name];
   if (function == nil) {
    say(err, errlen, [NSString stringWithFormat:@"no kernel named %@", name]);
    return 1;
   }
   id<MTLDevice> device = (__bridge id<MTLDevice>)c->device;
   NSError *why = nil;
   MTLComputePipelineReflection *reflection = nil;
   made = [device newComputePipelineStateWithFunction:function
                                              options:MTLPipelineOptionBindingInfo
                                           reflection:&reflection
                                                error:&why];
   if (made == nil) { failed(err, errlen, @"pipeline", why); return 1; }
   pipelines[name] = made;
   NSMutableDictionary *declared = (__bridge NSMutableDictionary *)c->declared;
   declared[name] = declared_sizes(reflection);
  }
  (void)(__bridge_transfer id)c->pipeline;
  c->pipeline = (__bridge_retained void *)made;
  strncpy(c->kernel, kernel, sizeof(c->kernel) - 1);
  return 0;
 }
}

const char *slayer_metal_name(void *ctx) {
	return ((context *)ctx)->name;
}

void slayer_metal_limit(void *ctx, double seconds) {
	((context *)ctx)->limit = seconds;
}

// kernel_named is a phrase naming the context's selected kernel, for an
// error, with what it was doing after it
static NSString *kernel_named(context *c, NSString *doing) {
	return [NSString stringWithFormat:@"kernel \"%s\" %@", c->kernel, doing];
}

// encode commits one dispatch of the context's pipeline over a grid,
// with the buffers bound in order and the origin after them, and gives
// back the committed command without waiting, or nil saying why not
static id<MTLCommandBuffer> encode(context *c,
                                   NSArray<id<MTLBuffer>> *buffers,
                                   MTLSize grid, const void *origin,
                                   int originlen, char *err, int errlen) {
	if (c->stuck) {
		say(err, errlen, slayer_stuck());
		return nil;
	}
	if (c->accelerations != NULL && origin == NULL) {
		say(err, errlen, @"an accelerated kernel requires Grid");
		return nil;
	}
	id<MTLCommandQueue> queue = (__bridge id<MTLCommandQueue>)c->queue;
	id<MTLComputePipelineState> made =
		(__bridge id<MTLComputePipelineState>)c->pipeline;
	id<MTLCommandBuffer> commands = [queue commandBuffer];
	if (commands == nil) {
		say(err, errlen, @"could not make a command buffer");
		return nil;
	}
	id<MTLComputeCommandEncoder> encoder = [commands computeCommandEncoder];
	if (encoder == nil) {
		say(err, errlen, @"could not make a command");
		return nil;
	}
	[encoder setComputePipelineState:made];
	for (NSUInteger index = 0; index < buffers.count; index++) {
		[encoder setBuffer:buffers[index] offset:0 atIndex:index];
	}
	if (origin != NULL) {
		[encoder setBytes:origin length:(NSUInteger)originlen
		          atIndex:buffers.count];
	}
	if (c->accelerations != NULL) {
		if (@available(macOS 14.0, *)) {
			NSArray<id<MTLAccelerationStructure>> *all =
				(__bridge NSArray *)c->accelerations;
			[encoder setAccelerationStructure:all[c->acceleration_root]
			          atBufferIndex:buffers.count + 1];
			for (id<MTLAccelerationStructure> structure in all) {
				[encoder useResource:structure usage:MTLResourceUsageRead];
			}
		}
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
	return commands;
}

// await waits a committed command out for at most the context's limit,
// counts its time on the gpu, and says why it failed if it did, or which
// command it gave up on. after giving up the context is not used again.
static int await(context *c, id<MTLCommandBuffer> commands, NSString *what,
                 char *err, int errlen) {
	NSString *why = nil;
	if (!slayer_wait_command(commands, c->limit, what, &why)) {
		c->stuck = 1;
		say(err, errlen, why);
		return 1;
	}
	if (commands.error != nil) {
		failed(err, errlen, @"run", commands.error);
		return 1;
	}
	c->gpu += commands.GPUEndTime - commands.GPUStartTime;
	return 0;
}

// run encodes one dispatch and waits for it
static int run(context *c, NSArray<id<MTLBuffer>> *buffers, MTLSize grid,
               const void *origin, int originlen, char *err, int errlen) {
	id<MTLCommandBuffer> commands =
		encode(c, buffers, grid, origin, originlen, err, errlen);
	if (commands == nil) {
		return 1;
	}
	return await(c, commands, kernel_named(c, @"dispatch"), err, errlen);
}

// declared_for is how many bytes the context's selected kernel declares
// for the buffer at an index, or zero where it declares none
static NSUInteger declared_for(context *c, int index) {
	NSDictionary *all = (__bridge NSDictionary *)c->declared;
	NSDictionary *sizes = all[[NSString stringWithUTF8String:c->kernel]];
	return [sizes[@(index)] unsignedIntegerValue];
}

// buffer is a shared buffer holding a copy of length bytes. a metal
// buffer cannot be empty, so an empty one is zeros as many as the kernel
// declares for it, at least one, which is what API validation checks a
// bound buffer against.
static id<MTLBuffer> buffer(id<MTLDevice> device, const void *bytes,
                            NSUInteger length, NSUInteger declared) {
	if (bytes == NULL || length == 0) {
		NSUInteger size = declared > 0 ? declared : 1;
		return [device newBufferWithLength:size options:shared];
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
		id<MTLBuffer> in = buffer(device, points, size * 3, declared_for(c, 0));
		id<MTLBuffer> found = [device newBufferWithLength:size
		                                          options:shared];
		id<MTLBuffer> asked = buffer(device, params, given, declared_for(c, 2));
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

// grid_stream owns the retained input buffers, two band buffers the
// gpu and the cpu take turns with, and at most one committed command
// not yet waited out; the Go call keeps its kernel.
typedef struct {
	context *kernel;
	void *inputs_held;
	void *bands[2];
	void *flying; // the committed command for the band drawn ahead
	int flying_first, flying_down, flying_band;
	int width, height, rows;
	int expected_inputs, inputs;
	size_t row_bytes;
} grid_stream;

// commit_band commits the kernel over one band of rows from first,
// writing into one of the stream's two band buffers, without waiting
static id<MTLCommandBuffer> commit_band(grid_stream *stream, int band,
                                        int first, int down, char *err,
                                        int errlen) {
	NSArray<id<MTLBuffer>> *inputs =
		(__bridge NSArray<id<MTLBuffer>> *)stream->inputs_held;
	id<MTLBuffer> out = (__bridge id<MTLBuffer>)stream->bands[band];
	NSArray<id<MTLBuffer>> *buffers = [inputs arrayByAddingObject:out];
	unsigned int origin[2] = {0, (unsigned int)first};
	MTLSize grid = MTLSizeMake((NSUInteger)stream->width,
	                           (NSUInteger)down, 1);
	return encode(stream->kernel, buffers, grid, origin, sizeof(origin),
	              err, errlen);
}

// band_named is a phrase naming a band of rows in the context's kernel
static NSString *band_named(context *c, int first, int down) {
	NSString *rows = [NSString stringWithFormat:@"band of rows %d to %d",
	                  first, first + down - 1];
	return kernel_named(c, rows);
}

// settle waits out a command drawn ahead that the caller did not ask
// for, counting its time, so the stream never holds two in flight. it
// is non-zero, with err saying why, when that command failed or timed out.
static int settle(grid_stream *stream, char *err, int errlen) {
	if (stream->flying == NULL) {
		return 0;
	}
	id<MTLCommandBuffer> flying = (__bridge_transfer id)stream->flying;
	stream->flying = NULL;
	NSString *what =
		band_named(stream->kernel, stream->flying_first, stream->flying_down);
	return await(stream->kernel, flying, what, err, errlen);
}

void *slayer_metal_grid_begin(void *ctx, int width, int height, int rows,
                             int per, int count, char *err, int errlen) {
	@autoreleasepool {
		if (ctx == NULL || width <= 0 || height <= 0 || rows <= 0 ||
		    per <= 0 || count < 0) {
			say(err, errlen, @"invalid streamed grid preparation");
			return NULL;
		}
		context *c = ctx;
		if ((size_t)width > SIZE_MAX / sizeof(float) / (size_t)per ||
		    (size_t)rows > SIZE_MAX / ((size_t)width * (size_t)per * sizeof(float))) {
			say(err, errlen, @"streamed grid size overflow");
			return NULL;
		}
		id<MTLDevice> device = (__bridge id<MTLDevice>)c->device;
		size_t row_bytes = (size_t)width * (size_t)per * sizeof(float);
		grid_stream *stream = calloc(1, sizeof(grid_stream));
		if (stream == NULL) {
			say(err, errlen, @"could not allocate a grid stream");
			return NULL;
		}
		// two band buffers: the one the caller reads and the one the
		// gpu draws the next band into meanwhile
		for (int band = 0; band < 2; band++) {
			id<MTLBuffer> made =
				[device newBufferWithLength:row_bytes * (size_t)rows
				                    options:shared];
			if (made == nil) {
				say(err, errlen, @"could not make a streamed output buffer");
				slayer_metal_grid_close(stream);
				return NULL;
			}
			stream->bands[band] = (__bridge_retained void *)made;
		}
		stream->kernel = c;
		stream->inputs_held =
			(__bridge_retained void *)[NSMutableArray array];
		stream->width = width;
		stream->height = height;
		stream->rows = rows;
		stream->expected_inputs = count;
		stream->row_bytes = row_bytes;
		return stream;
	}
}

int slayer_metal_grid_add(void *handle, const void *input, long length,
                          char *err, int errlen) {
	@autoreleasepool {
		grid_stream *stream = handle;
		if (stream == NULL || stream->inputs >= stream->expected_inputs || length < 0 ||
		    (length > 0 && input == NULL)) {
			say(err, errlen, @"invalid streamed grid input");
			return 1;
		}
		id<MTLDevice> device = (__bridge id<MTLDevice>)stream->kernel->device;
		id<MTLBuffer> one = buffer(device, input, (NSUInteger)length,
		                           declared_for(stream->kernel, stream->inputs));
		if (one == nil) {
			say(err, errlen, @"could not make a streamed input buffer");
			return 1;
		}
		NSMutableArray<id<MTLBuffer>> *inputs =
			(__bridge NSMutableArray<id<MTLBuffer>> *)stream->inputs_held;
		[inputs addObject:one];
		stream->inputs++;
		return 0;
	}
}

int slayer_metal_grid_update(void *handle, int index, const void *input,
                             long length, char *err, int errlen) {
	@autoreleasepool {
		grid_stream *stream = handle;
		if (stream == NULL || index < 0 || index >= stream->inputs ||
		    length < 0 || (length > 0 && input == NULL)) {
			say(err, errlen, @"invalid streamed grid update");
			return 1;
		}
		// nothing may be in flight while an input it reads changes
		if (settle(stream, err, errlen) != 0) {
			return 1;
		}
		NSMutableArray<id<MTLBuffer>> *inputs =
			(__bridge NSMutableArray<id<MTLBuffer>> *)stream->inputs_held;
		NSUInteger declared = declared_for(stream->kernel, index);
		id<MTLBuffer> held = inputs[(NSUInteger)index];
		if (input == NULL || length == 0) {
			// an empty input is as many zeros as the kernel declares
			NSUInteger size = declared > 0 ? declared : 1;
			if (size == held.length) {
				memset(held.contents, 0, size);
				return 0;
			}
		} else if ((NSUInteger)length == held.length) {
			memcpy(held.contents, input, (size_t)length);
			return 0;
		}
		id<MTLDevice> device =
			(__bridge id<MTLDevice>)stream->kernel->device;
		id<MTLBuffer> made = buffer(device, input, (NSUInteger)length, declared);
		if (made == nil) {
			say(err, errlen, @"could not make a streamed input buffer");
			return 1;
		}
		inputs[(NSUInteger)index] = made;
		return 0;
	}
}

int slayer_metal_grid_same(void *handle, int index, const void *input,
                           long length, int *same, char *err, int errlen) {
	@autoreleasepool {
		grid_stream *stream = handle;
		*same = 0;
		if (stream == NULL || index < 0 || index >= stream->inputs ||
		    length <= 0 || input == NULL) {
			say(err, errlen, @"invalid streamed grid comparison");
			return 1;
		}
		NSArray<id<MTLBuffer>> *inputs =
			(__bridge NSArray<id<MTLBuffer>> *)stream->inputs_held;
		id<MTLBuffer> held = inputs[(NSUInteger)index];
		if (held.length == (NSUInteger)length &&
		    memcmp(held.contents, input, (size_t)length) == 0) {
			*same = 1;
		}
		return 0;
	}
}

int slayer_metal_grid_refresh(void *handle, int index, const void *input,
                              long length, long chunk, long *sent, char *err,
                              int errlen) {
	@autoreleasepool {
		grid_stream *stream = handle;
		*sent = 0;
		if (stream == NULL || index < 0 || index >= stream->inputs ||
		    length <= 0 || input == NULL || chunk <= 0) {
			say(err, errlen, @"invalid streamed grid refresh");
			return 1;
		}
		NSArray<id<MTLBuffer>> *inputs =
			(__bridge NSArray<id<MTLBuffer>> *)stream->inputs_held;
		id<MTLBuffer> held = inputs[(NSUInteger)index];
		if (held.length != (NSUInteger)length) {
			say(err, errlen, @"a refresh needs an input of the held length");
			return 1;
		}
		// nothing may be in flight while an input it reads changes
		if (settle(stream, err, errlen) != 0) {
			return 1;
		}
		char *into = held.contents;
		const char *from = input;
		for (long at = 0; at < length; at += chunk) {
			long part = length - at < chunk ? length - at : chunk;
			if (memcmp(into + at, from + at, (size_t)part) != 0) {
				memcpy(into + at, from + at, (size_t)part);
				*sent += part;
			}
		}
		return 0;
	}
}

// band_buffers makes the stream's two band buffers of bytes each, or
// is non-zero with err saying why, leaving the old ones in place
static int band_buffers(grid_stream *stream, size_t bytes, char *err,
                        int errlen) {
	id<MTLDevice> device = (__bridge id<MTLDevice>)stream->kernel->device;
	id<MTLBuffer> made[2];
	for (int band = 0; band < 2; band++) {
		made[band] = [device newBufferWithLength:bytes options:shared];
		if (made[band] == nil) {
			say(err, errlen, @"could not make a streamed output buffer");
			return 1;
		}
	}
	for (int band = 0; band < 2; band++) {
		(void)(__bridge_transfer id)stream->bands[band];
		stream->bands[band] = (__bridge_retained void *)made[band];
	}
	return 0;
}

int slayer_metal_grid_shape(void *handle, int width, int height, int rows,
                            int per, char *err, int errlen) {
	@autoreleasepool {
		grid_stream *stream = handle;
		if (stream == NULL || width <= 0 || height <= 0 || rows <= 0 ||
		    per <= 0) {
			say(err, errlen, @"invalid streamed grid shape");
			return 1;
		}
		if ((size_t)width > SIZE_MAX / sizeof(float) / (size_t)per ||
		    (size_t)rows > SIZE_MAX /
		                   ((size_t)width * (size_t)per * sizeof(float))) {
			say(err, errlen, @"streamed grid size overflow");
			return 1;
		}
		// nothing drawn ahead may land in a band of the old shape
		if (settle(stream, err, errlen) != 0) {
			return 1;
		}
		size_t row_bytes = (size_t)width * (size_t)per * sizeof(float);
		size_t old = stream->row_bytes * (size_t)stream->rows;
		if (row_bytes * (size_t)rows != old &&
		    band_buffers(stream, row_bytes * (size_t)rows, err, errlen) != 0) {
			return 1;
		}
		stream->width = width;
		stream->height = height;
		stream->rows = rows;
		stream->row_bytes = row_bytes;
		return 0;
	}
}

int slayer_metal_grid_band(void *handle, int first, int down, float *out,
                           char *err, int errlen) {
	@autoreleasepool {
		grid_stream *stream = handle;
		if (stream == NULL || stream->inputs != stream->expected_inputs ||
		    first < 0 || down <= 0 || down > stream->rows) {
			say(err, errlen, @"invalid streamed grid band");
			return 1;
		}
		// the asked band is usually the one drawn ahead; any other
		// guess is waited out and dropped, and the ask drawn fresh
		id<MTLCommandBuffer> ready = nil;
		int band = 0;
		if (stream->flying != NULL && stream->flying_first == first &&
		    stream->flying_down == down) {
			ready = (__bridge_transfer id)stream->flying;
			stream->flying = NULL;
			band = stream->flying_band;
		} else {
			if (settle(stream, err, errlen) != 0) {
				return 1;
			}
			ready = commit_band(stream, band, first, down, err, errlen);
			if (ready == nil) {
				return 1;
			}
		}
		// the next band is committed before this one is read, so the
		// gpu draws ahead while the cpu copies and consumes. a band that
		// cannot be committed is an error, as the ask would be.
		int next_first = first + down;
		if (next_first < stream->height) {
			int next_down = stream->height - next_first;
			if (next_down > stream->rows) {
				next_down = stream->rows;
			}
			int other = 1 - band;
			id<MTLCommandBuffer> flying = commit_band(stream, other,
				next_first, next_down, err, errlen);
			if (flying == nil) {
				return 1;
			}
			stream->flying = (__bridge_retained void *)flying;
			stream->flying_first = next_first;
			stream->flying_down = next_down;
			stream->flying_band = other;
		}
		if (await(stream->kernel, ready, band_named(stream->kernel, first, down),
		          err, errlen) != 0) {
			return 1;
		}
		id<MTLBuffer> drawn = (__bridge id<MTLBuffer>)stream->bands[band];
		memcpy(out, drawn.contents, stream->row_bytes * (size_t)down);
		return 0;
	}
}

void slayer_metal_grid_close(void *handle) {
	if (handle == NULL) { return; }
	grid_stream *stream = handle;
	// under a pool, as every call from Go is: what these objects
	// autorelease as they go has nowhere else to drain
	@autoreleasepool {
		if (stream->kernel != NULL && stream->kernel->stuck) {
			// nothing more is waited for: the device may never finish it
			(void)(__bridge_transfer id)stream->flying;
			stream->flying = NULL;
		}
		char ignored[1];
		settle(stream, ignored, sizeof(ignored));
		(void)(__bridge_transfer id)stream->inputs_held;
		(void)(__bridge_transfer id)stream->bands[0];
		(void)(__bridge_transfer id)stream->bands[1];
	}
	free(stream);
}

double slayer_metal_gpu(void *ctx) {
	return ((context *)ctx)->gpu;
}

void slayer_metal_close(void *ctx) {
	if (ctx == NULL) {
		return;
	}
	context *c = ctx;
	// under a pool, as every call from Go is: what these objects
	// autorelease as they go has nowhere else to drain
	@autoreleasepool {
		(void)(__bridge_transfer id)c->accelerations;
		(void)(__bridge_transfer id)c->pipeline;
		(void)(__bridge_transfer id)c->pipelines;
		(void)(__bridge_transfer id)c->declared;
		(void)(__bridge_transfer id)c->library;
		(void)(__bridge_transfer id)c->queue;
		(void)(__bridge_transfer id)c->device;
	}
	free(c);
}

// Each node is a triangle range or placements of earlier nodes. Building in
// that order keeps dependencies explicit. Resources stay owned by the kernel.
// accelerate builds a hierarchy and gives it to the context. With replace
// it takes the place of one already there, which stays until the new one
// is built; without, a context that has one is refused.
static int accelerate(void *ctx, const void *vertices, long length,
                      int stride, const int32_t *nodes, int count,
                      const float *matrices, const uint32_t *references,
                      int root, int replace, int *unavailable, char *err,
                      int errlen) {
	@autoreleasepool {
		*unavailable = 0;
		context *c = ctx;
		if (c->accelerations != NULL && !replace) {
			say(err, errlen, @"the kernel already has an acceleration structure");
			return 1;
		}
		if (@available(macOS 14.0, *)) {
			id<MTLDevice> device = (__bridge id<MTLDevice>)c->device;
			if (!device.supportsRaytracing) {
				*unavailable = 1;
				return 1;
			}
			id<MTLCommandQueue> queue = (__bridge id<MTLCommandQueue>)c->queue;
			id<MTLBuffer> points = buffer(device, vertices, (NSUInteger)length, 0);
			if (points == nil) {
				say(err, errlen, @"could not allocate acceleration vertices");
				return 1;
			}
			NSMutableArray<id<MTLAccelerationStructure>> *built = [NSMutableArray array];
			for (int n = 0; n < count; n++) {
				int kind = nodes[n * 3], first = nodes[n * 3 + 1];
				int amount = nodes[n * 3 + 2];
				MTLAccelerationStructureDescriptor *descriptor;
				if (kind == 0) {
					MTLAccelerationStructureTriangleGeometryDescriptor *geometry =
						[MTLAccelerationStructureTriangleGeometryDescriptor descriptor];
					geometry.vertexBuffer = points;
					geometry.vertexBufferOffset = (NSUInteger)first * 3 * stride;
					geometry.vertexStride = (NSUInteger)stride;
					geometry.vertexFormat = MTLAttributeFormatFloat3;
					geometry.triangleCount = (NSUInteger)amount;
					geometry.opaque = YES;
					MTLPrimitiveAccelerationStructureDescriptor *primitive =
						[MTLPrimitiveAccelerationStructureDescriptor descriptor];
					primitive.geometryDescriptors = @[geometry];
					descriptor = primitive;
				} else {
					NSUInteger size = sizeof(MTLAccelerationStructureUserIDInstanceDescriptor);
					id<MTLBuffer> instances = [device newBufferWithLength:MAX(1, size * amount)
						options:shared];
					if (instances == nil) {
						say(err, errlen, @"could not allocate acceleration instances");
						return 1;
					}
					memset(instances.contents, 0, instances.length);
					MTLAccelerationStructureUserIDInstanceDescriptor *placed = instances.contents;
					for (int i = 0; i < amount; i++) {
						int p = first + i;
						memcpy(&placed[i].transformationMatrix, matrices + p * 12,
						       sizeof(placed[i].transformationMatrix));
						placed[i].accelerationStructureIndex = references[p * 2];
						placed[i].mask = references[p * 2 + 1];
						placed[i].userID = (uint32_t)i;
					}
					MTLInstanceAccelerationStructureDescriptor *group =
						[MTLInstanceAccelerationStructureDescriptor descriptor];
					group.instanceDescriptorType = MTLAccelerationStructureInstanceDescriptorTypeUserID;
					group.instanceDescriptorBuffer = instances;
					group.instanceCount = (NSUInteger)amount;
					group.instancedAccelerationStructures = [built copy];
					descriptor = group;
				}
				MTLAccelerationStructureSizes sizes =
					[device accelerationStructureSizesWithDescriptor:descriptor];
				id<MTLAccelerationStructure> structure =
					[device newAccelerationStructureWithSize:sizes.accelerationStructureSize];
				id<MTLBuffer> scratch = [device newBufferWithLength:MAX(1, sizes.buildScratchBufferSize)
					options:MTLResourceStorageModePrivate];
				if (structure == nil || scratch == nil) {
					say(err, errlen, @"could not allocate an acceleration structure or scratch");
					return 1;
				}
				id<MTLCommandBuffer> commands = [queue commandBuffer];
				id<MTLAccelerationStructureCommandEncoder> encoder =
					[commands accelerationStructureCommandEncoder];
				if (encoder == nil) {
					say(err, errlen, @"could not encode an acceleration build");
					return 1;
				}
				[encoder buildAccelerationStructure:structure descriptor:descriptor
				          scratchBuffer:scratch scratchBufferOffset:0];
				[encoder endEncoding];
				[commands commit];
				NSString *what = [NSString stringWithFormat:
					@"the acceleration build of node %d of %d", n + 1, count];
				NSString *why = nil;
				if (!slayer_wait_command(commands, c->limit, what, &why)) {
					c->stuck = 1;
					say(err, errlen, why);
					return 1;
				}
				if (commands.error != nil) {
					failed(err, errlen, @"acceleration build", commands.error);
					return 1;
				}
				c->gpu += commands.GPUEndTime - commands.GPUStartTime;
				[built addObject:structure];
			}
			(void)(__bridge_transfer id)c->accelerations;
			c->accelerations = (__bridge_retained void *)built;
			c->acceleration_root = root;
			return 0;
		}
		*unavailable = 1;
		return 1;
	}
}

int slayer_metal_accelerate(void *ctx, const void *vertices, long length,
                            int stride, const int32_t *nodes, int count,
                            const float *matrices, const uint32_t *references,
                            int root, int *unavailable, char *err, int errlen) {
	return accelerate(ctx, vertices, length, stride, nodes, count, matrices,
	                  references, root, 0, unavailable, err, errlen);
}

int slayer_metal_reaccelerate(void *ctx, const void *vertices, long length,
                              int stride, const int32_t *nodes, int count,
                              const float *matrices,
                              const uint32_t *references, int root,
                              int *unavailable, char *err, int errlen) {
	return accelerate(ctx, vertices, length, stride, nodes, count, matrices,
	                  references, root, 1, unavailable, err, errlen);
}
