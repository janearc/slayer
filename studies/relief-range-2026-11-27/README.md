# Experimental geometric range: first study

Experimental build `ac066195b` adds a fixed body-space, finite mountain
belt to the old diagnostic displaced mesh. It applies the made-up 2,400 km
length and 900 km width to land at the previously measured target direction.
For this diagnostic only, the existing `ridge_km` field chooses belt height:
0 or an exaggerated 30 km. The field, mesh wiring and numbers are not a
supported constructor or a geological claim. Camera, seed, lights and other
inputs match; the global mesh has 256 by 128 divisions. The wider camera
stands approximately 2,000 km out and 1,000 km sideways from the target.

The original raws and images from jobs 351–352 are in the November 27
render archive. The images are too dark to assess. Jobs 353–354 develop those raws with identical -3 stops, knee 0.8,
top 4 in the matching `-print/` directories. Independent inspection finds
that 30 km adds a conspicuous black polygonal protrusion at the upper-left
edge, with fuzzy painted land and no identifiable mountains or discrete
cast shadows. This is not a successful visual range result. The cause of
the edge's appearance is not established by this pair.

## Narrower segmented range

Build `31f833239` narrows the belt to 450 km, applies picked uneven
along-belt peak modulation and samples it with a 512 by 256 global mesh.
Jobs 357–358 use matched camera/light/seed at 0 and an exaggerated 60 km
height. The November 27 render archive contains the original frames and
raws for the 0 and 60 km peaks and their matching -3-stop developments. The 60 km print shows a much larger
black angular form over the mottled terrain; independent inspection did
not identify a coherent range or a discrete shadow. Changing shape, height
and mesh density together does not isolate the cause. This candidate is
not presentation quality and is not proposed for integration.

Jobs 362–363 rendered the 60 km scene at 300 by 300 and one ray with CPU
and native GPU respectively. The archive retains the raw for each renderer. Independent inspection saw a
similar black angular form on both, with sharper jagged tips on CPU. This
does not establish pixel parity or exclude a shared cause.

Experimental pixel probe build `ef3d1bc0c` traced the 300-pixel GPU job's
camera rays against the staged mesh. At (130,110), within the black area,
it hit the mesh and measured normal cosine 0.021642668 to the primary
light centre; a ray offset 100 scene units along that normal was blocked
by the same mesh. At adjacent (190,110), the mesh hit had cosine
0.100407112 and its offset ray was not blocked by that mesh. The probe
does not include other objects, the emitter disc or media, and its offset
is not the renderer's. The two samples support mesh self-occlusion as a
plausible contributor, not an explanation of every dark pixel.

## Daylight target

A staged-mesh search with a primary-light radial cosine of 0.35–0.55
selected vertex 28321 at (-1846939.786, -5892503.017, -1846939.449)
in world coordinates, radial cosine 0.360999. Diagnostic build
`fe5b8e2f0` places the segmented 450 km belt there and compares artistic
heights 0 and 15 km using the same 512 by 256 mesh. The camera's centre
pixel hit the mesh about 5.8 km from the selected point; its normal cosine
to the primary was 0.342205 and the mesh-only ray with a 100-unit offset
was unblocked. That probe does not include other geometry, the emitter disc
or media.

Jobs 379–380 drew the two views; 381–382 developed their retained raw
identically with knee 0.8, top 4, -3 stops. The archive contains both daylight originals and their developed prints. Independent inspection found nearly the same
mottled blue-grey and green land with slight changes but no recognizable
range, cast shadow or silhouette. This does not establish compelling
relief. The large black shape in the prior grazing view is absent here;
camera, target, light angle and height changed, so this is not a causal
comparison of that shape.
