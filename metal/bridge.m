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
	void *held;     // the meshes and groups kept between hierarchy passes
	int kept, refit, built; // the last pass's nodes by their work
	int fault;      // the fault the next hierarchy pass meets, for tests
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
	MTLCommandBufferDescriptor *made_as = [MTLCommandBufferDescriptor new];
	if (slayer_keeps_encoder_states()) {
		made_as.errorOptions =
			MTLCommandBufferErrorOptionEncoderExecutionStatus;
	}
	id<MTLCommandBuffer> commands =
		[queue commandBufferWithDescriptor:made_as];
	if (commands == nil) {
		say(err, errlen, @"could not make a command buffer");
		return nil;
	}
	id<MTLComputeCommandEncoder> encoder = [commands computeCommandEncoder];
	encoder.label = [NSString stringWithUTF8String:c->kernel];
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

// await_as waits a committed command out for at most the context's
// limit, counts its time on the gpu, and says why it failed, after the
// word for what failed, or which command it gave up on. after giving up
// the context is not used again. stopped, when given, is set to one when
// the system stopped the command for the display, which leaves the
// context usable.
static int await_as(context *c, id<MTLCommandBuffer> commands,
                    NSString *what, NSString *failing, int *stopped,
                    char *err, int errlen) {
	NSString *why = nil;
	if (!slayer_wait_command(commands, c->limit, what, &why)) {
		c->stuck = 1;
		say(err, errlen, why);
		return 1;
	}
	if (commands.error != nil) {
		// the system may stop a command itself; which one, and how long
		// it had run on the gpu, is what finds the cause
		NSString *named = [NSString stringWithFormat:
			@"%@ of %@ after %.3f s on the gpu", failing, what,
			commands.GPUEndTime - commands.GPUStartTime];
		say(err, errlen, slayer_command_failure(named, commands.error));
		if (stopped != NULL) {
			*stopped = slayer_stopped_for_display(commands.error) ? 1 : 0;
		}
		return 1;
	}
	c->gpu += commands.GPUEndTime - commands.GPUStartTime;
	return 0;
}

