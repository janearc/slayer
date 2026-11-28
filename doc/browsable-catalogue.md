# a field guide to the manufactory

make a specimen book with pictures of manufactured objects, source notes
and render settings. it opens as local HTML with search and filters for
family and detail.

from the checkout root, after [Game setup](building.md):

```sh
game run catalogue-list
game run catalogue-render -- spitzer hubble webb --out out/space-book
```

open `out/space-book/index.html`. omitting the names renders the catalogue
at its default detail. the specimen book needs no terrain download. the
default plan requests quick detail and four rays per pixel. it uses native
Metal by default. add `--renderer cpu` to choose the CPU explicitly.

render again to the same book to resume. matching cards are kept, failed
or pending cards are retried, and changes to a card’s plan, renderer or
profile book trigger a redraw. a bad card carries its error and the run
continues. the command reports any failures and exits nonzero. choose a
fresh output directory to redraw every card.

`--models DIR` loads external model books. the directory needs every book
named by the [menu](../cmd/catalogue/menu.yaml), even for a one-kind render.
when omitted, the catalogue uses its embedded books.

## one file to send

```sh
game run catalogue-pdf -- --book out/space-book --out out/space-book.pdf
```

the new PDF contains the pictures, source references, settings and render
provenance. by default, the command refuses a book with incomplete cards.
`--with-failures` gives each incomplete card a page carrying its result.
full render receipts and settings stay with the HTML book.

## compare detail and views

copy [the default plan](../cmd/catalogue/plan.yaml) and set its `details`
to `[hull, clumps, quick, demo]`. then select the kinds to render:

```sh
game run catalogue-render -- coast-redwood giant-sequoia webb --plan my-book.yaml --out out/details
```

each kind uses the intersection of the plan’s detail levels and the ones
it supports. when that intersection is empty, render refuses the choice.
framing stays fixed across detail levels. the plan can also change its
camera, lighting, tone and sample or geometry budgets. see the
[bundled plan schema](../cmd/catalogue/plan.schema.json) for the fields.

- `orbit: {yaw: 20, pitch: -8}` turns the camera in degrees while keeping
  its distance and target. pitch is around the camera’s right axis after
  yaw.
- `cutaway: true` hides a supported observatory’s enclosure and labels
  the resulting card.
- `studio: true` uses the plan’s sky and lamps even for a model with its
  own staging. Webb’s mirror can then show the studio surroundings.

## read a render record

a card’s PNG sits beside its settings and render record. the record holds
physical bounds, stored faces, binary provenance and the renderer used.
native Metal receipts include measured GPU timings and query counts.
source notes identify measured dimensions and authored appearance choices.
for geometry counts without rendering, use the [model inventory](model-resources.md).
