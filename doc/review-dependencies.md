# Review dependencies

Slayer now uses the published Puzzlemouth v0.1.4. Celestial and
celestial-data still use exact private review candidates supplied by the
remediation bundle. `module-candidates.json`, beside the bundles, records
the candidates' source commits and archive hashes.

Puzzlemouth includes schema validation and contextual source excerpts.
Grissom also needs Celestial's FITS and Grissom packages and
celestial-data packages absent from the older published dependency set.
The parent workspace previously supplied all three implicitly.

For a standalone checkout, point Go at the supplied file proxy. Adjust only
the first path below to the unpacked review directory. The proxy contains
the private candidates; released dependencies use the public proxy.

```sh
export REVIEW_DIR=/path/to/remediation-2026-11-04
export GOPROXY="file://${REVIEW_DIR}/go-proxy,https://proxy.golang.org"
export GONOSUMDB=github.com/janearc/celestial,github.com/janearc/celestial-data
export GOWORK=off
```

Use Go 1.27.3, as required by Puzzlemouth v0.1.4. Configure `SLAYER_SHELF` as
described in the [Game workflow](developing.md), then run `game build` and `game check`
from the Slayer root. Source verification precedes the build. Rendering and
GPU checks follow the machine owner's render policy.

The checksum exception applies only to the two private modules. Their
canonical module hashes are retained in `go.sum`; the supplied manifest and
archive hashes provide the review provenance. No replace directives or
workspace are required for this setup.

Before a Slayer release, publish the dependency revisions through their
owners' release process and update the pins to released versions. Preparing
these archives does not publish a module, create a release tag or push a
branch. The supplied bundles preserve the local source for review.
