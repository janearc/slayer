# Direct rendering adapter

The legacy `azazel render FILE [settings]` entry calls Slayer's
`session.Run` for one set. It bypasses queue admission and supervision.
Use [the queue](../README.md) for scheduled renders.
