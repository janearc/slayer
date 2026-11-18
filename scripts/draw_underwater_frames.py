#!/usr/bin/env python3
"""Directly render remaining underwater still jobs with bounded concurrency."""
from concurrent.futures import ThreadPoolExecutor, as_completed
import os
from pathlib import Path
import subprocess

REPO = Path(__file__).resolve().parents[1]
ROOT = (Path(os.environ['SLAYER_RENDERS'])
        / 'animations' / '2026-11-16-studio-alfa-underwater')
DRAW = str(REPO / 'bin' / 'slayer')
ENV = os.environ.copy()
ENV.update({
    'GOMAXPROCS': '16',
})

def draw(frame: int) -> tuple[int, int]:
    job = ROOT / 'jobs' / f'frame-{frame:06}.yaml'
    log = ROOT / 'logs' / f'frame-{frame:06}.txt'
    log.parent.mkdir(exist_ok=True)
    with log.open('w') as out:
        result = subprocess.run([DRAW, 'draw', str(job), '--fresh'],
                                env=ENV, stdout=out, stderr=subprocess.STDOUT)
    print(f'frame {frame:06}: exit {result.returncode}', flush=True)
    return frame, result.returncode

failed = []
with ThreadPoolExecutor(max_workers=3) as pool:
    futures = [pool.submit(draw, frame) for frame in range(1, 36)]
    for future in as_completed(futures):
        frame, status = future.result()
        if status:
            failed.append(frame)
if failed:
    raise SystemExit(f'failed frames: {failed}')
print('all remaining 35 frames rendered', flush=True)
