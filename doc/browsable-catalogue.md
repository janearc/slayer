# A field guide to the manufactory

The written catalogue has a companion specimen book. It gives each named
kind a picture, detail label and source notes. Search and family/detail
filters work locally in the browser; no web service is involved.

With the [Game setup](developing.md), render the default book:

```sh
game run catalogue-render
```

Open `out/catalogue/index.html` in a browser. The default is a small,
quick-detail view of the kinds supporting that detail. These are native Metal
renders by default; the command does not silently substitute a CPU image.
For a model that needs CPU tracing, select it explicitly:

```sh
game run catalogue -- render harbor-seal --renderer cpu --out out/catalogue
```

CPU and GPU cards may share a book. Each card names its path and retains its
own receipt. Selecting a card with a different renderer redraws that card;
other cards are kept. Some menu entries do not support quick detail and need
an alternate plan.
No survey or terrain download is needed for the specimen book.

The default takes four samples per pixel against a uniform backdrop.

List the names or choose a few, using Game's argument forwarding:

```sh
game run catalogue-list
game run catalogue -- render spitzer hubble webb --out out/space-book
```

Plant and animal cards read checked model books. With no `--models`, the
catalogue uses its embedded books. To use external books, pass `--models`
with their directory. The catalogue checks the whole menu before selecting
cards, so the directory must contain every book referenced by that menu,
even when rendering one specimen. Missing or invalid books are refused.
Each profile-backed card records its descriptor and exact model bytes;
changing either causes that card to be redrawn on resume.

A specimen that fails does not stop the run. Its card is marked failed
with the reason, the run goes on, and at the end it names every specimen
that failed, one line each, and exits non-zero. A run with no failure exits
zero.

A second run resumes a book. Give it the same `--out`, which holds a
`catalogue.json` the command wrote: cards already rendered under the same
plan are kept and not redrawn, cards that failed or are pending are drawn,
kinds added to the menu since are added and drawn, and kinds the menu no
longer has are dropped from the index and named.

A card drawn under a
different plan is drawn again. Each card keeps the record of the binary
that drew it. A destination that exists and is not a book is refused and
left as it was. To draw everything again, give a new `--out`.

## One file to send

A finished book composes into a single PDF that holds what the HTML
holds. It has a cover (the title, the date the book's record was last
written, the counts of specimens and groups, and the binary that drew
it), a contents page with each specimen's page number, and then one page
for each specimen, in the menu's order.

A specimen's page has its picture, as large as the page allows with its
words, then the group and kind, the description, the recipe and source
notes, the references as links, and a provenance block: the settings
hash, the binary's hash, the compiled revision and the device.

A specimen whose words do not fit under its picture at a reasonable size
runs on to a second page. Every page after the cover has the book's
title and its number in the foot, and a reader's sidebar lists the
groups and kinds.

```sh
game run catalogue-pdf
game run catalogue -- pdf --book out/catalogue --out out/catalogue.pdf
```

The defaults are the ones shown. Prose is set in Helvetica, hashes and
addresses in Courier, nothing below 11 points. The PDF's standard fonts
cover Latin-1 and the common typographic marks; a menu entry with any
other character is refused, naming the character and the card, rather
than printed as a question mark.

A book holding pending or failed cards is refused, and the refusal names
them; run `render` again on the book to finish it. To compose such a book
anyway, give `--with-failures`: each card that did not render gets a page
saying so and why, and the cover counts them.

The PDF's destination must
be new. The full render receipts and settings stay in the book; the
document is the part you hand someone. Pictures are kept lossless, and
the 106-specimen book comes to about 2.4 MB.

## Detail comparisons

Copy [the default plan](../cmd/catalogue/plan.yaml) and change its `details`
to `[hull, clumps, quick, demo]`. Then render selected kinds:

```sh
game run catalogue -- render coast-redwood giant-sequoia webb --plan my-book.yaml --out out/details
```

A kind produces only the requested details it actually supports. Selecting
a kind that supports none of them is refused before output is created.

The [menu](../cmd/catalogue/menu.yaml) records current support. Unsupported
detail is not relabelled as a successful render.

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

An optional plan `orbit` turns that framed view without moving its target or
changing its distance. Supply both angles in degrees, for example
`orbit: {yaw: 20, pitch: -8}`. Yaw turns around the entry's up axis; pitch
turns around the camera's resulting right axis. An orbit clamps the final
elevation within 85 degrees of the horizon.

An absent or zero orbit keeps the
authored view exactly, and lighting is unchanged. The existing `wide` field
still controls field of view from 5 to 120 degrees.

For an inspection view, add `cutaway: true` to a copy of the plan. The book
hides an observatory's enclosure and labels that view explicitly. The source
object's bounds and the camera framing stay fixed across detail levels.
Other specimens keep their usual appearance and receive no cutaway label.

An entry may carry a staging of its own: Webb's is space, with one sun and
nothing behind it. To see what such a specimen is made of, add `studio: true`
to a copy of the plan. The plan's sky and lamps then light it, as they light
an entry with no staging. A mirror staged in space mirrors the dark.

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
compiled Git stamp, physical bounds and stored faces. GPU receipts hold actual
GPU timings and native query counts; CPU receipts name the CPU path and its
wall time. Missing build provenance is explicitly unknown.

The index's "Rendered on the native GPU path" label means that view completed
with real native closest-hit refinements. It does not establish CPU agreement
for every view. Numerical comparisons and scientific validation remain
separate from a rendered picture.

Source links distinguish measured reference dimensions from authored scenery.
The named campfire and bonfire entries show their static fuel geometry;
volumetric fire recipes and shared weather are separate scene capabilities.
The book currently covers named factory kinds, not every possible scene recipe.
