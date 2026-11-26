# Jobs in an editor buffer

A job open in an editor may differ from its file. The editor needs to admit
the buffer through the same checks as `Read` while resolving relative paths
against the original job's directory. It must not save the buffer merely to
inspect the scene. The original path still names errors and `File`.

`ReadSource(path, source)` admits the supplied bytes at that location. `Read`
reads the file once and delegates to it. Import snapshots and inputs retain
normal job semantics. A buffer with invalid YAML or an invalid job is refused;
changing the job's referenced set or its contents changes its picture key.
Tests cover relative paths, changed and invalid buffers, and disk preservation.

`StillFrom(path, source)` admits an editor's set buffer as a checked still.
It resolves references against the original set location, retains the parsed
set and fixes the buffer and decoded import, model-profile and composition
digests to that snapshot. Other sources follow existing renderer behavior.
The session stages that retained set even when the saved file later changes.
It does not write the set. `StillOf` keeps its existing file-based behavior.

Tests change the saved set after constructing the still and compare retained
geometry and picture identity. They also check invalid source and relative
references, so a stale disk read cannot pass as the editor's buffer.
