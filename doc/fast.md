# Rendering faster

Ray tracing is easy to start and expensive to run. This roadmap separates
four optimisations by the work they remove. The sequence matters: later
stages build on earlier ones. It is a design sequence, not a claim that
all four stages are implemented.

## 1. Bake the haze

A view ray takes 48 steps through haze. Each step takes another 10 toward
each lamp, sampling noise along the way—roughly 500 noise evaluations per
ray for one lamp. This dominates the swarm scene’s rendering time.

Before tracing a frame, sample density into a 3D texture and store each
lamp’s transmission in a shadow volume. Integration can then use hardware
texture interpolation instead of repeated noise evaluation.

This is an approximation absent from the CPU reference. Tests declare a
looser image tolerance, and cells must be small enough to resolve the
finest wisp with several samples.

## 2. Reuse resources between frames

Recompiling kernels, uploading unchanged buffers, and regrowing geometry
for every frame wastes work. A persistent engine should retain those
resources and update only what changes: camera, emitters, or pulse state.

## 3. Use hardware ray tracing

Metal exposes acceleration structures for hardware traversal and triangle
intersection. A scene-wide hierarchy built using surface-area cost can
improve on Slayer’s current split at the median face.

Analytic spheres use bounding boxes with custom intersection functions.
Keep Slayer’s software hierarchy as a fallback on hardware without ray
tracing support and as a reference for the new path.

## 4. Group rays by work

GPU threads execute in groups. A combined kernel can leave most of a group
waiting while one ray marches through haze and another immediately hits
the sky.

A wavefront tracer separates tracing into kernels and queues rays between
stages. Each group then performs similar work. Plan this stage alongside
hardware traversal so the queues feed the accelerator efficiently.
