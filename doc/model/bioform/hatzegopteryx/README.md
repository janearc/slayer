# Hatzegopteryx

The optional `detailed-plain` variant uses the same geometry and pose as
`detailed` but omits the authored streaked membrane paint while retaining
the mottled hide.
This is an artistic compromise using the GPU's existing support for unpainted
posed meshes; it does not implement rest-space texture lookup. Its renderer
compatibility and appearance have not yet been established.

The opt-in `detailed` reptile study restores the detailed pterosaur study at
 ebdc4f98fb8c01e76946436030256dad4d155297, with its wing rig from
4266c3c7d. The profile phase selects a fixed wingbeat phase in
radians. The 2026-11-20 report and pictures document a more detailed head,
jointed limbs, two membrane panels per wing and geometric body fibres;
membrane stripes and separate-shell seams remain visible. Pose, proportions,
colours, crest and soft tissue are hypotheses; this is not an approved
anatomical reconstruction. The rig lengths and inferred membrane attachments
are not direct measurements of Hatzegopteryx. See
[SOURCES.md](SOURCES.md) for anatomical references.
The adapted study's current renderer compatibility and appearance remain to
be established by new images.

An additional opt-in `soaring` study is under construction. It uses a
gliding outline with a separate tapered rostrum, a stout differentiated
neck, open wing controls and thin membrane, and trailing jointed legs.
These exact shapes, colours, attachments and pose are invented. The
comparisons in [SOURCES.md](SOURCES.md) motivate the long toothless
rostrum, long hindlimbs and stout neck; they do not supply a complete
Hatzegopteryx outline. The separate shells can leave seams, the old wing
surface has known defects, and no visual result or anatomical approval is
claimed for this study. Its inspection scene is
`doc/examples/hatz-soaring/scene.yaml`.

An authored giant azhdarchid pterosaur of about 11 m wingspan, after
Hatzegopteryx thambema. It is an invented individual, not a scan and not a
reconstruction of a known animal: the holotype is a piece of rear skull and
an incomplete humerus. It is not an anatomically convincing Hatzegopteryx in
any of its forms below, and no version of it has been approved as one.

## What ships

- `hatz-gargoyle`: a small, static stone ornament made from the final
  motion study's rest shape, at quarter scale, in a neutral stone material.
  Set: `sets/bioforms/hatz-gargoyle-study.yaml`.
- The gargoyle creature `hatzegopteryx` in
  `manufacture/cities/gargoyle-creatures.yaml`: a crouched animal on the
  slab with the wings raised and swept back and the long beak out over the
  edge, the beak's tip the spout. `manufacture/gargoyle_hatz.go` folds the
  wings half way and curls them up; the hinge, radius and turn there are
  picked by eye against a render.
- `HatzDensityStudy`, formerly `HatzFeedbackFour`: the motion study used by
  the gargoyle. Its builder lives in
  `manufacture/animals/vertebrates/reptiles/hatzegopteryx/density.go`. Earlier
  uncalled studies and their tests remain in Git history.
- The profile-selected composite study: an earlier standing and flying
  reconstruction with a fold from 0 to 1. It rejects a fold outside that
  range and any time that is not finite. Its constructor lives at
  `manufacture/animals/vertebrates/reptiles/hatzegopteryx/composite.go`.

[SOURCES.md](SOURCES.md) lists each source, what it supports and what it does
not.

## The gargoyle

The kind `hatz-gargoyle` can be drawn directly in a set or put into a
superblock's roof-part mix. Native closed mesh faces are carried into the
roof berth geometry with local placement. Unsupported native shapes and
oversized extractions are refused. Placement uses the actual mesh envelope
and the berth floor. The model is static and registers no animal scene and
no individual render job.

An immutable rest prototype is reused so the field is not rebuilt for every
berth, but the face budgets count each placed native part separately, so a
dense repeated deployment can be refused even where geometry is shared.

Its first plinth view reads as an aircraft-like spread-wing shape. It is a
placement and geometry test, not a finished perched sculpture. A good
gargoyle needs a deliberately perched silhouette, leg and wing poses, facial
detail and stone texture. Architectural scale, roof-edge placement,
structure, drainage and weathering are not established.

## How the last motion study is put together

`HatzDensityStudy` is one closed skin from an additive density field of
masses for trunk, neck, a deeper skull tapering to a long toothless beak,
longer hindlimb segments and thin wing masses, with eyes and a mouth marked
on the surface as hypotheses. The rest mesh keeps its indices and coordinates
as it moves. Each wing is displaced by a lattice of cubic controls placed
along its span, with different phase offsets.

