// viewer.m: a window on Metal for grissom. A camera at the sphere's centre
// is the projection from inside, so each cell of the sky draws as a flat
// triangle. The earth is drawn from outside, as hexes: each corner of its
// triangles is a hex, and each point takes the colour of its nearest
// corner. Standing on the earth, what lies below the horizon is ground.

#import <Cocoa/Cocoa.h>
#import <Metal/Metal.h>
#import <MetalKit/MetalKit.h>
#include <string.h>
#include "viewer.h"

extern void goDrag(double dx, double dy);
extern void goScroll(double dy);
extern void goKey(int key);
extern int goView(float *uniforms, int width, int height);

static NSImage *lightIcon;
static NSImage *darkIcon;

const char *viewer_icons(const uint8_t *light, size_t light_size,
                         const uint8_t *dark, size_t dark_size) {
  @autoreleasepool {
    if (light == NULL || light_size == 0) return "missing light image";
    if (dark == NULL || dark_size == 0) return "missing dark image";
    NSData *lightData = [NSData dataWithBytes:light length:light_size];
    NSData *darkData = [NSData dataWithBytes:dark length:dark_size];
    NSImage *day = [[NSImage alloc] initWithData:lightData];
    NSImage *night = [[NSImage alloc] initWithData:darkData];
    if (day == nil) return "cannot decode light image";
    if (night == nil) return "cannot decode dark image";
    lightIcon = day;
    darkIcon = night;
    return NULL;
  }
}

// Keep the Dock artwork in step with the viewer's effective appearance.
static void applyApplicationIcon(NSAppearance *appearance) {
  if (lightIcon == nil || darkIcon == nil) return;
  NSString *matched = [appearance bestMatchFromAppearancesWithNames:
      @[NSAppearanceNameAqua, NSAppearanceNameDarkAqua]];
  NSApp.applicationIconImage =
      [matched isEqualToString:NSAppearanceNameDarkAqua] ? darkIcon : lightIcon;
}

// uniformFloats is the uniforms' size, as the shader's struct lays them out.
#define uniformFloats 36

static const char *shaders =
"#include <metal_stdlib>\n"
"using namespace metal;\n"
"struct Uniforms { float4x4 view; float outline; float width; float hex;\n"
"  float ground; float4 line; float4 up; float4 earth; float2 centre;\n"
"  float cross; float pad; };\n"
"struct Out { float4 position [[position]]; float3 bary; float3 world;\n"
"  float4 c0 [[flat]]; float4 c1 [[flat]]; float4 c2 [[flat]]; };\n"
"vertex Out vert(uint id [[vertex_id]],\n"
"                const device packed_float3 *corners [[buffer(0)]],\n"
"                const device uchar4 *colours [[buffer(1)]],\n"
"                constant Uniforms &u [[buffer(2)]]) {\n"
"  Out o;\n"
"  float3 p = float3(corners[id]);\n"
"  o.position = u.view * float4(p, 1.0);\n"
"  uint c = id % 3;\n"
"  o.bary = float3(c == 0, c == 1, c == 2);\n"
"  o.world = p;\n"
"  uint base = id - c;\n"
"  o.c0 = float4(colours[base]) / 255.0;\n"
"  o.c1 = float4(colours[base + 1]) / 255.0;\n"
"  o.c2 = float4(colours[base + 2]) / 255.0;\n"
"  return o;\n"
"}\n"
"fragment float4 frag(Out in [[stage_in]], constant Uniforms &u [[buffer(0)]]) {\n"
"  float3 b = in.bary;\n"
"  float3 colour = in.c0.rgb;\n"
"  float edge = min(min(b.x, b.y), b.z);\n"
"  if (u.hex > 0.5) {\n"
"    float top = max(max(b.x, b.y), b.z);\n"
"    float second = b.x + b.y + b.z - top - min(min(b.x, b.y), b.z);\n"
"    colour = (b.x == top) ? in.c0.rgb : ((b.y == top) ? in.c1.rgb : in.c2.rgb);\n"
"    edge = top - second;\n"
"  }\n"
"  float drawn = 1.0 - smoothstep(0.0, fwidth(edge) * u.width, edge);\n"
"  colour = mix(colour, u.line.rgb, drawn * u.outline);\n"
"  if (u.ground > 0.5 && dot(normalize(in.world), u.up.xyz) < 0.0) {\n"
"    colour = u.earth.rgb;\n"
"  }\n"
"  if (u.cross > 0.5) {\n"
"    float2 d = abs(in.position.xy - u.centre);\n"
"    if ((d.x < 1.5 && d.y < 16.0) || (d.y < 1.5 && d.x < 16.0)) {\n"
"      colour = u.line.rgb;\n"
"    }\n"
"  }\n"
"  return float4(colour, 1.0);\n"
"}\n";

