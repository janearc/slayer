# The boats

Three designs of boat, each made from a seed so that no two are alike: a
sampan, a steel inshore fishing boat and a passenger ferry. They are kinds of the manufactory (manufacture/boat.go),
with their figures in manufacture/boats.yaml.

## The kinds

Each design is two kinds, by its state:

| design | under way | at anchor |
| --- | --- | --- |
| a small wooden working boat, 5 to 8 m | `sampan` | `sampan-at-anchor` |
| a steel inshore fishing boat, 15 to 25 m | `fishing-boat` | `fishing-boat-at-anchor` |
| a passenger ferry, 20 to 38 m | `ferry` | `ferry-at-anchor` |

A boat under way shows a red light on its port side, a green one to
starboard and a white one at its stern. A power vessel, which is the fishing
boat, the ferry and a sampan with an outboard motor, adds a white light at
its masthead. A boat at anchor shows one white light all round. A sampan's
lantern, a fishing boat's floodlights and a ferry's cabin lamps burn in both
states.

Every light is a real lamp of the arc the regulations give it, with
a lens that glows where the light is and a screen behind the lens.

The four complexities are the usual ones: `hull` is the hull and the house,
`clumps` adds the planking, the bulwark, the mast, the posts and the gear
that stands on the deck, `quick` the windows, the seats, the tyres, the
cargo and the painted letters, `demo` the rest. A part that repeats, a
window or a tyre or a plank, is stored once and placed many times.

## What is on them

Every piece is a real part with its own shape, not a texture. At `demo` a
sampan has about 270 to 410 parts, a fishing boat about 400 to 440 and a
ferry about 600 to 810, counted over six seeds; at `hull` each has a few
dozen. The counts differ from seed to seed because the clutter is drawn from
the seed: how many crates, floats, buckets and coils there are, and where.

The sampan has its planks counted along each side, its ribs, a floor of
boards, thwarts, thole pins, a canopy of slabs laid with bamboo laths on bent
hoops with the tarpaulin lashed over them and tied down to the gunwale, a
sculling oar on a pin or an outboard with a tiller and a fuel can, a bamboo
pole, a stove and a kettle, coils, a bucket, crates, tyres or a rope along
its side, and a lantern in a frame on a hook.

The fishing boat has plates with a rubbing strake and a boot topping, a
bulwark with stanchions, a cap rail and freeing ports with rust running from
them, an anchor in its hawse pipe with a windlass and chain, bollards and
cleats, hatches with coamings, a wheelhouse with framed windows, a door, a
ladder and a railing round its roof.

It also has a radar, whip aerials, a horn, a
searchlight, a funnel, a mast with booms, shrouds and stays, a net drum,
piled nets, floats, crates, coils and buckets, tyres, lifebuoys and a liferaft,
a rudder and a propeller, and floodlights on arms.

The ferry has an open-sided cabin, its roof on pillars over sills and
headers, so that the rows of seats and the lit ceiling show through the
windows; a wheelhouse over its front where the seed gives one, railings round
every deck, a gate in the starboard rail, lifebuoys, a funnel, and a name on a
board on each side of the cabin.

Faded paint and rust are the boat's own: the paint fades toward rust by the
seed's wear, and the rust that runs from a scupper or a fitting is a thin
part of its own. A scene may add weathering of its own to a boat with
`world.weathering`; that is drawn by the CPU tracer only, and the GPU leaves
it out.

## Letters

Each boat carries a four-digit number on each bow, drawn from its seed, and
a name on its stern, in painted letters: each letter is a solid made by the
typography package from a face of go's own, standing a little out of the
hull.

The face and the names are in the `lettering` block of boats.yaml. That
face has the Latin letters and digits and nothing else. The block has a place
for Chinese, `lettering.chinese`, which is empty: when a face of Simplified
Chinese is available, name it there with the names to set in it, and the boats
are lettered in it. No font is kept in the boats' data.

## Its space

Metres, y up, the waterline at y nought, the centreline on x nought, the bow
toward minus z and starboard toward plus x, the middle of the length at z
nought. That is the Alfa's frame. A boat is set on the sea by putting its
origin at the sea's height.

## Where the figures came from

Every range in boats.yaml says in its basis whether it was `recalled`, taken
from memory and not looked up, or `picked`, chosen to look right. Nothing in
the file is measured. The sampan's and the fishing boat's lengths, and the
arcs of the lights, are recalled; every other figure is picked.
