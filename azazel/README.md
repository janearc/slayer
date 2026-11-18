# azazel

Azazel is Slayer's command scheduler. Agents submit existing foreground
Slayer commands to a shared durable queue; one worker admits and supervises
them and retains their outcomes. Game builds and checks Azazel.

Start with [the human guide](../doc/azazel.md), or run
`bin/azazel --agent-help` for the agent workflow and a complete request.
`bin/azazel schema` prints the YAML/JSON contract.

The queue is under validation and is not ready for the shared render
migration. The guide identifies remaining recovery, output-preservation,
resource and gallery work. Process success does not establish a verified
picture or film.

The [design](DESIGN.md) describes the complete system. The
[Bookshelf review](BOOKSHELF-REVIEW.md) records the source guidance and
current implementation boundaries. Private [reservations](OWNED-RESERVATIONS.md)
and [transactions](TRANSACTION.md) remain separate from render attempt leases.

The older [engine](engine/README.md) adapter backs direct `azazel render`;
that entry point does not submit to the queue.