// a mesh on the device.
@interface Mesh : NSObject
@property(strong) id<MTLBuffer> corners;
@property(strong) id<MTLBuffer> colours;
@property int count;
@end

@implementation Mesh
@end

static Mesh *meshOn(id<MTLDevice> device, viewer_mesh from) {
  Mesh *mesh = [Mesh new];
  mesh.count = from.count;
  mesh.corners =
      [device newBufferWithBytes:from.corners
                          length:(NSUInteger)from.count * 9 * sizeof(float)
                         options:MTLResourceStorageModeShared];
  mesh.colours =
      [device newBufferWithBytes:from.colours
                          length:(NSUInteger)from.count * 12
                         options:MTLResourceStorageModeShared];
  return mesh;
}

static void drawMesh(id<MTLRenderCommandEncoder> encoder, Mesh *mesh,
                     float *uniforms) {
  [encoder setVertexBuffer:mesh.corners offset:0 atIndex:0];
  [encoder setVertexBuffer:mesh.colours offset:0 atIndex:1];
  [encoder setVertexBytes:uniforms
                   length:uniformFloats * sizeof(float)
                  atIndex:2];
  [encoder setFragmentBytes:uniforms
                     length:uniformFloats * sizeof(float)
                    atIndex:0];
  [encoder drawPrimitives:MTLPrimitiveTypeTriangle
              vertexStart:0
              vertexCount:(NSUInteger)mesh.count * 3];
}

static Mesh *meshes[3];

// the hud: a picture slayer's hud drew, laid over the scene, and the
// pipeline that lays it, blending its premultiplied light over the scene.
static id<MTLDevice> theDevice;
static id<MTLTexture> hudPicture;
static id<MTLRenderPipelineState> overlay;

static const char *overlayShaders =
"#include <metal_stdlib>\n"
"using namespace metal;\n"
"struct Quad { float4 position [[position]]; float2 uv; };\n"
"vertex Quad hudVert(uint id [[vertex_id]]) {\n"
"  float2 p = float2((id << 1) & 2, id & 2);\n"
"  Quad q;\n"
"  q.position = float4(p * 2.0 - 1.0, 0.0, 1.0);\n"
"  q.uv = float2(p.x, 1.0 - p.y);\n"
"  return q;\n"
"}\n"
"fragment float4 hudFrag(Quad in [[stage_in]], texture2d<float> t [[texture(0)]]) {\n"
"  constexpr sampler s(filter::linear);\n"
"  return t.sample(s, in.uv);\n"
"}\n";

void viewer_hud(const uint8_t *rgba, int width, int height) {
  if (rgba == NULL || width <= 0 || height <= 0) {
    hudPicture = nil;
    return;
  }
  if (hudPicture == nil || (int)hudPicture.width != width ||
      (int)hudPicture.height != height) {
    MTLTextureDescriptor *shape = [MTLTextureDescriptor
        texture2DDescriptorWithPixelFormat:MTLPixelFormatRGBA8Unorm
                                     width:width
                                    height:height
                                 mipmapped:NO];
    shape.usage = MTLTextureUsageShaderRead;
    shape.storageMode = MTLStorageModeShared;
    hudPicture = [theDevice newTextureWithDescriptor:shape];
  }
  [hudPicture replaceRegion:MTLRegionMake2D(0, 0, width, height)
                mipmapLevel:0
                  withBytes:rgba
                bytesPerRow:(NSUInteger)width * 4];
}

// encode draws one frame: the earth from outside, or the sky with its
// coarse sphere behind it; then the hud over it.
static void encode(id<MTLRenderCommandEncoder> encoder,
                   id<MTLRenderPipelineState> pipeline, int width,
                   int height) {
  // the sky's uniforms first, then the earth's, which standing on the
  // earth draws over the sky from the eye.
  float uniforms[2 * uniformFloats] = {0};
  int mode = goView(uniforms, width, height);
  [encoder setRenderPipelineState:pipeline];
  if (mode == 1) {
    [encoder setFrontFacingWinding:MTLWindingCounterClockwise];
    [encoder setCullMode:MTLCullModeBack];
    drawMesh(encoder, meshes[viewer_earth], uniforms);
  } else {
    [encoder setCullMode:MTLCullModeNone];
    drawMesh(encoder, meshes[viewer_back], uniforms);
    drawMesh(encoder, meshes[viewer_sky], uniforms);
    if (mode == 2) {
      [encoder setFrontFacingWinding:MTLWindingCounterClockwise];
      [encoder setCullMode:MTLCullModeBack];
      drawMesh(encoder, meshes[viewer_earth], uniforms + uniformFloats);
    }
  }
  if (hudPicture != nil && overlay != nil) {
    [encoder setRenderPipelineState:overlay];
    [encoder setCullMode:MTLCullModeNone];
    [encoder setFragmentTexture:hudPicture atIndex:0];
    [encoder drawPrimitives:MTLPrimitiveTypeTriangle
                vertexStart:0
                vertexCount:3];
  }
}

