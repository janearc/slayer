# Relocated scene context

Slayer locates sources.yaml beside an input or in its ancestors. Configuring
SLAYER_SHELF identifies the dataset shelf separately; it does not replace the
manifest. Moving a scene into private example or regression evidence storage
must preserve its original manifest context.

CopySourceManifest searches from the original input directory upward and copies
only the nearest sources.yaml into an already-created private destination. It
preserves exact bytes, accepts an identical existing copy, and refuses an
existing different copy. Manifests must be regular files, not symlinks, and at
most 8 MiB. No manifest found is allowed. Filesystem and copy failures propagate.

This helper copies metadata only. It never copies datasets or changes the
original input directory. Its callers must use new private destination folders;
ordinary user directories are not staging destinations.
