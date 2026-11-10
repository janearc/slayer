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
	void *library;
	void *pipelines;
	void *accelerations;
	int acceleration_root;
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
                                            char *err, int errlen,
                                            id<MTLLibrary> __autoreleasing *kept) {
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
	if (made != nil) { *kept = library; }
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
		id<MTLComputePipelineState> made =
			pipeline(device, source, kernel, err, errlen, &compiled);
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
		strncpy(c->name, [device.name UTF8String], sizeof(c->name) - 1);
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
   made = [device newComputePipelineStateWithFunction:function error:&why];
   if (made == nil) { failed(err, errlen, @"pipeline", why); return 1; }
   pipelines[name] = made;
  }
  (void)(__bridge_transfer id)c->pipeline;
  c->pipeline = (__bridge_retained void *)made;
  return 0;
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
	if (c->accelerations != NULL && origin == NULL) {
		say(err, errlen, @"an accelerated kernel requires Grid");
		return 1;
	}
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

// grid_stream owns only the retained buffers; the Go call keeps its kernel.
typedef struct {
	context *kernel;
	void *buffers;
	int width, rows;
	int expected_inputs, inputs;
	size_t row_bytes;
} grid_stream;

void *slayer_metal_grid_open(void *ctx, int width, int rows, int per,
                            const void **inputs, const long *lengths, int count,
                            char *err, int errlen) {
	void *stream = slayer_metal_grid_begin(ctx, width, rows, per, count, err, errlen);
	if (stream == NULL) { return NULL; }
	for (int index = 0; index < count; index++) {
		if (slayer_metal_grid_add(stream, inputs[index], lengths[index], err, errlen) != 0) {
			slayer_metal_grid_close(stream);
			return NULL;
		}
	}
	return stream;
}

void *slayer_metal_grid_begin(void *ctx, int width, int rows, int per,
                             int count, char *err, int errlen) {
	@autoreleasepool {
		if (ctx == NULL || width <= 0 || rows <= 0 || per <= 0 || count < 0) {
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
		NSMutableArray<id<MTLBuffer>> *buffers =
			[NSMutableArray array];
		size_t row_bytes = (size_t)width * (size_t)per * sizeof(float);
		id<MTLBuffer> band = [device newBufferWithLength:row_bytes * (size_t)rows
		                                        options:shared];
		if (band == nil) {
			say(err, errlen, @"could not make a streamed output buffer");
			return NULL;
		}
		[buffers addObject:band];
		grid_stream *stream = calloc(1, sizeof(grid_stream));
		if (stream == NULL) {
			say(err, errlen, @"could not allocate a grid stream");
			return NULL;
		}
		stream->kernel = c;
		stream->buffers = (__bridge_retained void *)buffers;
		stream->width = width;
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
		id<MTLBuffer> one = buffer(device, input, (NSUInteger)length);
		if (one == nil) {
			say(err, errlen, @"could not make a streamed input buffer");
			return 1;
		}
		NSMutableArray<id<MTLBuffer>> *buffers =
			(__bridge NSMutableArray<id<MTLBuffer>> *)stream->buffers;
		// Keep the output last, after exactly the caller's ordered inputs.
		[buffers insertObject:one atIndex:(NSUInteger)stream->inputs];
		stream->inputs++;
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
		NSArray<id<MTLBuffer>> *buffers =
			(__bridge NSArray<id<MTLBuffer>> *)stream->buffers;
		return draw_band(stream->kernel, buffers, stream->width, first,
		                 down, (char *)out, stream->row_bytes, err, errlen);
	}
}

void slayer_metal_grid_close(void *handle) {
	if (handle == NULL) { return; }
	grid_stream *stream = handle;
	(void)(__bridge_transfer id)stream->buffers;
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
	(void)(__bridge_transfer id)c->accelerations;
	(void)(__bridge_transfer id)c->pipeline;
	(void)(__bridge_transfer id)c->pipelines;
	(void)(__bridge_transfer id)c->library;
	(void)(__bridge_transfer id)c->queue;
	(void)(__bridge_transfer id)c->device;
	free(c);
}

// Each node is a triangle range or placements of earlier nodes. Building in
// that order keeps dependencies explicit. Resources stay owned by the kernel.
int slayer_metal_accelerate(void *ctx, const void *vertices, long length,
                            int stride, const int32_t *nodes, int count,
                            const float *matrices, const uint32_t *references,
                            int root, int *unavailable, char *err, int errlen) {
	@autoreleasepool {
		*unavailable = 0;
		context *c = ctx;
		if (c->accelerations != NULL) {
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
			id<MTLBuffer> points = buffer(device, vertices, (NSUInteger)length);
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
				[commands waitUntilCompleted];
				if (commands.error != nil) {
					failed(err, errlen, @"acceleration build", commands.error);
					return 1;
				}
				c->gpu += commands.GPUEndTime - commands.GPUStartTime;
				[built addObject:structure];
			}
			c->accelerations = (__bridge_retained void *)built;
			c->acceleration_root = root;
			return 0;
		}
		*unavailable = 1;
		return 1;
	}
}
