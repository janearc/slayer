# Scrooge

Scrooge is an experiment in selecting geometry detail before staging.
`SLAYER_SCROOGE=1` enables view-dependent detail selection for
eligible manufactured things and stand members, and prepares some
geometry in parallel. eligibility depends on the kind’s available
extent and detail levels, its scale and whether it is burning.
compare against the same scene and build with the switch unset.

[planning before staging](plan-before-staging.md) describes the
advisory plan and the separate out-of-view omission trial. these
switches exercise parts of the design. the full renderer does not
yet walk one lazy hierarchy of every object and every ray.

## Hierarchy and detail contract

`maths/strategies/scrooge.Node` offers:

- `Aggregate()` for a conservative box, average surface colour, summed
  emission and the next layer’s finest detail size.
- `Coverage(cell)` for the fraction of a cell the node blocks.
  an underestimate retains potentially visible geometry.
- `Detail(footprint)` for immediate, lazily constructed children.
  it can return none when the aggregate represents the requested
  detail scale.

`Frugal` checks the node’s bounds and detail contract. changing
detail should preserve average appearance and needs tests across
camera motion.

## Design for a bounded renderer

the cut is a set of nodes refined by projected effect while a budget
remains. collapsing a low-priority branch keeps its aggregate and
lets the deterministic recipe rebuild its children later. rays
can queue by the geometry they need, amortizing that construction.
camera, shadow and volume requests have different footprints.

the design takes the priority queue from Lightcuts and ROAM, bounded
forgetting from simplified memory-bounded A*, and footprint-dependent
geometry from ray-differential caches and memory-coherent ray tracing.
a Metal frame would receive the selected geometry after CPU planning.
off-camera influence and aggregate error still need validation.
