# Cassini and Huygens visual models

Source: NASA/Brian E. Kumanchik, Cassini-Huygens (A), NASA 3D Resources:
https://science.nasa.gov/3d-resources/cassini-huygens-a/

Three original GLBs are retained under research/cassini-huygens. Their SHA-256
hashes are:

- attached.glb: f8bc110e23821021e99b88634843449cc8e98d86d0d8561c612c1cf0404e64e7
- cassini.glb: 87424f2c11b8e0c6a212292221e317396735f5e0876b7bb1204351998dd22f2c
- huygens.glb: d2cb317244d4a64eca408bcaf1c908f77c3c25124df2b4d083ec23f5eee14aca

Original Draco-compressed geometry was decoded with glTF Transform 4.5.1 and
node transforms baked to separate STL surfaces; original textures are not
reproduced. The export and repair scripts are saved beside the originals.

Cassini source surfaces had inconsistent winding and non-manifold edges,
which Slayer refused. For this WIP, winding was oriented and 133 of 25,052
NASA faces removed to eliminate non-manifold conflicts; the untouched GLB
remains the reference. The conversion and lost faces need further review.

Cassini has eight source mesh parts and 24,919 triangles. Huygens' two source
meshes contain 672 triangles, scaled by 2.7/2.6153474 to match the 2.7 m
entry-shield diameter in ESA's Huygens spacecraft description:
https://www.esa.int/Science_Exploration/Space_Science/Cassini-Huygens/Huygens_spacecraft

A segmented edge seam and 16 fittings have been added by eye; Huygens totals
66 parts and 3,168 triangles. Added fixtures are not measured flight hardware.
Cassini-Huygens retains NASA's attached Huygens transform: 10 NASA mesh parts,
25,591 triangles, without the separate probe's extra invented seam. Finishes
are display approximations; shading and the faceted source need refinement.
NASA media guidelines: https://www.nasa.gov/nasa-brand-center/images-and-media/
