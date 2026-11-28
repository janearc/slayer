# An animal from a profile book

an animal recipe reads a named species from a checked book. the path is
relative to the set. see [the seal set](../sets/bioforms/bay-seal.yaml) for
a complete recipe with a placement and camera. the recipe and its source
bytes are retained with the job,
so changing or removing the file after reading cannot change an already
opened job. A changed book changes the recipe and render keys.

```yaml
scene:
  recipes:
    sardines:
      animal:
        family: fish
        file: ../models/fish.yaml
        species: pacific-sardine
        pose: swim
        count: 500
    heron:
      animal:
        family: bird
        file: ../models/birds.yaml
        species: grey-heron
        pose: standing
    allosaurus:
      animal:
        family: theropod
        file: ../models/allosaurus.yaml
        species: allosaurus
        seconds: 0.5
    seal:
      animal:
        family: pinniped
        file: ../models/pinnipeds.yaml
        species: harbor-seal
        pose: swimming
    reptile:
      animal:
        family: reptile
        file: ../models/reptile-profiles.yaml
        species: purussaurus
    snake:
      animal:
        family: snake
        file: ../models/snake-profiles.yaml
        species: bitis-gabonica
    squid:
      animal:
        family: squid
        file: ../models/squid.yaml
        species: humboldt-squid
        phase: 1.1
    orca:
      animal:
        family: whale
        file: ../models/whale-profiles.yaml
        species: orca
        pose: swim
```

`family`, `file` and `species` are required. use `file: "@builtin"` to
select the book embedded for that family. Fish accept `swim`, `glide`
and `bones`; the default is swim. A positive `count` requests a school,
where the selected profile supplies school parameters; a school cannot be
combined with glide or bones. Birds accept `standing`, `gliding`, `bones`
and `outline`; the default is standing. Theropods use a finite `seconds`
value instead of pose or count. Reptile studies take their pose controls
from the checked profile. Pinniped poses are checked against the
selected anatomy. Squid accept an optional `phase` in radians, from -100000
to 100000; omitted means zero. It selects a fixed illustrative pose and is
not a time, gait or speed. Squid recipes reject pose, count, seconds, study
and representation; other families reject phase. Invalid books, absent species
and unsupported combinations are refused; there is no fallback to an embedded
kind.

Snake recipes make a static specimen and reject pose, count, time, phase,
study and representation. Each checked profile supplies its own length, body
stations, eye details and pattern. Measurements, sources and modelling choices
are described in the profile book; fossil soft tissues are speculative.

sauropod profiles also make static specimens. see the
[Brachiosaurus set](../sets/bioforms/sauropod-review/brachiosaurus-altithorax.yaml)
for the recipe and its separate full-body and head views.

Whales accept the time-dependent `swim` pose (the default) or a static
`spy-hop`; a rest skeleton uses `representation: bones` without a pose.
The current orca profile controls its authored body stations, swim-cycle
length, skin sheen and linear reflectances for its markings. The body,
binding and skeleton use those stations. Fins, flukes, face, marking
placement and skeletal details remain the existing orca study; it is not a
universal whale scaffold.

Pinniped recipes can select a rest `representation: bones` for a harbor seal
or walrus, and `representation: volumes` for a walrus. A skeleton takes no
pose; walrus volumes accept an omitted pose or `pose: lying`. The default
representation is `skin`. The selected profile drives each skeleton and the
walrus volume rig.

The harbor and monk seal loft profiles also hold their body outline stations
and length. The loft uses the shared rest skeleton and flippers: it requires
the existing station positions and permits each station's half-height and
half-width to vary by at most 10%, and its centre by at most 10% of the
reference half-height. The reader checks dimensions and the constructor checks
the closed, posed trunk and its bone bindings. These limits support variations
on the existing scaffolds; they do not supply a new species-specific skeleton,
pelt or flipper anatomy.

The walrus profile holds its own body outline stations. Its head, tail,
station positions and section exponents remain fixed to the existing rig;
the intervening trunk widths and heights may vary by up to 5%. Construction
checks the fixed bones against the changed outline and checks the field grid.
The tusks, limb dimensions, pose and hide remain part of the walrus scaffold.

A profile may also declare a fixed drawn-rig `study`. For example,
`study: detailed-wet`, `study: detailed-dry` and `study: bay` select the
harbor seal's existing studies. A study cannot be combined with `pose` or
`representation`. These drawn-rig studies use their own rig and profile look
settings; they do not use the loft skeleton counts in the same profile.
Available studies and their provenance are stated in the profile book.

Reptile profiles select the existing pliosaur, crocodylian or pterosaur
scaffold. `study: baseline` or `continuous` selects a pliosaur study;
`study: composite`, `cage` or `density` selects a pterosaur study. Omitting
`study` selects the original composite, while the crocodylian has one study.
These profiles carry the migrated body and limb landmarks, but several
species-specific details remain fixed to each study. A reptile recipe does
not accept a separate pose, count, time or representation.

For the Hatzegopteryx `detailed` and `detailed-plain` studies, a profile's
existing `phase` chooses an authored fixed wingbeat phase in radians; an
optional `bank` chooses a fixed rotation in radians about the animal's
forward axis, from -π/2 to π/2. Omission means zero bank. Use separate
profile books and recipes to place different phases and banks; each pose is
an artistic choice, not a flight dynamics model. The bank is applied before
the instance's scene heading. The painted membrane remains CPU-only.

The selected species name is data, not the behavior category used by the
scene reader. A species named `fire` therefore cannot become a fire by
accident. The book readers bound inputs and construction counts before
making geometry. The species' source and modelling choices remain with its
profile.

The current fish builder covers the ray-finned profiles in `fish.yaml`; it
is not a shark or a universal fish model. The bird builder uses the current
bird profiles. The initial theropod scaffold is the authored Allosaurus
study with profile-driven stations and materials; it is not a general body
for all dinosaurs. Pinniped profiles distinguish seal and walrus skeleton,
pose and skin construction. Additional anatomy needs its own supported
rules and profiles, not a renamed existing animal. Squid profiles select
size, body and appendage proportions, and coat for a shared illustrative
scaffold and pulse; changing the species name does not establish its anatomy.
