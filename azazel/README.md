# Azazel

Azazel is Slayer's durable render queue. It schedules foreground Slayer
commands, runs CPU and GPU work concurrently, and records job outcomes.
With a render root configured, it updates the gallery while jobs run.

See [the operator guide](../doc/azazel.md) for setup, submission, recovery and
limits. Run `bin/azazel --agent-help` for a client example and
`bin/azazel schema` for the YAML and JSON request contract.

- [Scheduler design](DESIGN.md)
- [Gallery publisher](pages/README.md)
- [Dated validation results and smoke recipes](VALIDATION.md)
