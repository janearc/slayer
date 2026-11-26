# Reference light

This package checks an identified reference before it is used. Its format is a
single file: the eight bytes `REGREF01`, a 32-byte opaque identity digest, a
32-byte SHA-256 digest of the payload, two little-endian uint32 dimensions,
then width times height RGB triples of little-endian IEEE float32 values in
top-to-bottom row order. The digest covers the payload bytes, not the header.

All RGB values must be finite; there is no upper light limit. Coverage and
depth are not recorded.

The caller supplies the expected identity and dimensions and a maximum file
size. It must derive identity from the complete build, render settings and
transitive input content; this package cannot establish that provenance. It
rejects a different identity or dimensions, extra or missing bytes, invalid
light and a mismatched content digest. A successful read is not proof of
reference convergence.

Publish validates the bytes, writes a temporary file in the same directory,
and links it into the destination atomically without replacement. An existing
destination, even a corrupt one, is left untouched and reported. Unsupported
hard links are an error. The temporary name is removed on return. This gives
atomic visibility, not a promise that data has been flushed to durable storage.
