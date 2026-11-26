#!/usr/bin/env python3
"""Compare checkpointed current public cases with their historical rows."""
import difflib
import json
from pathlib import Path
import sys

historical = Path(sys.argv[1])
current = Path(sys.argv[2])
name = sys.argv[3]
rows = []
for seed in (5, 11):
    for level in ('hull', 'clumps', 'quick', 'demo'):
        key = f'{name}-{seed}-{level}'
        before = historical / (key + '.txt')
        after = current / (key + '.txt')
        if not before.exists():
            continue
        if not after.exists():
            rows.append(dict(case=key, status='missing'))
            continue
        original, actual = before.read_text(), after.read_text()
        if original == actual:
            rows.append(dict(case=key, status='match'))
        else:
            diff = ''.join(difflib.unified_diff(original.splitlines(keepends=True),
                actual.splitlines(keepends=True), fromfile='historical', tofile='current'))
            (current / (key + '.diff')).write_text(diff)
            rows.append(dict(case=key, status='DIFF'))
with (current / 'comparisons.jsonl').open('a') as log:
    for row in rows:
        log.write(json.dumps(row, sort_keys=True) + '\n')
print(json.dumps(dict(name=name, cases=rows), sort_keys=True), flush=True)
