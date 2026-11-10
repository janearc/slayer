# OWL, the one that was never built

OWL was an ESO study for a 100 m optical telescope. It still deserves a
place in a scenery factory. It observes in the open, with a tapering space
frame and a shelter that moves away, rather than another enormous dome.

The new factory kinds are owl and owl-shelter. They are an interpretation
of the [2005 Blue Book overview](https://www.eso.org/sci/facilities/eelt/owl/Blue_Book/2_Overview.pdf) and its
[enclosure chapter](https://www.eso.org/sci/facilities/eelt/owl/Blue_Book/11_Enclosure_and_infrastructures.pdf). They are not an engineering reconstruction.

## Pick one version and say which

This recipe selects the overview's 100 m primary, 25.6 m flat secondary,
3,048 primary panels and 216 secondary panels. Its spherical primary has
the overview's 250 m curvature radius. Tables 2-2 and 2-3 establish those
roles; the panel layouts here remain authored.

The older [FAQ](https://www.eso.org/sci/facilities/eelt/owl/FAQs.html) says 3,042 primary panels. A separate design-parameter
summary also differs from the overview in some optical dimensions.
Those figures are not silently averaged or mixed into this version.
The source keys, physical PDF page numbers and excerpts live in the ledger.

The primary's flat-to-flat panel dimensions are not claimed from this
layout. Its authored sixfold annulus is scaled to the nominal outer extent.
Both reflective skins have closed backing that follows their own figures,
rather than a wedge reaching back to a common flat slab.

## The structure is part of the silhouette

The model has a wide azimuth rail, two altitude wheels, raked supports,
a deep triangulated mirror cell and a tapering upper space frame. Closed
joint blocks cover the inset ends of structural members.

The raked feet sit inside their plinths. Each altitude axle terminates at
the primary-plane frame joint, and each corrector support connects its
mirror backing to a built upper-frame station. These contacts are part of
the scenery construction, not a load or engineering analysis.

The secondary is a panel assembly with a central opening. Four separate
corrector roles sit inside the upper frame. Their sizes, spacing, supports
and poses are picked scenery; this is not a calibrated six-mirror train.

The paired shelter has four parked arch modules, bogies and a short length
of rail. The real proposal's roll-away principle is the reference. The
chosen geometry does not simulate its hinges, drives or deployment.

OWL keeps closed component meshes in ordinary instance groups. Exact local
geometry is shared within one assembly, including repeated joint blocks. No
component can exceed the existing 100,000-face flattening limit; the complete
group still consumes stored-geometry and instance budgets at staging.

The segmented optical inventories remain physical geometry at every detail.

Everything uses the existing mesh and instance renderer route.
There is no new GPU primitive or special optical intersection path.

## Place the pieces independently

Use kind: owl for the instrument and kind: owl-shelter for its parked
enclosure. Give both an explicit heading so their alignment does not depend
on a scene seed. Keep the parked shelter outside the instrument's envelope.

The authored instrument is north-facing at 55 degrees before placement.
Its enclosure is a separate object, so a catalogue view can show the
instrument without shrinking it into a much larger empty site boundary.

## Implementation status

This is code prepared for the next review break. The new source snapshots
are downloaded and pinned, but normal source verification, geometry checks,
GPU comparisons, compatibility capture and specimen renders are deferred.
No new appearance reference has been recorded from this draft.

The four details retain the same panel inventories, optical figure and
physical envelopes. Circular supports and shelter arches vary their
tessellation. Those are implementation claims awaiting the planned checks.
