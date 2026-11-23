# New Horizons VTAD alternate

NASA Visualization Technology Applications and Development 3D model:
https://science.nasa.gov/resource/new-horizons-3d-model/
Original GLB: research/new-horizons/vtad/source.glb, SHA-256
cf152d8cea17c4a83711041cb0d29ca1d980bfafa084ada2656569ea92cdd73f.

Five source mesh parts have 8,260 triangles after removing eight faces with
non-manifold conflicts and correcting winding. The original GLB is retained;
conversion and repair scripts are alongside it. Scale is selected so the dish
spans 2.1 m.

Its source textures are not yet reproduced: materials are flat
picked gold, dark and white colours. Geometry and finish are visual WIP, not
flight-grade or material-certified. NASA media guidelines apply:
https://www.nasa.gov/nasa-brand-center/images-and-media/

The instrument texture meshes have now been grouped by approximate colour:
a script samples NASA's original base-colour image at each triangle's UV
centroid and assigns a picked black, graphite, warm, silver or pale material.

The original 8,260 triangles are retained. These 12 material groups do not
represent 12 independently modelled spacecraft components. UV textures and
the original foil normal map are not rendered; the foil stays visually smooth
and the colour groups may lose small features. Sampling is an approximation,
not an assertion of the actual hardware's composition.
