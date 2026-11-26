# Adding catalogue profiles

The catalogue's checked animal and plant entries are backed by profile books.
The editor presents those entries with their group and description and uses
Slayer's built-in book by default; a custom book may be chosen instead. A
placement transaction inserts both a named recipe and its thing in one
schema-checked change, preserving the selected profile controls and writing
a custom book path relative to the open scene where possible.

It refuses duplicate names and leaves source unchanged on error. Undo
restores the previous text.

The workspace must read a minimal set containing the proposed recipe with
Slayer's set reader before committing, so the book, family, species and
options really match without loading all unrelated assets on the UI thread.
The real candidate is independently schema-checked; full-scene staging may
still report unrelated errors.

The `@builtin` book uses the profile data embedded in the Slayer build. Its
actual content digest is included in the job's render identity. A custom book
remains an external dependency. Tests must save, reopen and stage a fish and
plant, check a mismatched book and recipe collision, and check that unrelated
YAML survives. The add transaction supports space and v2 ground sets.
