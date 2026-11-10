# Smoke render records

Small renders make visual changes easier to spot. This directory holds the
images and timing history for Slayer's smoke scenes.

`results.jsonl` has one record per render: timestamp, commit, scene, CPU or
GPU, dimensions, rays per pixel, elapsed time, image path, and SHA-256.
The earliest entries were recorded manually on 2026-10-02 and 2026-10-03,
before the smoke workflow was established.

`index.html` shows the latest CPU and GPU render of each scene side by side,
along with its rendering-time history. Results, gallery, and images are
uncommitted data. From the Slayer root, `game build` then `game run lightbox`
serves them at <http://localhost:8765>.
