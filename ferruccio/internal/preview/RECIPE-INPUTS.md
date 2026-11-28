# Recipe books in queued renders

Ferruccio adds animals and plants with the `@builtin` book selector.
Slayer reads that selector from the frozen scene and loads the book from
its executable. Azazel fingerprints the executable, and Slayer records the
embedded book's digest in the render identity.

When declaring files for Azazel, omit the exact selector in animal and
plant profile blocks, including aliased values. Continue fingerprinting
external books, composition files, imports and referenced sets. A path
such as `./@builtin` still names a disk file. This applies to queued Preview
and Render fully, which share the same input declaration.
