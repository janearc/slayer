#!/usr/bin/env python3
"""Make auditable still jobs by moving the Alfa, ocean time and fixed camera together."""
import math
import os
from pathlib import Path

ROOT = (Path(os.environ['SLAYER_RENDERS'])
        / 'animations' / '2026-11-16-studio-alfa-underwater')
TEMPLATE = Path(__file__).resolve().parents[1] / 'sets/shots/alfa-underwater-template.yaml'
SPEED = 3.5
FPS = 6
HEADING = math.radians(340)

scene_template = TEMPLATE.read_text()
for frame in range(36):
    t = frame / FPS
    east = -1500 + SPEED * math.sin(HEADING) * t
    north = SPEED * math.cos(HEADING) * t
    camera_east = east + 98
    camera_north = north + 50
    scene = scene_template
    scene = scene.replace('east: {value: -1500.000000, unit: m}',
                          f'east: {{value: {east:.6f}, unit: m}}')
    scene = scene.replace('north: {value: 0.000000, unit: m}',
                          f'north: {{value: {north:.6f}, unit: m}}')
    scene = scene.replace('time: {value: 0.000000, unit: s}',
                          f'time: {{value: {t:.6f}, unit: s}}')
    scene = scene.replace('location: {at: [-1402, 50, -15.12]}',
                          f'location: {{at: [{camera_east:.6f}, {camera_north:.6f}, -15.12]}}')
    scene = scene.replace('orientation: {look_at: [-1500, 0, -15.12]}',
                          f'orientation: {{look_at: [{east:.6f}, {north:.6f}, -15.12]}}')
    (ROOT / 'scenes' / f'frame-{frame:06}.yaml').write_text(scene)
    job = f'''$schema: https://github.com/janearc/slayer/schemas/job-v1.json
scene: {{set: {ROOT}/scenes/frame-{frame:06}.yaml}}
film: {{width: 1080, height: 1920, rays: 1, tall: 80}}
renderer: gpu
quality: exact
camera: {{name: underwater-control}}
at: {{value: {t:.6f}, unit: s}}
outputs:
  picture: {ROOT}/frames/frame-{frame:06}.png
  raw: {ROOT}/frames/frame-{frame:06}.pfm
'''
    (ROOT / 'jobs' / f'frame-{frame:06}.yaml').write_text(job)
print(f'generated {36} underwater scene snapshots and jobs in {ROOT}')
