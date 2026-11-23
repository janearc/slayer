# Scrooge

Scrooge decides how much of a set a picture pays for. It lives in
`maths/strategies/scrooge`. This is its design; nothing in it is built
yet.

## Why

A set is a description, and drawing it today means making all of it
first. `sets/futian-shore.yaml` is 130 KB; staged, it is 108,083 things,
4.3 billion faces referenced and 14 to 20 GB, made in nearly two minutes
before the first ray, at any picture size. A 16 by 9 picture of it costs
what a large one does to set up. See `doc/plan-before-staging.md` for the
measurements.

Scrooge turns that around. Nothing is made until a picture asks for it,
and nothing is made finer than the picture can show.

## The tree, inside out

Every object in a set is a node, from a whole city to a window. A node
that has not been asked for detail is a promise: a recipe, a seed and a
place, with answers worked out from its numbers alone:

- `Box()`: what holds everything it could make. Never too small.
- `Coverage(cell)`: how much of a cell that wide it blocks, from nought
  to one. Never too much.
- `Emission()`: the summed light of what it carries.
- `Detail(footprint)`: its children, as fine as the footprint asks and
  no finer.

A city is one box until a camera asks. Asked, it gives its districts;
each district is asked again at the footprint where it stands, and the
near ones give blocks, then buildings, then windows, while the far ones
answer as boxes with their light summed. What lies behind a closed pixel
is never asked.

The manufactory's base type implements the node: every kind answers the
four. The set gives its top nodes. Scrooge knows nothing about scenes,
sets or devices; it imports only maths.

## Frugal by contract

- **One layer at a time.** `Detail` gives a node's immediate children,
  never its subtree. The asker decides, child by child, whether to ask
  again.
- **Nothing finer than asked.** A node whose children would all be
  smaller than the footprint gives none, and answers as its own
  aggregate.
- **Made as taken.** Children come as an iterator, each made when the
  asker takes it. An asker that stops early, because the pixels behind
  are closed, leaves the rest unmade.
- **Consistent on average.** A node's coverage, light and colour are
  the mean of its children's. Otherwise a tree brightens or thins as it
  opens, and a film shows it popping.

`scrooge.Frugal` checks each kind against these in its tests: asked at
footprint F, it fails on a node finer than F that is not an aggregate,
on anything made and not taken, and on a parent whose average its
children do not keep.

## How the tree is walked

Four known methods, together.

1. **A cut, refined by priority.** Scrooge keeps a cut of open
   nodes in a queue, ordered by how much each could change the picture:
   its span in pixels times how open those pixels still are. It splits
   the worst first and stops when every node on the cut is under a
   pixel, or its budget is spent.

   This is the cut of lightcuts (Walter
   and others, 2005) and the split queue of ROAM (Duchaineau and others,
   1997).
2. **Bounded memory by forgetting.** When its memory budget is full,
   scrooge collapses the subtree that matters least, by the same
   priority, and keeps the subtree's aggregate in its parent. Children
   are seeded, so a forgotten subtree grows again the same. Growing
   again replaces storing; nothing goes to disc.

   This is the forgetting
   of simplified memory-bounded A* (Russell, 1992), which keeps a game
   search inside a fixed memory by dropping its least promising leaves
   and backing their values up into their parents.
3. **Levels by footprint, kept in a cache.** Every ask carries a
   footprint: a camera ray one pixel, a shadow ray its cone toward the
   light, a cascade its cell's width. The footprint picks the level a
   node is grown to, and what is grown is kept, least recently used
   first out, so neighbouring rays and the next frame of a film reuse
   it.

   This is multiresolution geometry caching with ray differentials
   (Christensen and others, 2003), which drew production scenes far
   larger than memory with small caches.
4. **Rays brought to the geometry.** Rays are queued by the node they
   wait on, so a node grows its detail once, serves every ray that needs
   it, and can then be let go. Tiles of the picture keep the queues
   short. This is memory-coherent ray tracing (Pharr and others, 1997).

Occlusion decides what is asked at all. The occlusion cascades (see
`doc/plan-before-staging.md`) walk the top nodes front to back; a pixel
under a 256th open is closed, and nothing behind it is asked. The same
cascades from the sun say which hidden nodes still shade what is seen,
and those answer as aggregates.

## Where each part lives

In `maths/strategies/scrooge`:

- `node.go`: the node, its aggregate and the footprint.
- `resolver.go`: open a node or take its aggregate, for a footprint.
- `cut.go`: the cut and its priority queue.
- `forget.go`: the memory budget and the collapse into parents.
- `bvh.go`: the surface area heuristic over whatever boxes a node gives,
  built when it is opened, and the walk front to back.
- `queue.go`: rays waiting on nodes.
- `frugal.go`: the checker every kind's tests use.

Beside it, in `maths/strategies`: the eye, the light tree, and the
cascades, which scrooge calls and which call it.

Outside: the manufactory implements the node for every kind; the set
gives its top nodes; the camera and the renderers run the passes, and
they alone know pixels, the processor and the GPU. For the GPU, which
cannot open a node in the middle of a trace, scrooge resolves the cut
on the processor for each frame and the GPU is given only what is open,
an unopened node as one fuzzy box whose coverage is its extinction.

## What this is not

- Not a change to any set file.
- Not a loss of anything a pixel would show. A draw with every node
  opened to its finest is the reference a scrooge draw is compared
  against, picture to picture.
- Not exact for a node it averages. The aggregate is a mean; what a
  pixel cannot show is not drawn.

## Order of work

1. This doc, then the structs and interfaces with their tests and
   anti-tests, unrun at first.
2. The cut and the forgetting, against a made-up tree of numbers.
3. One kind that grows by footprint: the Norway pine, the hardest.
4. The set's top nodes, and the occlusion cascades over them.
5. A draw of Futian from scrooge's cut against the reference, at 16 by
   9 and 480 by 270: time, memory and the picture.