static id<MTLRenderPipelineState> pipelineOn(id<MTLDevice> device,
                                             char *err, int errlen) {
  NSError *why = nil;
  theDevice = device;
  id<MTLLibrary> library =
      [device newLibraryWithSource:@(shaders) options:nil error:&why];
  if (library == nil) {
    snprintf(err, errlen, "%s", why.localizedDescription.UTF8String);
    return nil;
  }
  MTLRenderPipelineDescriptor *described = [MTLRenderPipelineDescriptor new];
  described.vertexFunction = [library newFunctionWithName:@"vert"];
  described.fragmentFunction = [library newFunctionWithName:@"frag"];
  described.colorAttachments[0].pixelFormat = MTLPixelFormatBGRA8Unorm;
  id<MTLRenderPipelineState> pipeline =
      [device newRenderPipelineStateWithDescriptor:described error:&why];
  if (pipeline == nil) {
    snprintf(err, errlen, "%s", why.localizedDescription.UTF8String);
    return nil;
  }
  id<MTLLibrary> overlays =
      [device newLibraryWithSource:@(overlayShaders) options:nil error:&why];
  if (overlays == nil) {
    snprintf(err, errlen, "%s", why.localizedDescription.UTF8String);
    return nil;
  }
  MTLRenderPipelineDescriptor *laid = [MTLRenderPipelineDescriptor new];
  laid.vertexFunction = [overlays newFunctionWithName:@"hudVert"];
  laid.fragmentFunction = [overlays newFunctionWithName:@"hudFrag"];
  laid.colorAttachments[0].pixelFormat = MTLPixelFormatBGRA8Unorm;
  laid.colorAttachments[0].blendingEnabled = YES;
  laid.colorAttachments[0].sourceRGBBlendFactor = MTLBlendFactorOne;
  laid.colorAttachments[0].destinationRGBBlendFactor =
      MTLBlendFactorOneMinusSourceAlpha;
  laid.colorAttachments[0].sourceAlphaBlendFactor = MTLBlendFactorOne;
  laid.colorAttachments[0].destinationAlphaBlendFactor =
      MTLBlendFactorOneMinusSourceAlpha;
  overlay = [device newRenderPipelineStateWithDescriptor:laid error:&why];
  if (overlay == nil) {
    snprintf(err, errlen, "%s", why.localizedDescription.UTF8String);
    return nil;
  }
  return pipeline;
}

@interface SkyView : MTKView
@end

@implementation SkyView
- (void)viewDidChangeEffectiveAppearance {
  [super viewDidChangeEffectiveAppearance];
  applyApplicationIcon(self.effectiveAppearance);
}
- (BOOL)acceptsFirstResponder { return YES; }
- (void)mouseDragged:(NSEvent *)event { goDrag(event.deltaX, event.deltaY); }
- (void)scrollWheel:(NSEvent *)event { goScroll(event.scrollingDeltaY); }
- (void)keyDown:(NSEvent *)event {
  NSString *keys = event.charactersIgnoringModifiers;
  if (keys.length > 0) goKey([keys characterAtIndex:0]);
}
@end

@interface Painter : NSObject <MTKViewDelegate, NSApplicationDelegate>
@property(strong) id<MTLCommandQueue> queue;
@property(strong) id<MTLRenderPipelineState> pipeline;
@end

static Painter *painter;
static MTKView *shown;
static MTLClearColor background = {0.02, 0.02, 0.03, 1};

@implementation Painter
- (void)mtkView:(MTKView *)view drawableSizeWillChange:(CGSize)size {}
- (BOOL)applicationShouldTerminateAfterLastWindowClosed:(NSApplication *)app {
  return YES;
}
- (void)drawInMTKView:(MTKView *)view {
  MTLRenderPassDescriptor *pass = view.currentRenderPassDescriptor;
  if (pass == nil) return;
  CGSize size = view.drawableSize;
  id<MTLCommandBuffer> commands = [self.queue commandBuffer];
  id<MTLRenderCommandEncoder> encoder =
      [commands renderCommandEncoderWithDescriptor:pass];
  encode(encoder, self.pipeline, (int)size.width, (int)size.height);
  [encoder endEncoding];
  [commands presentDrawable:view.currentDrawable];
  [commands commit];
}
@end

