# A field guide to the manufactory

The written catalogue has a companion specimen book. It gives each named
kind a picture, detail label and source notes. Search and family/detail
filters work locally in the browser; no web service is involved.

With the [Game setup](developing.md), render the default book:

```sh
game run catalogue-render
```

Open `out/catalogue/index.html` in a browser. The default is a small,
quick-detail view of every named kind. These are native Metal renders;
the command refuses an unavailable GPU rather than substituting a CPU image.
No survey or terrain download is needed for the specimen book.

The default takes four samples per pixel against a uniform backdrop.

List the names or choose a few, using Game's argument forwarding:

```sh
game run catalogue-list
game run catalogue -- render spitzer hubble webb --out out/space-book
```

The destination must be new. A second invocation never replaces a previous
book. Completed images remain available if a later specimen fails. The index
keeps pending and failed entries visible; a partial book is not a successful
full run.

## Detail comparisons

Copy [the default plan](../cmd/catalogue/plan.yaml) and change its `details`
to `[hull, clumps, quick, demo]`. Then render selected kinds:

```sh
game run catalogue -- render coast-redwood giant-sequoia webb --plan my-book.yaml --out out/details
```

A kind produces only the requested details it actually supports. Selecting
a kind that supports none of them is refused before output is created.

The [menu](../cmd/catalogue/menu.yaml) records current support, and a test
constructs every declared kind/detail pair. Unsupported detail is not relabelled
as a successful render.

The plan also controls framing, distant lighting, background, attached-light
exposure, tone and image budgets. Its distances scale with each specimen's
physical envelope. Detail changes retain that envelope and camera framing.
These are authored display settings, not measured illumination.

A few entries add a view heading in the menu, so the default camera sees
the useful side of a shield or dome slit. The plan's camera direction is
rotated by that heading; the resulting pose stays fixed across details.
Loose shoots use their stem axis as the view's up, so they read upright.

Ground templates also declare a viewing elevation to look through their
observing slits. This changes the catalogue camera, not the telescope pose.

For an inspection view, add `cutaway: true` to a copy of the plan. The book
hides an observatory's enclosure and labels that view explicitly. The source
object's bounds and the camera framing stay fixed across detail levels.
Other specimens keep their usual appearance and receive no cutaway label.

A narrower field of view and smaller `distance` can show the instrument more
closely. These are display choices for the book; scene recipes still make
the complete object. Cutaway records describe the visible submitted geometry.

Every plan passes through Puzzlemouth and the bundled
[JSON Schema](../cmd/catalogue/plan.schema.json). Sample and stored-face
budgets are checked before GPU submission. They do not predict all temporary
construction allocations; keep render jobs serial on a shared machine.

## What the book records

Each specimen has its own directory, containing a PNG, the input settings
and a JSON render record. The record identifies the executing binary, any
compiled Git stamp, physical bounds, stored faces and actual GPU timings and
native query counts. Missing build provenance is explicitly unknown.

The index's "Rendered on the native GPU path" label means that view completed
with real native closest-hit refinements. It does not establish CPU agreement
for every view. Numerical comparisons and scientific validation remain
separate from a rendered picture.

Source links distinguish measured reference dimensions from authored scenery.
The named campfire and bonfire entries show their static fuel geometry;
volumetric fire recipes and shared weather are separate scene capabilities.
The book currently covers named factory kinds, not every possible scene recipe.

The interface and renderer are still being validated on the catalogue branch.
See the [feature backlog](feature-backlog.md) for the wider catalogue work.
