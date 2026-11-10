// viewer.h: the C face of the window, in plain C types, so cgo never sees
// an Objective-C object. The shape follows slayer's metal bridge.

#ifndef GRISSOM_VIEWER_H
#define GRISSOM_VIEWER_H

#include <stdint.h>
#include <stddef.h>

// viewer_icons loads both application appearances. Data is copied; an error
// is a static diagnostic string, and NULL means both images are ready.
const char *viewer_icons(const uint8_t *light, size_t light_size,
                         const uint8_t *dark, size_t dark_size);

// viewer_mesh is triangles to draw: count * 9 floats of corners on the
// unit sphere, and count * 3 * 4 bytes of colour, rgba a corner.
typedef struct {
  const float *corners;
  const uint8_t *colours;
  int count;
} viewer_mesh;

// The meshes the window draws, by their place in viewer_colours.
enum { viewer_sky = 0, viewer_back = 1, viewer_earth = 2 };

// viewer_run opens a window and draws until it closes: the sky's data and
// a coarse sphere behind it when looking at the sky, the earth when
// looking at it. It returns non-zero with err saying why when it cannot
// start.
int viewer_run(viewer_mesh sky, viewer_mesh back, viewer_mesh earth,
               const char *title, char *err, int errlen);

// viewer_colours replaces a mesh's colours.
void viewer_colours(int which, const uint8_t *colours, int count);

// viewer_background sets the colour behind the triangles, sRGB from none
// to one.
void viewer_background(double r, double g, double b);

// viewer_shot draws one frame of width by height offscreen through the
// same pipeline and writes its pixels, bgra, into out.
int viewer_shot(viewer_mesh sky, viewer_mesh back, viewer_mesh earth,
                int width, int height, uint8_t *out, char *err, int errlen);

// viewer_hud lays a picture over the scene, rgba and premultiplied, width
// by height; no picture takes the hud away.
void viewer_hud(const uint8_t *rgba, int width, int height);

#endif
