# Three personal telescopes

The manufactory names are dobsonian-36, apo-refractor-5 and sct-24.
Their apertures are the requested 36, 5 and 24 inches: 0.9144 m,
0.127 m and 0.6096 m using the exact international inch.

Use the names as scene kinds or inspect them through the catalogue:

    game run catalogue -- render dobsonian-36 apo-refractor-5 sct-24 \
      --out out/personal-telescopes

The Dobsonian has a mirror box, timber rocker, large altitude bearings,
paired V trusses and an upper cage. A flat secondary folds toward the side
focuser; a small parallel finder sits outside the cage.

[Obsession's construction description](https://www.obsessiontelescopes.com/why_buy/)
supplies the family reference: mirror box, rocker, side bearings and external
truss poles. The requested 36-inch instrument is an authored design.

The refractor has a long tube, dew shield, two cradle rings, a rear focuser,
diagonal and eyepiece, with a finder and counterweighted tripod.
[Explore Scientific's ED127](https://explorescientific.com/products/127mm-fcd100-apochromatic-refractor)
is a construction reference for the tube and cell, not a copied model.

The short Schmidt-Cassegrain has a primary and secondary, corrector proxy,
fork, pier, finder and rear focuser.
[Celestron's optical-tube guide](https://www.celestron.com/blogs/knowledgebase/the-ultimate-guide-to-celestron-optical-tubes)
describes the construction family.

All four details preserve each instrument's size and pose. Circular surfaces
change tessellation; small accessories remain present. Shapes use the
ordinary closed-mesh CPU and GPU paths.

The objective, eyepiece and corrector proxies are opaque polished scenery.
Apochromatic colour correction, a Schmidt corrector figure, image formation,
tracking and mount balance are not simulated.

Apertures are requested design choices, not measurements of a commercial
instrument. Other dimensions, support proportions and colours are authored
in manufacture/ground-choices.yaml. Tube length is not a focal length.