// await is await_as for a dispatch, which fails as a run
static int await(context *c, id<MTLCommandBuffer> commands, NSString *what,
                 char *err, int errlen) {
	return await_as(c, commands, what, @"run", NULL, err, errlen);
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

// flight is a committed band not yet waited out: its command, its rows
// and the band buffer it draws into
typedef struct {
	void *commands;
	int first, down, band;
} flight;

// grid_stream owns the retained input buffers, two band buffers the
// gpu and the cpu take turns with, and at most two committed commands
// not yet waited out, oldest first; the Go call keeps its kernel.
typedef struct {
	context *kernel;
	void *inputs_held;
	void *bands[2];
	flight flying[2];
	int flights;
	int ready_down, ready_band; // the band last waited out, to copy
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

// take_flight takes the oldest band in flight off the stream, giving
// back its command, which the caller then owns, and its rows and buffer
static id<MTLCommandBuffer> take_flight(grid_stream *stream, flight *taken) {
	*taken = stream->flying[0];
	stream->flying[0] = stream->flying[1];
	stream->flying[1] = (flight){0};
	stream->flights--;
	return (__bridge_transfer id)taken->commands;
}

// settle waits out every band in flight that the caller did not wait
// for, counting their time. it is non-zero, with err saying why, when
// one failed or timed out; the bands after it are still waited out.
static int settle(grid_stream *stream, char *err, int errlen) {
	int failures = 0;
	while (stream->flights > 0) {
		flight taken;
		id<MTLCommandBuffer> flying = take_flight(stream, &taken);
		NSString *what = band_named(stream->kernel, taken.first, taken.down);
		if (failures == 0) {
			failures = await(stream->kernel, flying, what, err, errlen);
		} else {
			char ignored[1];
			await(stream->kernel, flying, what, ignored, sizeof(ignored));
		}
	}
	return failures;
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

int slayer_metal_grid_commit(void *handle, int first, int down, char *err,
                             int errlen) {
	@autoreleasepool {
		grid_stream *stream = handle;
		if (stream == NULL || stream->inputs != stream->expected_inputs ||
		    first < 0 || down <= 0 || down > stream->rows ||
		    first + down > stream->height || stream->flights >= 2) {
			say(err, errlen, @"invalid streamed grid band");
			return 1;
		}
		// with nothing in flight, the buffer not last waited out, which
		// the caller may still be reading; with a band in flight, the
		// other one, which the caller has copied
		int band = stream->flights == 0 ? 1 - stream->ready_band
		                                : 1 - stream->flying[0].band;
		if (band == stream->ready_band) {
			stream->ready_down = 0;
		}
		id<MTLCommandBuffer> flying =
			commit_band(stream, band, first, down, err, errlen);
		if (flying == nil) {
			return 1;
		}
		stream->flying[stream->flights++] = (flight){
			(__bridge_retained void *)flying, first, down, band};
		return 0;
	}
}

int slayer_metal_grid_wait(void *handle, int *stopped, char *err,
                           int errlen) {
	@autoreleasepool {
		grid_stream *stream = handle;
		*stopped = 0;
		if (stream == NULL || stream->flights == 0) {
			say(err, errlen, @"no streamed grid band is in flight");
			return 1;
		}
		flight taken;
		id<MTLCommandBuffer> flying = take_flight(stream, &taken);
		NSString *what = band_named(stream->kernel, taken.first, taken.down);
		if (await_as(stream->kernel, flying, what, @"run", stopped, err,
		             errlen) != 0) {
			return 1;
		}
		stream->ready_band = taken.band;
		stream->ready_down = taken.down;
		return 0;
	}
}

int slayer_metal_grid_copy(void *handle, int down, float *out, char *err,
                           int errlen) {
	@autoreleasepool {
		grid_stream *stream = handle;
		if (stream == NULL || down <= 0 || down != stream->ready_down) {
			say(err, errlen, @"invalid streamed grid copy");
			return 1;
		}
		id<MTLBuffer> drawn =
			(__bridge id<MTLBuffer>)stream->bands[stream->ready_band];
		memcpy(out, drawn.contents, stream->row_bytes * (size_t)down);
		return 0;
	}
}

int slayer_metal_grid_settle(void *handle, char *err, int errlen) {
	if (handle == NULL) { return 0; }
	grid_stream *stream = handle;
	@autoreleasepool {
		if (stream->kernel != NULL && stream->kernel->stuck) {
			// nothing more is waited for: the device may never finish it
			while (stream->flights > 0) {
				flight taken;
				(void)take_flight(stream, &taken);
			}
		}
		return settle(stream, err, errlen);
	}
}

void slayer_metal_grid_close(void *handle) {
	if (handle == NULL) { return; }
	grid_stream *stream = handle;
	// under a pool, as every call from Go is: what these objects
	// autorelease as they go has nowhere else to drain
	@autoreleasepool {
		char ignored[1];
		slayer_metal_grid_settle(stream, ignored, sizeof(ignored));
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
		(void)(__bridge_transfer id)c->held;
		(void)(__bridge_transfer id)c->pipeline;
		(void)(__bridge_transfer id)c->pipelines;
		(void)(__bridge_transfer id)c->declared;
		(void)(__bridge_transfer id)c->library;
		(void)(__bridge_transfer id)c->queue;
		(void)(__bridge_transfer id)c->device;
	}
	free(c);
}

// hierarchy is a plan as Go stages it for the bridge: the positions and
// their stride, the nodes as kind, first and count triples, and each
// placement's packed matrix and its child and mask.
typedef struct {
	const void *vertices;
	long length;
	int stride;
	const int32_t *nodes;
	int count;
	const float *matrices;
	const uint32_t *references;
} hierarchy;

// the words of a node as acceleration.go stages them, and the kind that
// is a triangle range; every other kind is placements
enum {
	kind_word = 0,       // defined: the node's kind comes first
	first_word = 1,      // defined: then its first triangle or placement
	count_word = 2,      // defined: then how many
	node_words = 3,      // defined: three words a node
	triangle_kind = 0,   // defined: acceleration.go's kind for triangles
};

// the words of a placement as acceleration.go stages them
enum {
	child_word = 0,      // defined: a reference's child comes first
	mask_word = 1,       // defined: then its mask
	reference_words = 2, // defined: two words a reference
	matrix_words = 12,   // defined: a packed column-major 4 by 3 matrix
	triangle_corners = 3 // defined: a nonindexed triangle's vertices
};

// scratch_alignment is where each build's part of the one scratch buffer
// starts: a multiple of it, so no two parts overlap and each offset is
// at least as aligned as any buffer offset Metal asks for. it is chosen,
// not taken from a header, which names no alignment for scratch.
static const NSUInteger scratch_alignment = 256;

// aligned is bytes rounded up to the scratch alignment, at least one part
static NSUInteger aligned(NSUInteger bytes) {
	NSUInteger parts = (bytes + scratch_alignment - 1) / scratch_alignment;
	return MAX(1, parts) * scratch_alignment;
}

// node_work is what a pass does for a node: keeps a structure the
// context holds, refits a held structure to the node's new positions, or
// builds one.
typedef enum {
	work_keep = 0,  // defined: no work on the device
	work_refit = 1, // defined: a refit of a held structure
	work_build = 2, // defined: a build from nothing
} node_work;

// refits_before_rebuild is how many refits in a row a deforming mesh
// takes before it is built again: a refit keeps the tree's old shape,
// which serves the more poorly the further the mesh has moved. it is
// chosen, not measured.
static const int refits_before_rebuild = 30;

// passes_held is how many passes' meshes a context holds: this pass's
// and the one before's. it is gpu's keptTrees rule for the trees, so a
// film whose meshes change every frame holds two frames' structures at
// most, and never grows without bound.
static const long passes_held = 2;

// SlayerMesh is a triangle range's structure, held under a hash of its
// positions while a node of this pass or the one before placed it: the
// bytes it was built from at their stride, whether it was built to be
// refit, how many refits it has had and the last pass that placed it.
//
// a refit mesh keeps the structure it was refit from as a spare, which
// its next refit writes into where it fits.
@interface SlayerMesh : NSObject
@property(nonatomic) NSData *positions;
@property(nonatomic) int stride;
@property(nonatomic) uint64_t key;
@property(nonatomic) BOOL refittable;
@property(nonatomic) int refits;
@property(nonatomic) long placed;
@property(nonatomic) id structure;
@property(nonatomic) id spare;
@end

@implementation SlayerMesh
@end

// SlayerGroup is a node of placements as the last pass built it: its
// matrices and references, the structure each placement placed, and its
// own structure.
//
// it keeps the instance buffer it was built from and the structure the
// group before it gave up, a spare, which its next build uses again
// where they fit.
@interface SlayerGroup : NSObject
@property(nonatomic) NSData *matrices;
@property(nonatomic) NSData *references;
@property(nonatomic) NSArray *children;
@property(nonatomic) id structure;
@property(nonatomic) id spare;
@property(nonatomic) id instances;
@end

@implementation SlayerGroup
@end

// SlayerHeld is what a context keeps of its hierarchy between passes:
// the mesh structures the last passes_held passes placed, under a hash of
// their positions, how many passes have finished, and what each node of
// the last pass was, a mesh or a group.
//
// it keeps the last vertex buffer and scratch buffer too, which the next
// pass writes into again where they are the same length or larger.
@interface SlayerHeld : NSObject
@property(nonatomic) NSMutableDictionary *meshes;
@property(nonatomic) long pass;
@property(nonatomic) NSArray *nodes;
@property(nonatomic) id points;
@property(nonatomic) id scratch;
@end

@implementation SlayerHeld
@end

// SlayerBuilds is one pass over a hierarchy: for every node its level,
// its work, the mesh or group it is, the descriptor and the source of
// its work, and where in the one scratch buffer it works; the scratch
// every level needs, and the meshes the pass made, refit away or used.
@interface SlayerBuilds : NSObject
@property(nonatomic) NSMutableData *levels;
@property(nonatomic) int depth;
@property(nonatomic) NSMutableData *work;
@property(nonatomic) NSMutableData *keys;
@property(nonatomic) NSMutableArray *made;
@property(nonatomic) NSMutableArray *descriptors;
@property(nonatomic) NSMutableArray *sources;
@property(nonatomic) NSMutableData *offsets;
@property(nonatomic) NSUInteger scratch;
@property(nonatomic) NSMutableDictionary *fresh;
@property(nonatomic) NSMutableSet *used;
@property(nonatomic) NSMutableSet *consumed;
@property(nonatomic) NSMutableSet *written;
@property(nonatomic) id points;
@property(nonatomic) id scratchBuffer;
@end

@implementation SlayerBuilds
@end

// level_of gives node n its level: nought for a triangle range, and for
// placements one more than the highest of the children it places, which
// acceleration.go has checked are earlier nodes. empty placements are one.
static int level_of(const hierarchy *plan, const int *levels, int n) {
	const int32_t *node = plan->nodes + n * node_words;
	if (node[kind_word] == triangle_kind) {
		return 0;
	}
	int level = 1;
	for (int i = 0; i < node[count_word]; i++) {
		int at = (node[first_word] + i) * reference_words;
		int child = (int)plan->references[at + child_word];
		level = MAX(level, levels[child] + 1);
	}
	return level;
}

// leveled is a pass with every node's level and the count of levels set,
// so a node is built in a later level than everything it places
static SlayerBuilds *leveled(const hierarchy *plan) {
	SlayerBuilds *builds = [SlayerBuilds new];
	builds.levels = [NSMutableData dataWithLength:
		(NSUInteger)plan->count * sizeof(int)];
	builds.offsets = [NSMutableData dataWithLength:
		(NSUInteger)plan->count * sizeof(NSUInteger)];
	builds.work = [NSMutableData dataWithLength:
		(NSUInteger)plan->count * sizeof(int)];
	builds.keys = [NSMutableData dataWithLength:
		(NSUInteger)plan->count * sizeof(uint64_t)];
	builds.made = [NSMutableArray arrayWithCapacity:(NSUInteger)plan->count];
	builds.descriptors = [NSMutableArray array];
	builds.sources = [NSMutableArray array];
	for (int n = 0; n < plan->count; n++) {
		[builds.made addObject:[NSNull null]];
		[builds.descriptors addObject:[NSNull null]];
		[builds.sources addObject:[NSNull null]];
	}
	builds.fresh = [NSMutableDictionary dictionary];
	builds.used = [NSMutableSet set];
	builds.consumed = [NSMutableSet set];
	builds.written = [NSMutableSet set];
	int *levels = builds.levels.mutableBytes;
	int depth = 0;
	for (int n = 0; n < plan->count; n++) {
		levels[n] = level_of(plan, levels, n);
		depth = MAX(depth, levels[n] + 1);
	}
	builds.depth = depth;
	return builds;
}

// triangles is the descriptor of node n's triangle range in points, for
// a structure that may be refit or for one that may not: a structure
// built to be refit is shaped otherwise, so a mesh is built that way
// only once it has been seen to deform.
static MTLAccelerationStructureDescriptor *triangles(id<MTLBuffer> points,
	const hierarchy *plan, int n, BOOL refittable) API_AVAILABLE(macos(14.0)) {
	const int32_t *node = plan->nodes + n * node_words;
	MTLAccelerationStructureTriangleGeometryDescriptor *geometry =
		[MTLAccelerationStructureTriangleGeometryDescriptor descriptor];
	geometry.vertexBuffer = points;
	geometry.vertexBufferOffset = (NSUInteger)node[first_word] *
		triangle_corners * (NSUInteger)plan->stride;
	geometry.vertexStride = (NSUInteger)plan->stride;
	geometry.vertexFormat = MTLAttributeFormatFloat3;
	geometry.triangleCount = (NSUInteger)node[count_word];
	geometry.opaque = YES;
	MTLPrimitiveAccelerationStructureDescriptor *primitive =
		[MTLPrimitiveAccelerationStructureDescriptor descriptor];
	primitive.geometryDescriptors = @[geometry];
	if (refittable) {
		primitive.usage = MTLAccelerationStructureUsageRefit;
	}
	return primitive;
}

// place writes node n's placements into instance descriptors, each with
// its child's index among the earlier structures, its mask and its index
// among the node's placements as its user id
static void place(MTLAccelerationStructureUserIDInstanceDescriptor *placed,
	const hierarchy *plan, int n) API_AVAILABLE(macos(14.0)) {
	const int32_t *node = plan->nodes + n * node_words;
	for (int i = 0; i < node[count_word]; i++) {
		int p = node[first_word] + i;
		memcpy(&placed[i].transformationMatrix,
		       plan->matrices + p * matrix_words,
		       sizeof(placed[i].transformationMatrix));
		const uint32_t *reference = plan->references + p * reference_words;
		placed[i].accelerationStructureIndex = reference[child_word];
		placed[i].mask = reference[mask_word];
		placed[i].userID = (uint32_t)i;
	}
}

// placements is the descriptor of node n's placements over the earlier
// structures, written into the instance buffer kept, when it is large
// enough, and else into a new one; nil when a new one cannot be made.
// a built structure keeps no reference to its instance buffer.
static MTLAccelerationStructureDescriptor *placements(id<MTLDevice> device,
	const hierarchy *plan, int n, NSArray *earlier,
	id<MTLBuffer> kept) API_AVAILABLE(macos(14.0)) {
	NSUInteger amount = (NSUInteger)plan->nodes[n * node_words + count_word];
	NSUInteger size = sizeof(MTLAccelerationStructureUserIDInstanceDescriptor);
	id<MTLBuffer> instances = kept;
	if (instances == nil || instances.length < MAX(1, size * amount)) {
		instances = [device newBufferWithLength:MAX(1, size * amount)
		                                options:shared];
	}
	if (instances == nil) {
		return nil;
	}
	memset(instances.contents, 0, instances.length);
	place(instances.contents, plan, n);
	MTLInstanceAccelerationStructureDescriptor *group =
		[MTLInstanceAccelerationStructureDescriptor descriptor];
	group.instanceDescriptorType =
		MTLAccelerationStructureInstanceDescriptorTypeUserID;
	group.instanceDescriptorBuffer = instances;
	group.instanceCount = amount;
	group.instancedAccelerationStructures = earlier;
	return group;
}

// positions is where node n's triangles' bytes are among the plan's
// vertices, with their length
static const unsigned char *positions(const hierarchy *plan, int n,
                                      size_t *length) {
	const int32_t *node = plan->nodes + n * node_words;
	size_t corners = (size_t)triangle_corners * (size_t)plan->stride;
	*length = (size_t)node[count_word] * corners;
	return (const unsigned char *)plan->vertices +
		(size_t)node[first_word] * corners;
}

// the numbers content_hash mixes words with: FNV-1a's offset basis and
// prime, and a shift that folds a word's high bits into its low ones
static const uint64_t hash_basis = 0xcbf29ce484222325ULL; // defined: FNV-1a
static const uint64_t hash_prime = 0x100000001b3ULL;      // defined: FNV-1a
static const int hash_fold = 29; // chosen: about half a word

// content_hash is a hash of bytes, a word at a time. a match only names
// a mesh to compare; the bytes themselves decide whether it is the same.
static uint64_t content_hash(const unsigned char *bytes, size_t length) {
	uint64_t hash = hash_basis ^ (uint64_t)length;
	size_t at = 0;
	for (; at + sizeof(uint64_t) <= length; at += sizeof(uint64_t)) {
		uint64_t word;
		memcpy(&word, bytes + at, sizeof(word));
		hash = (hash ^ word) * hash_prime;
		hash ^= hash >> hash_fold;
	}
	for (; at < length; at++) {
		hash = (hash ^ bytes[at]) * hash_prime;
	}
	return hash;
}

// holds is whether a mesh was built from exactly these bytes at a stride
static BOOL holds(SlayerMesh *mesh, const unsigned char *bytes,
                  size_t length, int stride) {
	return mesh.stride == stride && mesh.positions.length == length &&
		memcmp(mesh.positions.bytes, bytes, length) == 0;
}

// found is the mesh of a bucket built from these bytes, or nil
static SlayerMesh *found(NSArray *bucket, const unsigned char *bytes,
                         size_t length, int stride) {
	for (SlayerMesh *mesh in bucket) {
		if (holds(mesh, bytes, length, stride)) {
			return mesh;
		}
	}
	return nil;
}

// last_at is what node n was in the last pass, a mesh or a group, or nil
static id last_at(SlayerHeld *held, int n) {
	if (held == nil || (NSUInteger)n >= held.nodes.count) {
		return nil;
	}
	return held.nodes[(NSUInteger)n];
}

// last_mesh is the mesh node n was in the last pass, or nil
static SlayerMesh *last_mesh(SlayerHeld *held, int n) {
	id last = last_at(held, n);
	return [last isKindOfClass:[SlayerMesh class]] ? last : nil;
}

// same_mesh is a held mesh built from node n's bytes: the one the node
// was last pass, compared first, or any under the bytes' hash, which it
// sets in key when it works it out; nil when none is
static SlayerMesh *same_mesh(SlayerHeld *held, const hierarchy *plan, int n,
                             uint64_t *key) {
	size_t length;
	const unsigned char *bytes = positions(plan, n, &length);
	SlayerMesh *last = last_mesh(held, n);
	if (last != nil && holds(last, bytes, length, plan->stride)) {
		*key = last.key;
		return last;
	}
	*key = content_hash(bytes, length);
	return found(held.meshes[@(*key)], bytes, length, plan->stride);
}

// is_triangles is whether node n is a triangle range
static BOOL is_triangles(const hierarchy *plan, int n) {
	return plan->nodes[n * node_words + kind_word] == triangle_kind;
}

// keep_meshes is the first look at the triangle ranges: each whose bytes
// a held mesh was built from keeps that mesh, and every other is to be
// built until plan_meshes says otherwise
static void keep_meshes(SlayerHeld *held, SlayerBuilds *builds,
                        const hierarchy *plan) {
	uint64_t *keys = builds.keys.mutableBytes;
	int *work = builds.work.mutableBytes;
	for (int n = 0; n < plan->count; n++) {
		if (!is_triangles(plan, n)) {
			continue;
		}
		SlayerMesh *mesh = same_mesh(held, plan, n, &keys[n]);
		work[n] = mesh != nil ? work_keep : work_build;
		if (mesh != nil) {
			builds.made[(NSUInteger)n] = mesh;
			[builds.used addObject:mesh];
		}
	}
}

// refit_source is the mesh node n was last pass, when a refit may carry
// it to the node's new positions: as many triangles at the same stride,
// built to be refit, refit fewer times than the limit, and neither kept
// by a node of this pass nor refit already into another; else nil
static SlayerMesh *refit_source(SlayerHeld *held, SlayerBuilds *builds,
                                const hierarchy *plan, int n, size_t length) {
	SlayerMesh *mesh = last_mesh(held, n);
	if (mesh == nil || mesh.stride != plan->stride ||
	    mesh.positions.length != length || !mesh.refittable ||
	    mesh.refits >= refits_before_rebuild) {
		return nil;
	}
	if ([builds.used containsObject:mesh] ||
	    [builds.consumed containsObject:mesh]) {
		return nil;
	}
	return mesh;
}

// deformed is whether node n was last pass a mesh of as many triangles
// at the same stride, which has moved: a mesh to build to be refit
static BOOL deformed(SlayerHeld *held, const hierarchy *plan, int n,
                     size_t length) {
	SlayerMesh *mesh = last_mesh(held, n);
	return mesh != nil && mesh.stride == plan->stride &&
		mesh.positions.length == length;
}

// new_mesh is the mesh node n's positions make this pass, refit from a
// source or built, entered among the pass's fresh meshes under its hash
static SlayerMesh *new_mesh(SlayerBuilds *builds, const hierarchy *plan,
                            int n, SlayerMesh *source, BOOL refittable) {
	size_t length;
	const unsigned char *bytes = positions(plan, n, &length);
	SlayerMesh *mesh = [SlayerMesh new];
	mesh.positions = [NSData dataWithBytes:bytes length:length];
	mesh.stride = plan->stride;
	mesh.key = ((const uint64_t *)builds.keys.bytes)[n];
	mesh.refittable = refittable;
	mesh.refits = source != nil ? source.refits + 1 : 0;
	NSMutableArray *bucket = builds.fresh[@(mesh.key)];
	if (bucket == nil) {
		bucket = [NSMutableArray array];
		builds.fresh[@(mesh.key)] = bucket;
	}
	[bucket addObject:mesh];
	return mesh;
}

// plan_meshes decides the work of each triangle range no held mesh
// serves: one whose bytes another range of this pass makes shares that
// mesh, one that deformed is refit where it may be, and the rest build
static void plan_meshes(SlayerHeld *held, SlayerBuilds *builds,
                        const hierarchy *plan) {
	int *work = builds.work.mutableBytes;
	const uint64_t *keys = builds.keys.bytes;
	for (int n = 0; n < plan->count; n++) {
		if (!is_triangles(plan, n) || work[n] == work_keep) {
			continue;
		}
		size_t length;
		const unsigned char *bytes = positions(plan, n, &length);
		SlayerMesh *twin = found(builds.fresh[@(keys[n])], bytes, length,
		                         plan->stride);
		if (twin != nil) {
			work[n] = work_keep;
			builds.made[(NSUInteger)n] = twin;
			continue;
		}
		SlayerMesh *source = refit_source(held, builds, plan, n, length);
		if (source != nil) {
			work[n] = work_refit;
			builds.sources[(NSUInteger)n] = source;
			[builds.consumed addObject:source];
		}
		BOOL refittable = source != nil || deformed(held, plan, n, length);
		builds.made[(NSUInteger)n] =
			new_mesh(builds, plan, n, source, refittable);
	}
}

// take_scratch gives node n its part of the one scratch buffer: the
// works of a level may run together, so each has its own part, and a
// level starts only when the one before it ends, so it uses them again
static void take_scratch(SlayerBuilds *builds, NSMutableData *ends, int n,
                         NSUInteger bytes) {
	NSUInteger *end = ends.mutableBytes;
	NSUInteger *offsets = builds.offsets.mutableBytes;
	int level = ((const int *)builds.levels.bytes)[n];
	offsets[n] = end[level];
	end[level] += aligned(bytes);
	builds.scratch = MAX(builds.scratch, end[level]);
}

// reused is a spare structure when it holds size bytes, else a new one:
// a spare is one a node gave up the pass before last, which nothing
// bound or kept refers to any longer.
//
// that is safe only while frames do not overlap: pass k+1 writes into a
// spare pass k gave up, which frame k-1's hierarchy placed. a frame k-1
// still tracing then would read a structure being written.
static id<MTLAccelerationStructure> reused(id<MTLDevice> device, id spare,
	NSUInteger size) API_AVAILABLE(macos(14.0)) {
	id<MTLAccelerationStructure> held = spare;
	if (held != nil && held.size >= size) {
		return held;
	}
	return [device newAccelerationStructureWithSize:size];
}

// describe_mesh describes node n's refit or build over points and makes
// the structure it writes: for a refit, as large as its source at least,
// into the source's spare where it fits, and the source is the next spare
static int describe_mesh(id<MTLDevice> device, id<MTLBuffer> points,
	const hierarchy *plan, SlayerBuilds *builds, NSMutableData *ends, int n,
	char *err, int errlen) API_AVAILABLE(macos(14.0)) {
	SlayerMesh *mesh = builds.made[(NSUInteger)n];
	MTLAccelerationStructureDescriptor *descriptor =
		triangles(points, plan, n, mesh.refittable);
	MTLAccelerationStructureSizes sizes =
		[device accelerationStructureSizesWithDescriptor:descriptor];
	NSUInteger size = sizes.accelerationStructureSize, scratch =
		sizes.buildScratchBufferSize;
	id source = builds.sources[(NSUInteger)n], spare = nil;
	if (source != [NSNull null]) {
		id<MTLAccelerationStructure> from = [source structure];
		size = MAX(size, from.size);
		scratch = sizes.refitScratchBufferSize;
		spare = [source spare];
		mesh.spare = from;
	}
	mesh.structure = reused(device, spare, size);
	if (mesh.structure == nil) {
		say(err, errlen, @"could not allocate an acceleration structure or scratch");
		return 1;
	}
	builds.descriptors[(NSUInteger)n] = descriptor;
	[builds.written addObject:mesh.structure];
	take_scratch(builds, ends, n, scratch);
	return 0;
}

// group_same is whether node n's placements are, to the byte, the ones
// a group was built from
static BOOL group_same(SlayerGroup *group, const hierarchy *plan, int n) {
	const int32_t *node = plan->nodes + n * node_words;
	size_t first = (size_t)node[first_word], amount = (size_t)node[count_word];
	size_t matrices = amount * matrix_words * sizeof(float);
	size_t references = amount * reference_words * sizeof(uint32_t);
	return group.matrices.length == matrices &&
		group.references.length == references &&
		memcmp(group.matrices.bytes, plan->matrices + first * matrix_words,
		       matrices) == 0 &&
		memcmp(group.references.bytes,
		       plan->references + first * reference_words, references) == 0;
}

// group_kept is the group node n was last pass, when its structure
// serves as it is: the same placements, of children that are the same
// structures and that nothing this pass writes; else nil
static SlayerGroup *group_kept(SlayerHeld *held, SlayerBuilds *builds,
                               const hierarchy *plan, int n) {
	id last = last_at(held, n);
	if (![last isKindOfClass:[SlayerGroup class]] ||
	    !group_same(last, plan, n)) {
		return nil;
	}
	SlayerGroup *group = last;
	const int32_t *node = plan->nodes + n * node_words;
	for (int i = 0; i < node[count_word]; i++) {
		int at = (node[first_word] + i) * reference_words;
		id child = [builds.made[plan->references[at + child_word]] structure];
		if (child != group.children[(NSUInteger)i] ||
		    [builds.written containsObject:child]) {
			return nil;
		}
	}
	return group;
}

// new_group is node n as a group to build: copies of its placements and
// the structure each places
static SlayerGroup *new_group(SlayerBuilds *builds, const hierarchy *plan,
                              int n) {
	const int32_t *node = plan->nodes + n * node_words;
	size_t first = (size_t)node[first_word], amount = (size_t)node[count_word];
	SlayerGroup *group = [SlayerGroup new];
	group.matrices = [NSData dataWithBytes:plan->matrices + first * matrix_words
		length:amount * matrix_words * sizeof(float)];
	group.references = [NSData
		dataWithBytes:plan->references + first * reference_words
		       length:amount * reference_words * sizeof(uint32_t)];
	NSMutableArray *children = [NSMutableArray array];
	for (size_t i = 0; i < amount; i++) {
		uint32_t child =
			plan->references[(first + i) * reference_words + child_word];
		[children addObject:[builds.made[child] structure]];
	}
	group.children = children;
	return group;
}

// earlier is the structures of the nodes before n, which its placements
// name by index
static NSArray *earlier(SlayerBuilds *builds, int n) {
	NSMutableArray *structures = [NSMutableArray arrayWithCapacity:(NSUInteger)n];
	for (int k = 0; k < n; k++) {
		[structures addObject:[builds.made[(NSUInteger)k] structure]];
	}
	return structures;
}

// describe_group keeps node n's group where group_kept says it serves,
// and else describes its build and makes the structure it builds into
static int describe_group(id<MTLDevice> device, const hierarchy *plan,
	SlayerHeld *held, SlayerBuilds *builds, NSMutableData *ends, int n,
	char *err, int errlen) API_AVAILABLE(macos(14.0)) {
	int *work = builds.work.mutableBytes;
	SlayerGroup *group = group_kept(held, builds, plan, n);
	work[n] = group != nil ? work_keep : work_build;
	if (group != nil) {
		builds.made[(NSUInteger)n] = group;
		return 0;
	}
	id last = last_at(held, n);
	SlayerGroup *before = [last isKindOfClass:[SlayerGroup class]] ? last : nil;
	MTLInstanceAccelerationStructureDescriptor *descriptor = (id)placements(
		device, plan, n, earlier(builds, n), before.instances);
	if (descriptor == nil) {
		say(err, errlen, @"could not allocate acceleration instances");
		return 1;
	}
	MTLAccelerationStructureSizes sizes =
		[device accelerationStructureSizesWithDescriptor:descriptor];
	group = new_group(builds, plan, n);
	group.instances = descriptor.instanceDescriptorBuffer;
	group.spare = before.structure;
	group.structure =
		reused(device, before.spare, sizes.accelerationStructureSize);
	if (group.structure == nil) {
		say(err, errlen, @"could not allocate an acceleration structure or scratch");
		return 1;
	}
	builds.made[(NSUInteger)n] = group;
	builds.descriptors[(NSUInteger)n] = descriptor;
	[builds.written addObject:group.structure];
	take_scratch(builds, ends, n, sizes.buildScratchBufferSize);
	return 0;
}

// needs_points is whether any triangle range is refit or built this pass
static BOOL needs_points(SlayerBuilds *builds, const hierarchy *plan) {
	const int *work = builds.work.bytes;
	for (int n = 0; n < plan->count; n++) {
		if (is_triangles(plan, n) && work[n] != work_keep) {
			return YES;
		}
	}
	return NO;
}

// copied_points is the plan's vertices in a buffer the device reads: the
// one the context kept from the last pass, when it is the same length,
// and else a new one. nothing reads it between passes: a built or refit
// structure keeps no reference to the vertices it was made from.
static id<MTLBuffer> copied_points(id<MTLDevice> device, SlayerHeld *held,
                                   const hierarchy *plan) {
	id<MTLBuffer> kept = held.points;
	if (kept == nil || plan->length <= 0 ||
	    kept.length != (NSUInteger)plan->length) {
		return buffer(device, plan->vertices, (NSUInteger)plan->length, 0);
	}
	memcpy(kept.contents, plan->vertices, (size_t)plan->length);
	return kept;
}

// faulted is whether a pass meets the fault a test set at this point,
// which it takes, so the pass after runs as it is; it says why
static int faulted(context *c, int fault, char *err, int errlen) {
	if (c->fault != fault) {
		return 0;
	}
	c->fault = acceleration_fault_none;
	say(err, errlen, @"an acceleration fault a test set");
	return 1;
}

// prepare decides every node's work, describes what is refit or built,
// in node order so a group sees its children's structures, and makes
// the structures they write; positions are copied only when needed
static int prepare(context *c, const hierarchy *plan, SlayerBuilds *builds,
	char *err, int errlen) API_AVAILABLE(macos(14.0)) {
	id<MTLDevice> device = (__bridge id<MTLDevice>)c->device;
	SlayerHeld *held = (__bridge SlayerHeld *)c->held;
	keep_meshes(held, builds, plan);
	plan_meshes(held, builds, plan);
	id<MTLBuffer> points = nil;
	if (needs_points(builds, plan)) {
		points = copied_points(device, held, plan);
		if (points == nil) {
			say(err, errlen, @"could not allocate acceleration vertices");
			return 1;
		}
		builds.points = points;
	}
	NSMutableData *ends = [NSMutableData dataWithLength:
		(NSUInteger)builds.depth * sizeof(NSUInteger)];
	const int *work = builds.work.bytes;
	for (int n = 0; n < plan->count; n++) {
		int failed = is_triangles(plan, n) ?
			(work[n] == work_keep ? 0 : describe_mesh(device, points, plan,
				builds, ends, n, err, errlen)) :
			describe_group(device, plan, held, builds, ends, n, err, errlen);
		if (failed != 0) {
			return 1;
		}
	}
	return faulted(c, acceleration_fault_allocation, err, errlen);
}

// encode_level encodes the builds of one level into an encoder of their
// own, after the fence the level before updated, if there was one, and
// updates the next fence, if there is one, once they are done
static int encode_level(id<MTLCommandBuffer> commands, SlayerBuilds *builds,
	int level, id<MTLBuffer> scratch, id<MTLFence> before, id<MTLFence> after,
	char *err, int errlen) API_AVAILABLE(macos(14.0)) {
	id<MTLAccelerationStructureCommandEncoder> encoder =
		[commands accelerationStructureCommandEncoder];
	if (encoder == nil) {
		say(err, errlen, @"could not encode an acceleration build");
		return 1;
	}
	if (before != nil) {
		[encoder waitForFence:before];
	}
	const int *levels = builds.levels.bytes, *work = builds.work.bytes;
	const NSUInteger *offsets = builds.offsets.bytes;
	for (NSUInteger n = 0; n < builds.made.count; n++) {
		if (levels[n] != level || work[n] == work_keep) {
			continue;
		}
		id<MTLAccelerationStructure> into = [builds.made[n] structure];
		if (work[n] == work_refit) {
			[encoder refitAccelerationStructure:[builds.sources[n] structure]
			                         descriptor:builds.descriptors[n]
			                        destination:into
			                      scratchBuffer:scratch
			                scratchBufferOffset:offsets[n]];
			continue;
		}
		[encoder buildAccelerationStructure:into
		                         descriptor:builds.descriptors[n]
		                      scratchBuffer:scratch
		                scratchBufferOffset:offsets[n]];
	}
	if (after != nil) {
		[encoder updateFence:after];
	}
	[encoder endEncoding];
	return 0;
}

// work_from is whether a node of a level from low to high, both
// included, is refit or built
static BOOL work_from(SlayerBuilds *builds, int low, int high) {
	const int *levels = builds.levels.bytes, *work = builds.work.bytes;
	for (NSUInteger n = 0; n < builds.made.count; n++) {
		if (levels[n] >= low && levels[n] <= high && work[n] != work_keep) {
			return YES;
		}
	}
	return NO;
}

// encode_builds encodes every level's work into one command buffer, a
// level with work to an encoder, each waiting on a fence the one before
// it updates. Metal's header for the encoder says every bottom level
// build must end before a top level build may begin, and leaves that
// order to the caller. a level with no work has no encoder: a command
// whose first encoders were empty gave back no start time on the device.
static int encode_builds(id<MTLDevice> device, id<MTLCommandBuffer> commands,
	SlayerBuilds *builds, id<MTLBuffer> scratch, char *err,
	int errlen) API_AVAILABLE(macos(14.0)) {
	id<MTLFence> before = nil;
	for (int level = 0; level < builds.depth; level++) {
		if (!work_from(builds, level, level)) {
			continue;
		}
		id<MTLFence> after = nil;
		if (work_from(builds, level + 1, builds.depth)) {
			after = [device newFence];
			if (after == nil) {
				say(err, errlen, @"could not make an acceleration fence");
				return 1;
			}
		}
		if (encode_level(commands, builds, level, scratch, before, after,
		                 err, errlen) != 0) {
			return 1;
		}
		before = after;
	}
	return 0;
}

// counted is how many nodes of a pass have each work
static void counted(context *c, SlayerBuilds *builds) {
	const int *work = builds.work.bytes;
	c->kept = c->refit = c->built = 0;
	for (NSUInteger n = 0; n < builds.made.count; n++) {
		c->kept += work[n] == work_keep;
		c->refit += work[n] == work_refit;
		c->built += work[n] == work_build;
	}
}

// held_by is what the context holds, made empty the first time
static SlayerHeld *held_by(context *c) {
	if (c->held == NULL) {
		SlayerHeld *made = [SlayerHeld new];
		made.meshes = [NSMutableDictionary dictionary];
		c->held = (__bridge_retained void *)made;
	}
	return (__bridge SlayerHeld *)c->held;
}

// released is whether a mesh no node of the last passes_held passes
// placed, as of a held count of passes
static BOOL released(SlayerMesh *mesh, long pass) {
	return pass - mesh.placed >= passes_held;
}

// let_go takes out of the held meshes every one the test names, its
// structure, spare and positions with it, and every bucket left empty
static void let_go(SlayerHeld *held, BOOL (^gone)(SlayerMesh *mesh)) {
	for (NSNumber *key in [held.meshes allKeys]) {
		NSMutableArray *bucket = held.meshes[key];
		NSIndexSet *going = [bucket indexesOfObjectsPassingTest:
			^BOOL(SlayerMesh *mesh, NSUInteger at, BOOL *stop) {
				(void)at;
				(void)stop;
				return gone(mesh);
			}];
		[bucket removeObjectsAtIndexes:going];
		if (bucket.count == 0) {
			[held.meshes removeObjectForKey:key];
		}
	}
}

// hold_meshes counts a finished pass and keeps its meshes: those refit
// into others go, those it made join the rest under their hashes, every
// mesh it placed is marked placed now, and those no node placed this
// pass or the one before go, so what is held never grows without bound
static void hold_meshes(SlayerHeld *held, SlayerBuilds *builds) {
	long pass = ++held.pass;
	NSSet *consumed = builds.consumed;
	for (NSNumber *key in builds.fresh) {
		NSMutableArray *bucket = held.meshes[key];
		if (bucket == nil) {
			held.meshes[key] = builds.fresh[key];
			continue;
		}
		[bucket addObjectsFromArray:builds.fresh[key]];
	}
	for (id node in builds.made) {
		if ([node isKindOfClass:[SlayerMesh class]]) {
			((SlayerMesh *)node).placed = pass;
		}
	}
	let_go(held, ^BOOL(SlayerMesh *mesh) {
		return [consumed containsObject:mesh] || released(mesh, pass);
	});
}

// hold makes a finished pass what the context holds: its meshes, as
// hold_meshes keeps them, and its nodes' structures as the hierarchy,
// with the root to bind
static void hold(context *c, SlayerBuilds *builds, int root) {
	SlayerHeld *held = held_by(c);
	hold_meshes(held, builds);
	held.nodes = [builds.made copy];
	if (builds.points != nil) {
		held.points = builds.points;
	}
	if (builds.scratchBuffer != nil) {
		held.scratch = builds.scratchBuffer;
	}
	(void)(__bridge_transfer id)c->accelerations;
	c->accelerations = (__bridge_retained void *)earlier(builds,
		(int)builds.made.count);
	c->acceleration_root = root;
	counted(c, builds);
}

// any_work is whether a pass refits or builds anything at all
static BOOL any_work(SlayerBuilds *builds) {
	const int *work = builds.work.bytes;
	for (NSUInteger n = 0; n < builds.made.count; n++) {
		if (work[n] != work_keep) {
			return YES;
		}
	}
	return NO;
}

// run_builds encodes a pass's work into one command, commits it and
// waits for it once
static int run_builds(context *c, const hierarchy *plan,
	SlayerBuilds *builds, char *err, int errlen) API_AVAILABLE(macos(14.0)) {
	id<MTLDevice> device = (__bridge id<MTLDevice>)c->device;
	SlayerHeld *held = (__bridge SlayerHeld *)c->held;
	id<MTLBuffer> scratch = held.scratch;
	if (scratch == nil || scratch.length < MAX(1, builds.scratch)) {
		scratch = [device newBufferWithLength:MAX(1, builds.scratch)
		                              options:MTLResourceStorageModePrivate];
	}
	builds.scratchBuffer = scratch;
	id<MTLCommandQueue> queue = (__bridge id<MTLCommandQueue>)c->queue;
	id<MTLCommandBuffer> commands = [queue commandBuffer];
	if (scratch == nil || commands == nil) {
		say(err, errlen, @"could not allocate an acceleration structure or scratch");
		return 1;
	}
	if (encode_builds(device, commands, builds, scratch, err, errlen) != 0) {
		return 1;
	}
	[commands commit];
	NSString *what = [NSString stringWithFormat:
		@"the acceleration build of %d nodes in %d levels", plan->count,
		builds.depth];
	if (await_as(c, commands, what, @"acceleration build", NULL, err,
	             errlen) != 0) {
		return 1;
	}
	return faulted(c, acceleration_fault_command, err, errlen);
}

// build_hierarchy keeps every node whose structure the context holds,
// refits or builds the rest in one command, waited for once, and gives
// the structures to the context with the root to bind. nothing the
// context holds changes until the command has finished well.
static int build_hierarchy(context *c, const hierarchy *plan, int root,
	char *err, int errlen) API_AVAILABLE(macos(14.0)) {
	SlayerBuilds *builds = leveled(plan);
	if (prepare(c, plan, builds, err, errlen) != 0) {
		return 1;
	}
	if (any_work(builds) && run_builds(c, plan, builds, err, errlen) != 0) {
		return 1;
	}
	hold(c, builds, root);
	return 0;
}

// accelerate builds a hierarchy and gives it to the context. With replace
// it takes the place of one already there, which stays until the new one
// is built; without, a context that has one is refused.
static int accelerate(void *ctx, const hierarchy *plan, int root,
                      int replace, int *unavailable, char *err,
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
			return build_hierarchy(c, plan, root, err, errlen);
		}
		*unavailable = 1;
		return 1;
	}
}

int slayer_metal_accelerate(void *ctx, const void *vertices, long length,
                            int stride, const int32_t *nodes, int count,
                            const float *matrices, const uint32_t *references,
                            int root, int *unavailable, char *err, int errlen) {
	hierarchy plan = {vertices, length, stride, nodes, count, matrices,
	                  references};
	return accelerate(ctx, &plan, root, 0, unavailable, err, errlen);
}

void slayer_metal_acceleration_work(void *ctx, int *kept, int *refit,
                                    int *built) {
	context *c = ctx;
	*kept = c->kept;
	*refit = c->refit;
	*built = c->built;
}

int slayer_metal_acceleration_held(void *ctx) {
	context *c = ctx;
	SlayerHeld *held = (__bridge SlayerHeld *)c->held;
	int count = 0;
	for (NSNumber *key in held.meshes) {
		for (SlayerMesh *mesh in held.meshes[key]) {
			count += 1 + (mesh.spare != nil);
		}
	}
	return count;
}

void slayer_metal_acceleration_fault(void *ctx, int fault) {
	context *c = ctx;
	c->fault = fault;
}

int slayer_metal_reaccelerate(void *ctx, const void *vertices, long length,
                              int stride, const int32_t *nodes, int count,
                              const float *matrices,
                              const uint32_t *references, int root,
                              int *unavailable, char *err, int errlen) {
	hierarchy plan = {vertices, length, stride, nodes, count, matrices,
	                  references};
	return accelerate(ctx, &plan, root, 1, unavailable, err, errlen);
}