void viewer_colours(int which, const uint8_t *colours, int count) {
  memcpy(meshes[which].colours.contents, colours, (size_t)count * 12);
}

void viewer_background(double r, double g, double b) {
  background = MTLClearColorMake(r, g, b, 1);
  if (shown != nil) shown.clearColor = background;
}

int viewer_run(viewer_mesh sky, viewer_mesh back, viewer_mesh earth,
               const char *title, char *err, int errlen) {
  @autoreleasepool {
    [NSApplication sharedApplication];
    [NSApp setActivationPolicy:NSApplicationActivationPolicyRegular];
    applyApplicationIcon(NSApp.effectiveAppearance);
    id<MTLDevice> device = MTLCreateSystemDefaultDevice();
    if (device == nil) {
      snprintf(err, errlen, "no metal device");
      return 1;
    }
    painter = [Painter new];
    painter.pipeline = pipelineOn(device, err, errlen);
    if (painter.pipeline == nil) return 1;
    painter.queue = [device newCommandQueue];
    meshes[viewer_sky] = meshOn(device, sky);
    meshes[viewer_back] = meshOn(device, back);
    meshes[viewer_earth] = meshOn(device, earth);
    NSRect frame = NSMakeRect(0, 0, 1400, 1000);
    NSWindow *window = [[NSWindow alloc]
        initWithContentRect:frame
                  styleMask:NSWindowStyleMaskTitled |
                            NSWindowStyleMaskClosable |
                            NSWindowStyleMaskMiniaturizable |
                            NSWindowStyleMaskResizable
                    backing:NSBackingStoreBuffered
                      defer:NO];
    SkyView *view = [[SkyView alloc] initWithFrame:frame device:device];
    view.colorPixelFormat = MTLPixelFormatBGRA8Unorm;
    view.clearColor = background;
    view.preferredFramesPerSecond = 60;
    view.delegate = painter;
    shown = view;
    window.contentView = view;
    window.title = @(title);
    [window center];
    [window makeKeyAndOrderFront:nil];
    [window makeFirstResponder:view];
    NSApp.delegate = painter;
    [NSApp activateIgnoringOtherApps:YES];
    [NSApp run];
  }
  return 0;
}

int viewer_shot(viewer_mesh sky, viewer_mesh back, viewer_mesh earth,
                int width, int height, uint8_t *out, char *err, int errlen) {
  @autoreleasepool {
    id<MTLDevice> device = MTLCreateSystemDefaultDevice();
    if (device == nil) {
      snprintf(err, errlen, "no metal device");
      return 1;
    }
    id<MTLRenderPipelineState> pipeline = pipelineOn(device, err, errlen);
    if (pipeline == nil) return 1;
    meshes[viewer_sky] = meshOn(device, sky);
    meshes[viewer_back] = meshOn(device, back);
    meshes[viewer_earth] = meshOn(device, earth);
    MTLTextureDescriptor *shape = [MTLTextureDescriptor
        texture2DDescriptorWithPixelFormat:MTLPixelFormatBGRA8Unorm
                                     width:width
                                    height:height
                                 mipmapped:NO];
    shape.usage = MTLTextureUsageRenderTarget;
    shape.storageMode = MTLStorageModeShared;
    id<MTLTexture> target = [device newTextureWithDescriptor:shape];
    MTLRenderPassDescriptor *pass = [MTLRenderPassDescriptor new];
    pass.colorAttachments[0].texture = target;
    pass.colorAttachments[0].loadAction = MTLLoadActionClear;
    pass.colorAttachments[0].storeAction = MTLStoreActionStore;
    pass.colorAttachments[0].clearColor = background;
    id<MTLCommandQueue> queue = [device newCommandQueue];
    id<MTLCommandBuffer> commands = [queue commandBuffer];
    id<MTLRenderCommandEncoder> encoder =
        [commands renderCommandEncoderWithDescriptor:pass];
    encode(encoder, pipeline, width, height);
    [encoder endEncoding];
    [commands commit];
    [commands waitUntilCompleted];
    [target getBytes:out
         bytesPerRow:(NSUInteger)width * 4
          fromRegion:MTLRegionMake2D(0, 0, width, height)
         mipmapLevel:0];
  }
  return 0;
}
