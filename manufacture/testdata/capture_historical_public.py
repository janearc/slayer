#!/usr/bin/env python3
"""Capture selected historical catalogue records one process per case.

Run from the historical repository root under Game's tests lane after
building geometry_sample_selected.go. Records and logs stay in OUTSIDE_DIR.
"""
import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import time

BINARY = Path(sys.argv[1]).resolve()
OUTSIDE_DIR = Path(sys.argv[2]).resolve()
NAMES = Path('manufacture/testdata/historical_public_names.txt')
TIME_LIMIT = 120
RSS_LIMIT_KIB = 4 * 1024 * 1024
FREE_LIMIT = 30 * 1024**3
LEVELS = ('hull', 'clumps', 'quick', 'demo')
seeds = tuple(int(x) for x in os.getenv('CAPTURE_SEEDS', '5,11').split(','))
levels = tuple(os.getenv('CAPTURE_LEVELS', ','.join(LEVELS)).split(','))
if not seeds or set(seeds) - {5, 11} or not levels or set(levels) - set(LEVELS):
    raise SystemExit('invalid seed or detail selection')
OUTSIDE_DIR.mkdir(parents=True, exist_ok=True)

names = NAMES.read_text().splitlines()
if len(sys.argv) > 3:
    wanted = set(sys.argv[3:])
    unknown = wanted - set(names)
    if unknown:
        raise SystemExit(f"unknown requested names: {sorted(unknown)}")
    names = [name for name in names if name in wanted]

for name in names:
    for seed in seeds:
        for level in levels:
            key = f'{name}-{seed}-{level}'
            result = OUTSIDE_DIR / f'{key}.txt'
            if result.exists():
                continue
            if shutil.disk_usage(OUTSIDE_DIR).free < FREE_LIMIT:
                raise SystemExit('stopped: less than 30 GiB free')
            env = dict(os.environ, REFERENCE_NAME=name,
                       REFERENCE_SEED=str(seed), REFERENCE_LEVEL=level)
            start = time.monotonic()
            proc = subprocess.Popen([str(BINARY)], stdout=subprocess.PIPE,
                                    stderr=subprocess.PIPE, env=env)
            peak_kib = 0
            reason = ''
            while proc.poll() is None:
                time.sleep(.2)
                elapsed = time.monotonic() - start
                if elapsed > TIME_LIMIT:
                    reason = 'timeout'
                    proc.kill()
                    break
                sample = subprocess.run(['/bin/ps', '-o', 'rss=', '-p', str(proc.pid)],
                                        capture_output=True, text=True)
                rss = sample.stdout.strip()
                if rss.isdigit():
                    peak_kib = max(peak_kib, int(rss))
                if peak_kib > RSS_LIMIT_KIB:
                    reason = 'rss limit'
                    proc.kill()
                    break
            stdout, stderr = proc.communicate()
            elapsed = time.monotonic() - start
            prefix = f'{name} {seed} {level} '.encode()
            valid = proc.returncode == 0 and stdout.startswith(prefix) and stdout.count(b'\n') == 1
            if valid:
                result.write_bytes(stdout)
            else:
                reason = reason or f'bad output / exit {proc.returncode}'
                (OUTSIDE_DIR / f'{key}-{time.time_ns()}.error').write_bytes(stderr + b'\nstdout: ' + stdout)
            record = dict(name=name, seed=seed, level=level, seconds=round(elapsed, 3),
                          sampled_peak_kib=peak_kib, ok=valid, reason=reason)
            with (OUTSIDE_DIR / 'progress.jsonl').open('a') as log:
                log.write(json.dumps(record, sort_keys=True) + '\n')
            print(json.dumps(record, sort_keys=True), flush=True)
