# The bundled render queue

Ferruccio chooses a queue from `--azazel-queue`, `FERRUCCIO_AZAZEL_QUEUE`, then
**Settings**. If all are empty it looks for an existing shared Azazel queue.
If it finds none, the packaged app uses
`~/Library/Application Support/Ferruccio/Queue`. The unpackaged command and
proof require an available queue and worker.

The app initializes only its private fallback, when the first Preview or
Render fully needs it. Azazel creates the database in a sibling staging
directory. Ferruccio installs that directory with an exclusive rename.
Concurrent installs use the winning complete queue. A partial or conflicting
incumbent is left in place and reported by the render action.

The packaged client can start dispatch for the selected queue after Azazel
acknowledges a render. It leaves active or paused schedulers alone. **Stop**
requests cancellation of the current job through Azazel. The worker can
finish other jobs after its window closes. See
[worker lifecycle](../../../internal/preview/WORKER-LIFECYCLE.md) for logs
and dispatch details.

## Paths

For settings-backed paths, a nonempty CLI flag takes precedence over the
corresponding environment variable, then the saved setting. The file is
`~/.config/ferruccio/config`. The bundle launcher supplies the Slayer,
Azazel and Game executables and the examples directory. A terminal launch
of the app wrapper can override the producer and client with
`FERRUCCIO_SLAYER` and `FERRUCCIO_AZAZEL`.

| Settings label | Flag | Environment | Fallback |
| --- | --- | --- | --- |
| Render queue | `--azazel-queue` | `FERRUCCIO_AZAZEL_QUEUE` | Shared queue, then app private queue |
| Render output | `--evidence` | `FERRUCCIO_EVIDENCE` | Shared render folder, then `~/Library/Application Support/Ferruccio/Renders` |
| Cache folder | `--cache` | `SLAYER_CACHE` | Shared Slayer cache, then `~/Library/Caches/Ferruccio/Slayer` |
| Source shelf | `--shelf` | `SLAYER_SHELF` | Shared source shelf, when present |
| Scene data | `--data` | `SLAYER_DATA` | Scene and producer defaults |

Before overriding a path, check its source in Settings. The working-view
helper and render command use the resolved scene paths.
Ferruccio does not import the Slayer project block from Game's config.
