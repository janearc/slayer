// a session on the gpu: one library of kernels, buffers that outlive a
// dispatch, and a batch of dispatches encoded into one command, so work
// that steps many times, as a fluid does, stays on the device between
// its steps. built with ARC; the session holds its objects through
// retained bridges, its pipelines and buffers in arrays, and gives them
// back on close.

#import <Foundation/Foundation.h>
#import <Metal/Metal.h>
#include <string.h>
#include "bridge.h"

typedef struct {
	void *device;
	void *queue;
	void *pipelines;
	void *buffers;
	void *commands;
	void *encoder;
	double gpu;
} session;

// tell writes why something failed into err, cut to fit
static void tell(char *err, int errlen, NSString *what) {
	if (err == NULL || errlen <= 0) {
		return;
	}
	strncpy(err, [what UTF8String], errlen - 1);
	err[errlen - 1] = 0;
}

// library compiles the source on a device, or says why not
static id<MTLLibrary> library(id<MTLDevice> device, const char *source,
                              char *err, int errlen) {
	NSError *why = nil;
	NSString *text = [NSString stringWithUTF8String:source];
	id<MTLLibrary> made = [device newLibraryWithSource:text
	                                           options:[MTLCompileOptions new]
	                                             error:&why];
	if (made == nil) {
		tell(err, errlen, [NSString stringWithFormat:@"compile: %@",
		                   why.localizedDescription]);
	}
	return made;
}

// pipelines makes a pipeline for each named kernel in a library
static NSMutableArray *pipelines(id<MTLDevice> device, id<MTLLibrary> made,
                                 const char **kernels, int count, char *err,
                                 int errlen) {
	NSMutableArray *all = [NSMutableArray array];
	for (int k = 0; k < count; k++) {
		NSString *name = [NSString stringWithUTF8String:kernels[k]];
		id<MTLFunction> function = [made newFunctionWithName:name];
		if (function == nil) {
			tell(err, errlen, [NSString stringWithFormat:
			                   @"no kernel named %@", name]);
			return nil;
		}
		NSError *why = nil;
		id<MTLComputePipelineState> one =
			[device newComputePipelineStateWithFunction:function
			                                      error:&why];
		if (one == nil) {
			tell(err, errlen, [NSString stringWithFormat:@"pipeline %@: %@",
			                   name, why.localizedDescription]);
			return nil;
		}
		[all addObject:one];
	}
	return all;
}

void *slayer_session_open(const char *source, const char **kernels,
                          int count, int *nodevice, char *err, int errlen) {
	@autoreleasepool {
		*nodevice = 0;
		id<MTLDevice> device = MTLCreateSystemDefaultDevice();
		if (device == nil) {
			*nodevice = 1;
			tell(err, errlen, @"no metal device");
			return NULL;
		}
		id<MTLLibrary> made = library(device, source, err, errlen);
		if (made == nil) {
			return NULL;
		}
		NSMutableArray *all = pipelines(device, made, kernels, count, err,
		                                errlen);
		if (all == nil) {
			return NULL;
		}
		session *s = calloc(1, sizeof(session));
		if (s == NULL) {
			tell(err, errlen, @"out of memory for the session");
			return NULL;
		}
		s->device = (__bridge_retained void *)device;
		s->queue = (__bridge_retained void *)[device newCommandQueue];
		s->pipelines = (__bridge_retained void *)all;
		s->buffers = (__bridge_retained void *)[NSMutableArray array];
		return s;
	}
}

int slayer_session_buffer(void *ctx, long length, char *err, int errlen) {
	@autoreleasepool {
		session *s = ctx;
		id<MTLDevice> device = (__bridge id<MTLDevice>)s->device;
		NSMutableArray *buffers = (__bridge NSMutableArray *)s->buffers;
		id<MTLBuffer> made = [device
			newBufferWithLength:(NSUInteger)(length > 0 ? length : 1)
			            options:MTLResourceStorageModeShared];
		if (made == nil) {
			tell(err, errlen, @"could not make a buffer");
			return -1;
		}
		memset(made.contents, 0, made.length);
		[buffers addObject:made];
		return (int)buffers.count - 1;
	}
}