The skin has 9,162 vertices. Over 13 sampled phases the outer wing tip's
height spans -1.283 to +1.294 m; per-edge lengths against the undeformed rest
mesh range from 0.7200 to 1.3884; absolute signed volume stays within 7.6458
to 7.6473 m3; and the cycle matches at its endpoints. Those are sampled
numbers, and the motion is authored. It was not compared with a gait.

## Known faults and unfinished parts

- The animal reads as a seal with wings. The beak is short and blunt, the head
  and neck are poorly differentiated, the legs are thick and stumpy with no
  visible joints or feet, and the wing is a broad rounded slab with no
  articulated shoulder, elbow, wrist or wing finger and no thin trailing
  edge. The eye and mouth are surface marks.
- The skin has not been checked for self-intersection or collision. Nearly
  constant global volume does not rule out local distortion.
- No aerodynamics, measured tissue, tension, muscle or physical folding is
  modelled. Directional stiffness or anisotropy, where tried, is an
  authored hypothesis from the Jeholopterus fibres.
- The studies before the last used separate shells for body and wings. They
  are not continuous skins, and their wing attachment is not exact.
- Further work needs a thin membrane anchored to an articulated wing, with
  explicit anisotropy as a stated hypothesis, joint and limb anatomy,
  collision checks and a comparison to source skeletons.

## Earlier builds

Three lines of study were made, each rejected in turn. Their code was
dropped, except the final motion study and the earlier study function above.

The first line, five builds, was found on audit to have carried animal
anatomy, controls and much code from one build to the next, so it does not
count as five independent starts. What it measured:

- Arm segments of 1.35, 1.45 and 2.85 m (shoulder to elbow, elbow to wrist,
  wrist to tip) held constant over eleven fold settings, with the folded
  wrist at 0.10 m so a 0.10 m hand sphere touches the ground. These are
  authored, not measured.
- A bone-length-constant elbow still let the leading edge buckle backward and
  the membrane cross itself, until the elbow was placed ahead of the span.
- One wing's one-sided area: 5.761487 m2 open, 3.343915 half folded and
  0.913791 folded in the third build, a shrink of 84 per cent. Pleats added
  to cure it overcorrected: 5.474277, 6.219944 and 12.947012 m2. Choosing the
  pleat height by search held the area at 5.474277 m2 in all three states,
  but 121 non-coplanar triangle crossings appeared at full fold, against none
  at open and half.

  Matching area is a numerical proxy, not tissue
  conservation, and the pleated sheet is not a usable skin.

The second line, a clean-slate series, built an implicit core, circular-arc
wing chords and then an indexed ring skin. Candidate 1 failed construction at
six of thirteen quarter-second samples because of inconsistent winding near
the beak, put the hind feet 0.04 to 0.05 m above the ground, and lost 48.5 per
cent of wing area on folding (4.305237, 4.517236 and 2.215994 m2).

Candidate 2 built at all thirteen samples, held arm segments of 1.35, 1.55 and 2.35 m,
and put the ground forehand 2 mm below the floor, but reused the earlier wing
rig and was rejected.

The third line, the motion studies, began from a complaint that an earlier
preview showed very little motion. In order:

1. Four-link wing chains with a relaxed triangular sheet. The tip height
   spanned -2.408 to +2.848 m, links of 0.85, 1.05, 1.15 and 1.45 m stayed
   fixed, and the nearest membrane vertex was up to 0.1281 m from a joint, so
   attachment was not exact.
2. A faceted cage body with a centroid-refined membrane. Each joint sat at the
   exact midpoint of its shell's boundary vertices, with error under 1e-9 m,
   but edge lengths ranged 0.4313 to 4.0696 times rest and the planned
   hindlimbs were not built. Tip height -2.184 to +2.514 m.

2B. A body from three polygon silhouettes with circular-arc wings: edge ratio
   0.9802 to 1.0198 and tip height -2.068 to +2.093 m, but a stiff blade with
   no hindlimbs.
3. One connected closed skin of 8,148 vertices from blended ellipsoids, posed
   by a spatial warp. Tip height -1.709 to +1.640 m and edge ratio 0.7503 to
   1.3424. The wing was a thick slab, with no eyes or mouth.
4. The shipped one, above.

What carried forward: motion has to be checked by eye and by number; a
boundary that is close is not attached; per-edge change shows defects that
total volume hides; and anatomy has to be reviewed apart from the maths of the
motion.
