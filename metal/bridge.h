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

// slayer_metal_function selects an entry point while keeping context resources.
int slayer_metal_function(void *ctx, const char *kernel, char *err, int errlen);

// slayer_metal_name is the GPU's own name, for the record of what ran.
const char *slayer_metal_name(void *ctx);

// slayer_metal_limit sets the seconds one command may take before the
// wait for it gives up with an error and the context is no longer used.
void slayer_metal_limit(void *ctx, double seconds);

// slayer_stop_waits stops every wait for a command, in every context and
// session of the process, when stop is non-zero, and lets them wait
// again when it is zero. A stopped wait returns at its next look, the
// context or session it waited in is not used again, and the command it
// waited for is left to the device.
void slayer_stop_waits(int stop);

// slayer_metal_points runs the kernel once per point: points is n packed
// triples of floats, out has room for n floats, and the parameters are
// copied to buffer 2 as given.
int slayer_metal_points(void *ctx, const float *points, int n,
                        const void *params, int paramlen,
                        float *out, char *err, int errlen);

// A grid stream retains inputs and one output band, and runs the kernel
// once per pixel of a band of rows at a time, so no one command runs
// long enough for the system to stop it. The inputs are bound at 0 to
// count-1; the band's part of out, width * rows * per floats, at count;
// and the band's origin, two unsigned ints (0, first row), at count+1.
// The caller retains ctx. Bands use global row coordinates and are
// copied into the caller's buffer.
// Begin/add copy each input synchronously; no caller input pointer is retained.
// A failed add leaves a closable stream. Bands require all count inputs.
// A band is committed, waited out and copied in three calls, with at
// most two bands in flight, so the caller decides when the display gets
// a turn between bands, and reads one band while the next is drawn.
void *slayer_metal_grid_begin(void *ctx, int width, int height, int rows,
                             int per, int count, char *err, int errlen);
int slayer_metal_grid_add(void *stream, const void *input, long length,
                          char *err, int errlen);
// slayer_metal_grid_update replaces one added input's bytes, waiting out
// anything drawn ahead first, so a stream kept between pictures uploads
// only what changed. The same length lands in the same buffer; a new
// length makes a new one.
int slayer_metal_grid_update(void *stream, int index, const void *input,
                             long length, char *err, int errlen);
// slayer_metal_grid_commit commits the band of down rows from first
// without waiting: with nothing in flight, into the band buffer not last
// waited out; with one band in flight, into the other buffer, which is
// the one last waited out, so the caller copies that band first. It
// refuses a third band in flight.
int slayer_metal_grid_commit(void *stream, int first, int down, char *err,
                             int errlen);
// slayer_metal_grid_wait waits out the oldest band in flight. stopped is
// one when the system stopped it for the display: the stream and its
// kernel stay usable and the same rows may be committed again. It
// returns 2, not 1, when the wait was stopped by slayer_stop_waits.
int slayer_metal_grid_wait(void *stream, int *stopped, char *err,
                           int errlen);
// slayer_metal_grid_copy copies the band last waited out, down rows, into
// out; down must be the rows that band was committed with.
int slayer_metal_grid_copy(void *stream, int down, float *out, char *err,
                           int errlen);
// slayer_metal_grid_same sets same to one when an added input holds
// exactly length bytes and they are input's, read where the device reads
// them; it copies nothing.
int slayer_metal_grid_same(void *stream, int index, const void *input,
                           long length, int *same, char *err, int errlen);
// slayer_metal_grid_refresh copies into an added input of the same length
// only the parts of chunk bytes that differ from input's, waiting out
// anything drawn ahead first, and sets sent to the bytes it copied.
int slayer_metal_grid_refresh(void *stream, int index, const void *input,
                              long length, long chunk, long *sent, char *err,
                              int errlen);
// slayer_metal_grid_shape draws later bands at another size, keeping the
// inputs: it waits out anything drawn ahead and makes new band buffers
// only when the band's bytes change.
int slayer_metal_grid_shape(void *stream, int width, int height, int rows,
                            int per, char *err, int errlen);
// slayer_metal_grid_settle waits out and forgets every band in flight,
// whatever became of them, or forgets them unwaited when the kernel is
// stuck, so a drawing that stopped early leaves the stream ready. It is
// non-zero, with err saying why, when a band it waited out failed.
int slayer_metal_grid_settle(void *stream, char *err, int errlen);
void slayer_metal_grid_close(void *stream);

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

// slayer_session_limit is slayer_metal_limit for a session's batches.
void slayer_session_limit(void *s, double seconds);

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

// Build an acyclic plan. Nodes are kind/first/count triples; placements are
// packed float4x3 matrices and child/mask pairs. The root is an instance node.
int slayer_metal_accelerate(void *ctx, const void *vertices, long length,
                            int stride, const int32_t *nodes, int count,
                            const float *matrices, const uint32_t *references,
                            int root, int *unavailable, char *err, int errlen);
// slayer_metal_reaccelerate is slayer_metal_accelerate for a context that
// may have a hierarchy already: the new one takes its place once built,
// and a build that fails leaves the old one.
int slayer_metal_reaccelerate(void *ctx, const void *vertices, long length,
                              int stride, const int32_t *nodes, int count,
                              const float *matrices,
                              const uint32_t *references, int root,
                              int *unavailable, char *err, int errlen);
// slayer_metal_accelerate_over is slayer_metal_accelerate, or with
// replace the reaccelerate, reading the vertices from input index of a
// grid stream of the same context, in place, instead of making a copy of
// them. The input must already hold exactly the vertices' bytes; one of
// another length is not used and the vertices are copied as they are
// without it. The context keeps no reference to the input after the call.
int slayer_metal_accelerate_over(void *ctx, void *stream, int index,
                                 int replace, const void *vertices,
                                 long length, int stride,
                                 const int32_t *nodes, int count,
                                 const float *matrices,
                                 const uint32_t *references, int root,
                                 int *unavailable, char *err, int errlen);
// A context holds the mesh structures its last two builds placed, under
// their positions, and a group's structure while its placements and
// children stay: a later build keeps what it can, refits a deformed mesh
// where it may, and builds the rest. slayer_metal_acceleration_work says
// how many nodes of the last build each of those were.
void slayer_metal_acceleration_work(void *ctx, int *kept, int *refit,
                                    int *built);

// slayer_metal_acceleration_held is how many mesh structures the context
// holds between builds, a mesh's spare counted as one more.
int slayer_metal_acceleration_held(void *ctx);

// the faults slayer_metal_acceleration_fault sets for a context's next
// hierarchy build, which meets it once, for tests of a build that fails
enum {
	acceleration_fault_none = 0,       // defined: the build runs as it is
	acceleration_fault_allocation = 1, // defined: after every structure is
	                                   // described, one is said not made
	acceleration_fault_command = 2,    // defined: the command runs, then
	                                   // is said to have failed
};

// slayer_metal_acceleration_fault sets the fault the next build meets.
void slayer_metal_acceleration_fault(void *ctx, int fault);

// slayer_idle_nanoseconds is the system's time since the last keyboard,
// mouse or trackpad input, read from IOHIDSystem's HIDIdleTime, or -1
// when it cannot be read. It only reads.
long long slayer_idle_nanoseconds(void);

// slayer_display_asleep is one when the main display sleeps.
int slayer_display_asleep(void);

#endif
