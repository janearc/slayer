# Caprica: twelve proposed perspectives

These twelve jobs share the binary, generated lava drop and generated
Caprica constructor from the earlier `2026-11-26-lava-drop/sets/caprica-2.yaml`.
Only the camera, output framing and exposure/print choices differ. Every
view is a fresh render. The source scene's comments retain the provenance
and uncertainty: in particular, the 3000 K drop temperature, its age and
cloud appearance choices are invented, and its noted Nauenberg radius source
was not shelved in that handoff. Do not describe these as observed worlds.

The proposed sequence is: 01 dayside disc; 02 quarter phase; 03 A-star
night side; 04 terminator limb; 05 convective deck; 06 grazing cloud tops;
07 above-deck oblique; 08 polar horizon; 09 lava drop close; 10 drop oblique;
11 Caprica from near the drop; 12 wide system portrait. Exact positions and
look targets are in `caprica-lookbook.yaml`; all are authored in east,
north, up metres. The original r7 deck, quarter and polar cameras and r4
drop close informed the new placements and exposures; no old image replaces
any requested view. The last two distant views are particularly provisional:
scale and exposure can hide or clip an object, and their compositions require
visual assessment before being called successful.

All jobs use `gpu-native`, ReSTIR off, 800 × 800, four rays per axis, the
existing stars, and individual print/exposure settings. Day and polar deck
prints follow the earlier r7 reference; drop prints follow r4. The night-side
exposure and new camera placements are artistic choices. Per-camera field
of view is explicit in each job. A PNG and untouched linear PFM are sent to
separate frame folders. The use of native GPU, the approximate weather and
the constructor do not establish physical realism. These are provisional
look-book views until each completed frame is inspected at full size.

## Selected views and observed limits

The following frame-folder suffixes under `2026-11-27-caprica-` are
the selected twelve, replacing the initial proposals where noted:

| Slot | Folder suffix | Observed character and limitation |
| --- | --- | --- |
| 01 | `01-dayside-oblique` | Mostly lit textured globe with a dark limb; still centered with dark flecks. |
| 02 | `02-quarter` | Half-lit globe and textured deck; night side is subdued. |
| 03 | `03-night` | Dim blue-green night-biased hemisphere with clouds. |
| 04 | `04-terminator-crop` | Off-center curved day/night boundary; the upper area is mostly black. |
| 05 | `05-deck` | Sculptural whorl and cloud deck along a curved limb. |
| 06 | `06-grazing-night-close-final` | Close blue night limb with pale cloud and subdued ground, deliberately dim. |
| 07 | `07-oblique-north-orbit` | High-angle lit disc and rough terminator; a broad orange area and dark flecks remain. |
| 08 | `08-cloud-skim-final` | Thin golden cloud limb and a featureless near-white point; the point has no resolved detail. |
| 09 | `09-drop-close` | Full lava-drop globe with stylized bright seams. |
| 10 | `10-drop-oblique-surface-crop-final` | Partial close lava horizon; seam centers approach white. |
| 11 | `11-drop-to-caprica-reverse-quarter-final` | Reverse Caprica quarter, not a view of the drop; resembles 02 and has boundary bands. |
| 12 | `12-drop-seam-horizon-final` | Second close lava horizon with converging seams; compositionally similar to 10. Preview assessed; final still awaits inspection. |

The independent visual researcher assessed the preceding previews and the
completed selected images at full size where available. The original 01,
04, 06–08 and 10–12 proposals were not retained as selected: some repeated
other views, and some failed to show the intended subject. In particular, no
successful system-wide portrait was established. The final selection is
twelve camera perspectives, not a claim of twelve strongly distinct subjects
or scientifically validated appearance.

## Reproducing the staged jobs

Each job resolves its input set relative to the job file and directs new
renders into the checkout's `out/` directory. The original gallery
files and submitted settings are historical evidence. The submitted paths
are recorded in the original revision's job files.

Build Slayer from `701507dec` in a separate checkout if matching the
historical renderer matters. The pinned binary's SHA-256 is
`7f8bc25d1e478b87ae3e66b2289d789284b75e7e1e2a8178a63a2585b5def7a7`.
Builds may vary with toolchain. A different hash identifies a different
binary. Configure the Slayer cache, shelf and data according to
[Building Slayer](../../building.md).

From this checkout, use a configured build with the selected job:

```sh
game run slayer -- draw doc/examples/caprica-lookbook/01-dayside-oblique-job.yaml
```

Create the selected output directory under `out/` if needed before
drawing. Once the twelve selected PNGs and PFMs are available, the
[look-book command](../../caprica-lookbook.md) can read the
[portable manifest](../../caprica-lookbook-manifest.json).

## Raw-output incident

The submitted 10 surface-crop final job accidentally wrote its PFM to the
earlier preview folder, overwriting that preview's raw. Its final PNG went
to the correct separate final folder; the final raw was copied there after
the overwrite. The submitted 08 polar-low dim and 11 reverse-quarter dim
preview jobs likewise wrote their PFMs into their respective earlier non-dim
preview folders, overwriting those earlier raw files. The previews' PNGs
remained separate. The current PFMs in those earlier folders belong to the
later dim renders, not the earlier previews; they were also copied into the
matching `-dim-preview` folders beside their PNGs. The earlier raw files
were not recovered. This does not affect provenance of the separate PNGs.
Subsequent jobs use distinct paths for both picture and raw output.
