// the C face of the bridge: what Go calls, in plain C types, so cgo never
// sees an Objective-C object. a context is an opaque handle; every call
// that can fail writes why into err and returns non-zero.

#ifndef SLAYER_METAL_BRIDGE_H
#define SLAYER_METAL_BRIDGE_H

#include <stdint.h>

// slayer_metal_open makes a context on the system's GPU with one compute
// kernel, compiled at runtime from source, so no metal toolchain is needed
// at build time. NULL on failure, with err saying why, and nodevice set
// when the reason is that there is no GPU at all.
void *slayer_metal_open(const char *source, const char *kernel, int *nodevice,
                        char *err, int errlen);

// slayer_metal_name is the GPU's own name, for the record of what ran.
const char *slayer_metal_name(void *ctx);

// slayer_metal_points runs the kernel once per point: points is n packed
// triples of floats, out has room for n floats, and the parameters are
// copied to buffer 2 as given.
int slayer_metal_points(void *ctx, const float *points, int n,
                        const void *params, int paramlen,
                        float *out, char *err, int errlen);

// slayer_metal_grid runs the kernel once per pixel of a picture width by
// height, a band of rows at a time, so no one command runs long enough
// for the system to stop it. the inputs, made into buffers once, are
// bound at 0 to count-1; the band's part of out, width * rows * per
// floats, at count; and the band's origin, two unsigned ints (0, first
// row), at count+1.
int slayer_metal_grid(void *ctx, int width, int height, int rows, int per,
                      const void **inputs, const long *lengths, int count,
                      float *out, char *err, int errlen);

// slayer_metal_gpu is the seconds the gpu has spent on every command the
// context has run, by the system's own timestamps on them: the time the
// work took on the gpu, and not the time the caller waited.
double slayer_metal_gpu(void *ctx);

// slayer_metal_close lets the context go.
void slayer_metal_close(void *ctx);

// a session is many kernels from one source, buffers that live on the
// device as long as it does, and batches of dispatches committed as one
// command, for work that runs many steps without coming back.

// slayer_session_open compiles the source once and makes a pipeline for
// each of count kernels, named in kernels. NULL on failure, as
// slayer_metal_open.
void *slayer_session_open(const char *source, const char **kernels,
                          int count, int *nodevice, char *err, int errlen);

// slayer_session_buffer makes a shared buffer of length bytes, zeroed,
// and is its number, or -1 with err saying why.
int slayer_session_buffer(void *s, long length, char *err, int errlen);

// slayer_session_contents is where a buffer's bytes are, in memory the
// cpu and the gpu share.
void *slayer_session_contents(void *s, int buffer);

// slayer_session_dispatch adds a dispatch of a kernel to the batch: its
// buffers bound from 0 in order, the params' bytes after them, over
// width by height threads; or, grouped, height groups of width threads.
int slayer_session_dispatch(void *s, int kernel, const int *buffers,
                            int count, const void *params, int paramlen,
                            int width, int height, int grouped, char *err,
                            int errlen);

// slayer_session_commit runs the batch, in order, waits for it, and
// begins another.
int slayer_session_commit(void *s, char *err, int errlen);

// slayer_session_gpu is the seconds the gpu has spent on the session.
double slayer_session_gpu(void *s);

// slayer_session_close lets the session and its buffers go.
void slayer_session_close(void *s);

#endif
