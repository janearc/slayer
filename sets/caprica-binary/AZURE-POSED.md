# Varied Azure flock study

Job 569 made the lead animal larger but an observer could distinguish only
about ten; one crossed its beak. The sky and horizon were largely flat,
the clouds did not resolve and the other animals repeated their poses.

This first unrendered follow-up moves one animal away from the lead's head
and uses four complete copies of the existing reptile profile book, changing
only the Hatzegopteryx `phase` and `bank`, in radians: glide 0/0, left
0.6/-0.30, right 1.1/0.35 and lift 2.1/-0.2. These are artistic pose
choices, not measured flight dynamics. Bank is local to the animal before
heading. Repeated recipes still share poses. `detailed-plain` omits membrane
paint because the GPU path refuses the painted posed membrane; the detailed
CPU reference retains it. The pose changes and absence of silhouette overlap
remain to be observed.

The background adds and thickens small nearby blue wisps; they are invented
cloud proxies, not a fluid simulation or guaranteed visible cloud detail.
It retains v5's invented photographic key/fill and theatrical enlarged,
elevated sun-cue spheres. Those cues have deliberately different positions,
sizes and colours from the real stellar images. The translated binary itself
is unchanged. These choices do not demonstrate real stellar visibility or
photometrically correct illumination. See `AZURE-FLOCK.md` for prior frames,
provenance and numerical limits.

Job 574 showed visibly different banked wings but cropped the lead animal
severely at the right edge; the lower field was pale and clouds read as
diffuse, not resolved structure. V7 moves the lead farther away and toward
the centre, separates another flyer, and reduces the artistic key, fill and
theatrical cue brightness. The ground and horizon gradient colours are
matched to seek a less abrupt artificial sky boundary; the three small
wisps are thinner with smaller noise scales. These combined artistic changes
are unrendered and do not isolate any cause of the prior washed-out field.
The same four pose books, binary geometry and nonphysical elevated sun cues
remain in use. Full-body framing, twelve readable individuals and cloud
structure require visual observation.

Job 577 contained a whole lead animal but an independent observer could
distinguish only eleven forms, with a second crossing under its wing. Its
lower field was pale and faintly mottled, without recognizable cloud forms.
Two matched one-ray diagnostics retain its camera, animals, lighting, sky,
binary, artistic sun cues and exposure: `azure-medium-giant-only.yaml`
removes all local wisps; `azure-medium-wisps-only.yaml` removes the generated
giant, leaving the local wisps. Their scene, job, pose profiles and outputs
are separately pinned. Comparing both to 577 can indicate which components
contribute to the appearance, but scattering interactions mean their effects
are not necessarily additive. The wisps-only view does not represent an
actual giant atmosphere, and neither diagnostic is a finished composition.

V8 moves the lead to 19 m from the camera and shifts two animals that
projected near it in 577. A pinhole estimate places their origins at roughly
(543,251), (131,174) and (718,258), respectively, in the 960×600 frame;
these are origins rather than silhouette bounds, so they do not establish
full framing or lack of overlap. This is an unrendered framing candidate;
its atmospheric elements are unchanged pending the two medium diagnostics.

The paired visual diagnostics showed the pale lower band in the giant-only
image, while wisps-only lacked that band but had only faint broad patches.
Job 580 showed twelve apparent animals in frame; one small animal nearly
touched the lead's upper wing, and no detailed clouds were evident. V9
retains Azure's generated gas volume and moves that small animal sideways.
Four overlapping, much denser small-scale wispy puffs behind the flock are
invented blue cloud staging, not generated planetary weather or physically
validated cloud. Their contrast, depth, any occlusion of stars or animals,
and the effect of the giant medium remain unverified until rendered.

The independent 589 cloud-deck study showed distinct but soft forms. V10
keeps Azure's generated gas volume, the twelve V9 animals and their authored
poses, and the fictional elevated sun-disc cues; it replaces V9's local
wisps with a bounded prescribed cloud deck behind the flock, using 589's
coverage, extinction, noise scale and seed. Box size, position, height and
lighting differ, so the standalone image does not predict this one. The
deck is an invented local proxy, not simulated giant weather; portions extend
above Azure's nominal gas boundary. Flock separation, cloud appearance and
visibility of both cues remain unverified until the new frame is inspected.

Job 590 showed twelve apparent animals and two cues, with soft clouds cut
off at an unnaturally straight horizontal line near image y=211. A similar
line was already visible in job 569 before the cloud deck; the v10 cloud
base also lies at camera height, so the image alone cannot isolate the sky,
giant or deck boundary. V11 lowers the same invented cloud box base from
20 to -15 m, deepens it to 110 m and applies a 25 m smooth density fade on
all six faces. The giant gas and artistic sky/sun cues remain in place. The
feathered local deck remains an artistic proxy, with some of it outside the
giant's nominal gas boundary. Its effect on the hard horizon and cloud
quality must be observed; no finished scene is claimed.
