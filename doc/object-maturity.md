# Object maturity

The manufactory can report on its own kinds: how much each is made of,
and where its numbers came from. This page says what the report counts,
how to read it, and what it cannot tell you.

```sh
game run maturity
go run ./tools/maturity -family spacecraft
go run ./tools/maturity -numbers hubble webb
```

In code it is `manufacture.MaturityReport`, one `Maturity` a kind, in
alphabetical order.

## What each column is

| Column | What it counts |
|---|---|
| kind | the name `KindNamed` takes |
| family | kinds one maker builds: spacecraft, observatory, plant |
| parts | solids the kind is assembled from, at quick detail |
| pieces | separate constructed pieces, at quick detail; see below |
| mesh faces | faces in the kind's mesh groups, at quick detail |
| looks | surfaces the kind wears |
| lights | lights it carries |
| details | the representations it can be made at |
| size | its own box at quick detail: across, up, deep |
| measured | named numbers it reads that have a pinned source |
| picked | named numbers it reads that someone chose |

A kind is built of parts, of meshes, or of both, so one of those two
columns is often nought. `-numbers` lists the measured and picked names
under each kind.

## Parts and pieces

A part is one solid the maker adds. A piece is a run of triangles that
touch: two triangles are of one piece if they share a corner, or are
joined through others that do.

The two differ where one part is many things. A segmented mirror is one
part and hundreds of panels, each its own piece. OWL is built of 526
mesh members and is 3,788 pieces, most of them mirror panels. The ELT
is 175 parts and 972 pieces.

Cutting a round surface finer adds triangles and no pieces, so an
instrument has the same pieces at every detail.

Use pieces, not parts or mesh members, when asking how much was
constructed. Two cautions. Pieces that touch count as one, so a model
of welded solids reads low. For a plant the pieces are leaves and
needles; the count runs to millions and grows with the detail.

## Reading it

Pieces say how much was constructed. Measured and picked say
how much of it rests on a source. Read them together.

- Many pieces and several measured numbers: a detailed model on real
  dimensions. The large observatories and Webb are here.
- Many pieces and one measured number: a detailed model scaled from one
  dimension, with proportions chosen by eye. Hubble and Spitzer are here.
- Few pieces and no measured number: a plain stand-in.

## What it does not tell you

The report counts. It does not judge.

- No dimension is compared with the thing itself. A model can read ten
  measured numbers and still put them together wrongly.
- It does not say what a kind's tests prove, or whether it has any.
- It does not know what was left out. That is in each kind's own notes.

## Which numbers are counted

A number is counted for a kind if the kind reads it from research or
from one of the manufactory's books of choices, either while it is made
or when its row of the table is built.

A row that reads measured numbers as the table is built says whose they
are by building itself through `rowOf`. The large observatories and the
space telescopes do. A new row that reads `research.Value` before its
kind is made needs the same, or its measured count will read nought.

Numbers a kind takes from a file that is not a book of named numbers
are not counted. The lunar module, the launch stacks, the summit
instruments' table and the yard equipment keep theirs that way, so
their measured and picked counts are low for that reason alone.

## Cost

The whole report makes every kind at every detail, which takes a few
seconds because of the plants. It watches every number read in the
process while it runs, so run it by itself and not beside a render.
