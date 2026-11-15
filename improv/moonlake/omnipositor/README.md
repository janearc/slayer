# Omnipositor

For ordinary Slayer set jobs, the separate [`scene` adapter](SCENE.md)
composites a surface plate with actual 3D foreground transport. Its YAML
plan explicitly chooses mist resolution and plate reuse; no lake or shore
is required. The `stretch` pipeline described below remains Moonlake's.

Omnipositor composites a half-open range of film frames without writing the intermediate images used by the historical six-command chain. The current default path executes the extracted stages on the CPU. The experimental GPU path moves the plate, moon, overhead, fixtures, fire, sky glow, strobes, searchlight and laser splats, back haze, the UFO beam, matte, mist and develop computations to Metal. Searchlight and laser geometry is planned on the CPU; ocean stays on the CPU unless the separate experimental GPU ocean is selected. Its stages currently allocate and transfer intermediate buffers separately; this is not yet the device-resident, double-buffered pipeline described in the specification.

The command accepts `stretch --from FRAME --to FRAME --settings FILE --out DIR --png` for CPU PNG frames. `--encode FILE --ffmpeg /absolute/path/to/ffmpeg` streams the stretch to HEVC. At least one output is required. Relative input paths in the settings file resolve from that file's directory.

The required settings document names `frame`, `score`, `cues`, `passes`, `fire`, `camera`, `night`, `beam`, `gates`, `fps`, `disk_floor_bytes`, and `output`. The exact field names and a complete example live in `settings/settings_test.go`. Optional paths use `{path: ...}` or `{off: true}`. Input images must have the declared frame dimensions, except where a stage explicitly supports a different moon-reference size. The disk floor applies to each output filesystem, not to the checkout volume.

`--experimental-gpu --encode FILE` selects the hybrid path. Adding `--experimental-gpu-ocean` selects the newly assembled, unvalidated GPU ocean instead of the CPU ocean. Both refuse `--png` because GPU device identity and verified resume are not yet bound to the frame identity. `--trace FILE` writes a Chrome trace of per-frame stage wall times after a successful hybrid run. These spans include CPU dispatch, transfers and waits; they are not per-stage GPU timestamps. The encoder does not currently mux audio. The command checks its configured output-volume disk floor and load/swap gates, but these checks are not shared azazel admission. Run scene renders through the machine's render queue, not directly from a shell.

CPU PNG resume binds the frame to its inputs, settings and executable and verifies the stored PNG before reuse. Publication assumes cooperative writers on a trusted local filesystem. The output entries are links to immutable generations; old generations are not collected automatically. Overlapping publishers that race during publication can receive a refusal and must retry. Explicit redraw can replace a previously published owned frame.

`compare ORACLE CANDIDATE --check` compares PNG images against the specification's pixel and PSNR thresholds. PFM comparison is also available without `--check`. Neither self-comparison nor the presence of the comparison tool establishes equivalence to the historical chain.

The historical benchmark, full Metal lighting and ocean stages, device-resident intermediate buffers, double buffering, production profiling, shared azazel admission, and full-film validation are not complete. The experimental path has not been checked against the historical scene or watched in QuickTime.
