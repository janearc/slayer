# Slayer

Slayer is a ray tracer for building pictures and films from scenes: from a
hillside at moonrise to a city, spacecraft, or star inside a Dyson swarm. It
combines procedural geometry, materials, light and a photographic darkroom.
Supported scenes can also render using Metal on macOS.

Slayer keeps computed light separate from development: save a raw frame and
revisit its exposure and white balance without tracing it again. Its models
distinguish measured inputs from authored choices.

Slayer is in early development. Interfaces and supported features can change,
and some scenes require additional data or substantial rendering resources.
Today Slayer is intended for people who write software and the agents working
with them.

## Try it

**Ferruccio** is Slayer's native scene editor for Apple silicon Macs. Open a
scene, change it and render a preview. Download the disk image from the
[releases](https://github.com/janearc/slayer/releases), then follow the
[Ferruccio quick start](doc/applications/ferruccio/quick-start.md).

For the command line, [Getting started](doc/getting-started.md) takes you
from a checkout to a first picture. [Building Slayer](doc/building.md)
covers the required tools, source shelf and Game workflow. Once built, the
first picture needs no additional scene data.

## Learn more

- [How Slayer works](OPERATION.md): scenes, tracing, development and output.
- [Examples](EXAMPLES.md) and the [visual guide](doc/visualguide/README.md):
  scenes and commands to try.
- [Renderer](doc/renderer.md), [light](doc/light.md), [modes](doc/modes.md)
  and [film](ninjafilm/README.md): how pictures are made.
- [Manufactory catalogue](doc/catalogue.md) and [making models](doc/making-models.md):
  procedural objects and how to contribute one.
- [Azazel](doc/azazel.md): the shared render queue and its worker.
- [Contributing](CONTRIBUTING.md), [testing](doc/testing.md) and the
  [package guide](PACKAGES.md): working on Slayer.

`doc/` also contains design notes and historical studies. Those documents
may describe proposals or earlier results; the current guides, source and
command help describe what is implemented.
