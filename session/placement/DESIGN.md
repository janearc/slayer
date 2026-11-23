# Manufactured space selection

SpaceGeometry builds manufactured placements through the Make and SpaceAt
primitives shared by staging.craft. Inherited complexity, scene/object seed
mixing and the placement transform are identical to rendering. It starts no
renderer and exposes no mutable geometry. Hit intersects native shape.Hitter
surfaces in their own placement space and returns the nearest eligible name.
A bounding box alone never selects an object.

The world half is built through the same pure stage.Add used by the session.
Its native surfaces participate as blockers with no selectable identity.
This includes the playground pad. Volumes and participating media are outside
the surface selection contract; opacity is not integrated along rays.

Hidden surfaces do not block. All returned surfaces must support exact
intersection. Explicit scene seeds are required. Unsupported set features,
terrain, recipes and burning manufactured objects are refused.

Ferruccio picking.New joins checked draft Objects identities with native set
reading, camera Shots and film.Wide. It builds geometry once per source
snapshot. Pick uses film.Look at the actual image aspect; normalized positions
resolve to authored film pixels. Drag intersects native rays with shape.Plane
translated to the selected object height. Parallel, behind-camera, unknown-ID
and nonfinite gestures fail. Valid movement returns metres east/north.

Selection never edits source and owns no undo. The editor caches the immutable
Scene per source generation and rebuilds it off the UI thread when source
changes. A geometry snapshot is valid only for the matching preview source.
