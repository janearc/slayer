# Ferruccio queue dispatch

A packaged Ferruccio window starts dispatch on demand after its submission is
visible in a checked Azazel status response. A paused queue stays paused. The
client gives a recent scheduler time to the current worker and tries again if
that time stops advancing while its job is pending.

The app runs the selected queue client with `work --queue PATH`. Azazel takes
its own coordinator lock before admitting jobs. Without `--serve` it drains
pending work and exits. The worker runs in its own process group, writes to
`azazel-worker.log` in the initiating render folder, and can finish other jobs
after that window closes. The app never signals this process. The existing
Stop action still addresses only the exact submitted render.
