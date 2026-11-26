# A plant from a profile book

A plant recipe can read a species from a profile book beside a set, or from
another path relative to that set. This is a construction recipe, not a new
registry name or an alias:

```yaml
scene:
  recipes:
    oak:
      plant:
        family: seasonal
        file: ../models/plant-profiles.yaml
        species: valley-oak
      season: autumn
```

`file` and `species` are required. `family` selects the construction policy;
it may be `seasonal`, `conifer`, `fern`, or `cactus`, and defaults to
`seasonal` when omitted. The corresponding examples are
`manufacture/plants/seasonal/profiles.yaml`, `manufacture/plants/conifers/conifer-profiles.yaml`,
`manufacture/plants/ferns/fern-profiles.yaml`, and `manufacture/plants/cacti/cactus-profiles.yaml`.
The species must exist in the selected family's book. `season` is optional
for seasonal plants and may be `summer`, `autumn`, or `winter`; the other
families reject it. A new species can be added as a profile without adding a
Go kind. Each profile says who authored it and when;
`status: picked` records a modelling choice, while `status: read` or
`status: derived` also requires a `source`. The reader checks the whole book
and refuses an
invalid file, missing species or unknown recipe key. It never falls back to a
built-in species when a requested file cannot be read.

Fern profiles include an `anatomy` block for crown span and radius, stalk and
leaflet taper, and leaflet reach, shortening, girth and minimum width. All
fields are required and checked. Frond lean and variation, paired leaflet
placement and the stem tessellation remain fixed family construction choices;
the embedded generic fern is authored scenery, not a measured species.

A checked conifer book can also select a loose shoot, using
`plant: {family: conifer, file: ../models/conifer-profiles.yaml, species:
coast-redwood, representation: shoot}`. Coast redwood and giant sequoia
support this representation; the other current conifers and plant families
refuse it. Omitting `representation` keeps the whole plant.

The book is read and validated with the set. Its bytes participate in the
recipe and job render keys, and the checked set is frozen for a draw and its
resume. Editing a book changes those keys; an already read job keeps its
snapshot. Foliage cells are shared only within a checked book and keyed by the
geometry of the foliage, detail and variant. Two versions with the same
species name do not share the wrong leaves. Species names do not control
scene behavior: a profile named `fire` remains a plant, not a fire.

A tower or a tree-bearing clustered city can use the same seasonal book for
its planted trees:

```yaml
scene:
  recipes:
    planted tower:
      kind: tower
      pins: {skin: concrete, planted: true, storeys: 30}
      tree-profiles: {file: ../models/plant-profiles.yaml}
    green city:
      kind: city-clustered-green-10000
      tree-profiles: {file: ../models/plant-profiles.yaml}
```

The file path is relative to the set. `tree-profiles` is also supported by
`city-clustered-night-10000`; other city kinds and unrelated recipes refuse
it. The book must contain every species that the selected tower or city can
choose. An explicitly named missing, invalid, or incomplete book is refused;
when the field is omitted, the existing embedded profiles are used. Its exact
bytes join the recipe and job keys, and an opened job keeps the checked book
for staging and resumption even if the source file later changes or disappears.

Books are limited to 1 MiB and 256 profiles. A seasonal plant with more than
100,000 foliage attachments is refused rather than truncated. Authored
numerical figures are bounded; seasonal plant axes and roots are checked
before they are used. Seeds preserve the existing draw order for unchanged
profiles; changing the structure of a profile can change its seeded result.

Conifer needle lengths may name two measured `length-references`. The reader
checks their source, units, and values against the research catalog. Read or
derived profile values without verified named references remain valid; they
are authored provenance, not verified measured-number counts.

The registry-based maturity report does not list profile-backed species.
Catalogue inventory can measure its profile-backed cards: it includes the
selected profile descriptor and a source identity derived from the card and
the exact checked book bytes, along with geometry counts and limitations.
It does not turn authored profile provenance into measured-number counts.

Numeric profile values must be present and non-null, including every color
channel; zero is a value where the field's range permits it. Seasonal `roots`
and conifer `shoots` blocks may be omitted. If supplied, their numeric fields
must be complete. A seasonal profile may omit fields for its other foliage
form, and a plant without a leader may omit `leader-gain`. Books contain one
YAML document and do not use null values or YAML merge keys. Regular YAML
aliases may be used.

In Go, `manufacture.ReadPlantProfiles(path, bytes)` gives a checked seasonal
book and `book.NewPlant(name)` constructs a species from it. The other
families expose `ReadConiferProfiles`, `ReadFernProfiles`, and
`ReadCactusProfiles`, with corresponding `NewConifer`, `NewFern`, and
`NewCactus` methods. Named methods such as `book.NewValleyOak()` select a
known species without embedding its data in the constructor.
`manufacture.WithSeason` applies a season where supported.

The existing embedded default profiles and `plant-choices.yaml` still exist;
this path prepares species data for a separate models repository but does not
move those defaults or make the shared numerical construction policies external.