void *slayer_session_contents(void *ctx, int buffer) {
	session *s = ctx;
	NSMutableArray *buffers = (__bridge NSMutableArray *)s->buffers;
	return ((id<MTLBuffer>)buffers[buffer]).contents;
}

// encoding is the batch's encoder, begun if there is none
static id<MTLComputeCommandEncoder> encoding(session *s) {
	if (s->encoder == NULL) {
		id<MTLCommandQueue> queue = (__bridge id<MTLCommandQueue>)s->queue;
		id<MTLCommandBuffer> commands = [queue commandBuffer];
		s->commands = (__bridge_retained void *)commands;
		s->encoder =
			(__bridge_retained void *)[commands computeCommandEncoder];
	}
	return (__bridge id<MTLComputeCommandEncoder>)s->encoder;
}

int slayer_session_dispatch(void *ctx, int kernel, const int *buffers,
                            int count, const void *params, int paramlen,
                            int width, int height, int grouped, char *err,
                            int errlen) {
	@autoreleasepool {
		session *s = ctx;
		NSArray *all = (__bridge NSArray *)s->pipelines;
		NSArray *held = (__bridge NSArray *)s->buffers;
		if (kernel < 0 || kernel >= (int)all.count) {
			tell(err, errlen, @"no such kernel");
			return 1;
		}
		id<MTLComputePipelineState> made = all[kernel];
		id<MTLComputeCommandEncoder> encoder = encoding(s);
		[encoder setComputePipelineState:made];
		for (int b = 0; b < count; b++) {
			[encoder setBuffer:held[buffers[b]] offset:0 atIndex:b];
		}
		if (params != NULL && paramlen > 0) {
			[encoder setBytes:params length:(NSUInteger)paramlen
			          atIndex:(NSUInteger)count];
		}
		if (grouped) {
			[encoder dispatchThreadgroups:MTLSizeMake(height, 1, 1)
			        threadsPerThreadgroup:MTLSizeMake(width, 1, 1)];
			return 0;
		}
		NSUInteger across = made.threadExecutionWidth;
		NSUInteger down = made.maxTotalThreadsPerThreadgroup / across;
		if (height == 1) {
			across = made.maxTotalThreadsPerThreadgroup;
			down = 1;
		}
		[encoder dispatchThreads:MTLSizeMake(width, height, 1)
		   threadsPerThreadgroup:MTLSizeMake(across, down, 1)];
		return 0;
	}
}

int slayer_session_commit(void *ctx, char *err, int errlen) {
	@autoreleasepool {
		session *s = ctx;
		if (s->encoder == NULL) {
			return 0;
		}
		id<MTLComputeCommandEncoder> encoder =
			(__bridge_transfer id<MTLComputeCommandEncoder>)s->encoder;
		id<MTLCommandBuffer> commands =
			(__bridge_transfer id<MTLCommandBuffer>)s->commands;
		s->encoder = NULL;
		s->commands = NULL;
		[encoder endEncoding];
		[commands commit];
		[commands waitUntilCompleted];
		if (commands.error != nil) {
			tell(err, errlen, [NSString stringWithFormat:@"run: %@",
			                   commands.error.localizedDescription]);
			return 1;
		}
		s->gpu += commands.GPUEndTime - commands.GPUStartTime;
		return 0;
	}
}

double slayer_session_gpu(void *ctx) { return ((session *)ctx)->gpu; }

void slayer_session_close(void *ctx) {
	if (ctx == NULL) {
		return;
	}
	session *s = ctx;
	if (s->encoder != NULL) {
		id<MTLComputeCommandEncoder> encoder =
			(__bridge_transfer id<MTLComputeCommandEncoder>)s->encoder;
		[encoder endEncoding];
		(void)(__bridge_transfer id)s->commands;
	}
	(void)(__bridge_transfer id)s->buffers;
	(void)(__bridge_transfer id)s->pipelines;
	(void)(__bridge_transfer id)s->queue;
	(void)(__bridge_transfer id)s->device;
	free(s);
}
