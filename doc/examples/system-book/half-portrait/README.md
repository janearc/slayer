# Twelve 1080×960 system views

This edition renders each selected system-book image at 1080×960 and `rays: 3`
(nine samples per pixel). These jobs preserve each selected scene, camera,
renderer and exposure/development choices. The new aspect ratio differs
from the selected square and 640×400 images, so framing, all twelve flock
silhouettes and highlights need visual assessment. Some originals used four
rays per axis; this requested edition uses three, not sixteen samples.

| Job | Selected source | Scene and camera | Pinned binary |
|---|---|---|---|
| 01 | transfer 595 | `../system.yaml`, `transfer` | 02afa87e9 |
| 02 | Amber crown 597 | `../system.yaml`, `jovian-crown` | 02afa87e9 |
| 03 | Azure 621 | `../azure-primary-v1.yaml`, `azure-primary` | 02afa87e9 |
| 04 | twilight flock 623 | `flock/azure-flock-twilight-v15.yaml`, `flock` | 0e7ed2a60 |
| 05 | Fresno 622 | `../fresno-haze-v2.yaml`, `fresno-haze` | 02afa87e9 |
| 06 | Fresno close 500 | `caprica/10-drop-oblique-surface-crop.yaml`, `10-drop-oblique` | 701507dec |
| 07 | Caprica quarter 478 | `caprica/caprica-lookbook.yaml`, `02-quarter` | 701507dec |
| 08 | Caprica night 479 | `caprica/caprica-lookbook.yaml`, `03-night` | 701507dec |
| 09 | Caprica deck 481 | `caprica/caprica-lookbook.yaml`, `05-deck` | 701507dec |
| 10 | Caprica terminator 492 | `caprica/04-terminator-crop.yaml`, `04-terminator` | 701507dec |
| 11 | pair development 639 | `../caprica-fresno-pair-v1.yaml`, `pair-view` | 02afa87e9 |
| 12 | Amber portrait 636 | `../amber-rebase-local-v2.yaml`, `amber` | 02afa87e9 |

All use GPU-native and ReSTIR off, with fresh named PNG and PFM outputs.
Slot 11 first writes `draw.png` and `light.pfm`; its selected 639 image was
a separate −2-stop development. After the new raw exists, run the separate
`11-caprica-fresno-print.yaml` development to produce `proof.png`; do not
substitute the old raw. Slot 06 writes its raw to its own new folder; the
old job wrote the final raw over a preview's raw, which was copied back into
the final folder and recorded in the earlier receipt.

The scene versions are deliberately mixed. Older Caprica/Fresno pictures
use the prior hot-white-dwarf model; the working system uses the cooler
7400 K companion. The binary stream and Amber crown are invented wisps, not
simulated accretion or aurora. Azure's flock is an artistic local scene with
enlarged light cues and an unpainted membrane; Amber's portrait is a separate
local scene omitting wisps, auxiliary lamps, other planets and comet pieces.
Amber and Azure are fixed positions at the depicted epoch, not propagated
orbits. These qualifications remain with the eventual book captions.

Binary SHA-256 receipts:

- `02afa87e9`: `8602c43d223511da8b77b3d496c03deef18cec8004051b7314dc2c7827e72064`
- `0e7ed2a60`: `aa80e976705d32ea487055599197d0aa778ea07fe88563cf8d081af3101100a4`
- `701507dec`: `7f8bc25d1e478b87ae3e66b2289d789284b75e7e1e2a8178a63a2585b5def7a7`

## Coordinating the twelve queued jobs

The jobs resolve their scene files from this directory and write to the
checkout's `out/2026-11-27-system-half-*/` directories. For a queued
edition, `coordinator.sh` watches the twelve submitted draws in the order
listed above, submits the slot 11 development, and builds the book when all
twelve PNG/PFM pairs are available. The existing queue keys use
`system-half-<slot-and-view>-<pinned-revision>`.

Set `AZAZEL_BINARY`, `AZAZEL_QUEUE`, `CAPRICA_FIRST_SEQUENCE`, and
`CAPRICA_PRINT_REQUEST` to the queue client, queue, first draw's sequence
number, and prepared slot 11 development request. If you stage outputs
elsewhere, set `CAPRICA_FRAME_PREFIX` to the full path prefix preceding
the slot names. The look-book generator defaults to `bin/caprica-lookbook`
in the checkout. Set `CAPRICA_BOOK_DIRECTORY` to redirect the generated
book. To update an existing gallery, supply `CAPRICA_GALLERY_NOTES` and
`CAPRICA_GALLERY_ENTRY` (book directory relative to that gallery), and
set `CAPRICA_GALLERY_REFRESH` to its refresh script if required.
