package diskcache

import "time"

// heldFor is how long after its last use an entry is never removed to
// make room: a run that has just read one may read it again for its
// next frame. Picked by the disk caches slice's author, 2026-11-12: a
// frame of the slowest scene today takes about two minutes.
const heldFor = 10 * time.Minute

// staleParts is how old a .part file is before it is taken for a killed
// writer's and removed. Picked by the same author, the same day: the
// largest entry, a staged scene's grid, is written in under a second.
const staleParts = time.Hour
