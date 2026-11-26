# Region identity

A manifest names the snapshot files, executable digest, complete frame,
rectangle, renderer, rays per side, camera, scene time and the digests of
effective rendering and development settings, plus the explicit scrooge detail
mode. `Digest` validates those fields and hashes a canonical representation.

Input order does not affect the key.
Changing a named input or a setting changes the identity, subject to the
usual properties of SHA-256.

The caller must explicitly assert that the input list is complete. This is
an assertion, not a proof: this package does not discover dependencies or
read the files, verify the running executable, enforce snapshot immutability,
or check that the renderer used these settings. The caller must hash actual
bytes, bind the effective settings to the manifest, and keep the snapshot
stable while it is used. An absent assertion is refused.
