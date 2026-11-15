# azazel/engine

This package draws one set file. It calls the library that `slayer render`
calls, `hillside.Run`, with the set's path and the render settings, and gives
back hillside's exit number.

It is not the queue. It has no job folder, no admission, no locking and no
publication. The queue's packages under `azazel/internal` are not wired to
it, and they do not import it or the renderer.

`azazel render FILE [settings]` is the command over it. It draws the same
picture as `slayer render` does for the same set and settings.
