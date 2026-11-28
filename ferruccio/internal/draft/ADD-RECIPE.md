# Adding an existing recipe

A space set can place one of its explicitly authored recipes as a named
thing. The caller supplies a new name, the recipe name, an explicit seed,
and an east, north, up position in metres. The transaction checks the source,
requires an existing recipe and a unique name, and validates the result
before replacing the document. Failure leaves the exact source unchanged.
The caller owns undo by retaining the prior text, as for pose edits.

This is a first authoring primitive, not a catalogue or a general add-object
interface. It does not invent plant profiles, fish parameters, terrain anchors,
or a scale field. Tests must check a real constructed object from the edited
source, comments and unrelated objects, undo, duplicate names, missing
recipes, invalid coordinates and non-space sets.

The initial interface may list existing recipe names directly from the checked
draft and ask for an explicit new name. It must label that limited scope, use
the same undo history as other edits, and not start a render on addition.

The same transaction can write a named constructor kind. A caller must choose
from Slayer's live constructor registry, rather than inventing kind names;
the checked document still validates the whole resulting set before commit.
Kinds and named recipes are distinct choices even when their names collide.
The constructed saved result must be checked before describing the edit as
working. Profile-based fish and plants use the separate
[profile transaction](ADD-PROFILES.md).
