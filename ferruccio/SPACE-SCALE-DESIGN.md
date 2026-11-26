# Uniform size for explicit space placements

The current set schema has no general object scale. A control that writes an
invented field would not change a render. This proposal adds a dimensionless,
positive uniform scale to an explicitly placed `scene.set.things` entry in a
space set, with omission meaning one. It is a geometric transform around the
object's own origin. It does not regenerate a fish's biology or a plant's
growth. Existing unscaled scenes retain their exact meaning.

The set reader and v2 schema must agree on limits and refuse unsupported
placements instead of ignoring the value. Slayer's staged craft, the native
working display, exact picking, source edits, history and saved/reopened jobs
must consume the same value. Attached lights need an explicit rule for their
position, exclusion distance and intensity.

Part-spin, relative and orbital
placements, ground anchoring, stand exclusions, generated stand members and
deformations must either be proven correct or refused by name. A parent
placement's origin stays fixed while its local geometry is scaled.

The current control is limited to the supported form below. Tests cover default
and invalid values, comments and undo, profile fish and plants, picking,
actual staged geometry, attached lights, exclusions and save/reopen. The
combined fish, fern and city test and queued views are recorded under
`ferruccio-evidence/2026-11-25-authoring-acceptance-v2/`; native-window
interaction and the remaining checkpoint 3 scope are still open.

The first supported form is an explicit east/north space placement in a v2
set. Scale is 0.01 through 100, inclusive, and is applied to the native
subspace; an omitted value leaves the existing zero-value transform unchanged.
The origin and heading do not move. An attached light's position and exclusion
distance follow the transform, while its intensity and angles stay authored.

Part-spin and non-explicit placements are refused until their pivot and
relative semantics are tested. Deforming construction is refused by the
stager when scale is present. Ground and stand scaling remain unavailable.
