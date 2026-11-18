# A visitor beside the Larson

This is the bundled **Larson + visitor** study in Ferruccio. Choose it from the
study list and render its preview; Ferruccio copies the YAML and CC0 mannequin
resources into an independent working folder. The scene combines Slayer's
existing `mcrn-larson-landed` manufacture with Oakland's static glTF import.

The eight-part mannequin and its binary buffer are original CC0 fixtures; their
rights declaration is in `LICENSE.txt` and the set. See the
[Ferruccio and Oakland integration notes](../../FERRUCCIO-OAKLAND.md).

The mannequin's two-metre height is a chosen geometric reference. It is
not a scan or a claim about a particular person's anatomy. Its matte
colour, the landing pad and lighting are authored choices. The Larson's
geometry and materials come from the existing manufactory unchanged.

The set places the figure at east 7, north 1, up 0 metres. Change `at`
or `heading` under `scene.world.imports.visitor` to place it elsewhere;
the Larson's placement is under `scene.set.things.larson`. The camera uses
the same map coordinates through `location.at` and `orientation.look_at`.

Render again to see the actual scene update; Ferruccio keeps the previous
completed image while Slayer renders. Camera changes are undoable YAML edits.
USD, textures, skinning and animation are not implemented.
