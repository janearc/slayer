# GPU beam and atmospheric lighting

The experimental lighting accelerator contains Metal implementations of sky glow, strobes, fire, back haze and the UFO beam. The beam computes the projected halo, cone intersections, layered fog and water-foot contribution per pixel. The below-foot halo clamps the negative across value to zero, retaining the historical beam correction.

Searchlights and lasers still plan their projected splats on the CPU. A Metal kernel applies those splats in order per pixel using bounded spatial tile lists, avoiding competing writes. Dense beams and broad splats still need performance measurement. Fire frames and other inputs are uploaded for each call rather than held on the device across the stretch.

The tests in this package pass in the draw tier as of 2026-11-14. Historical beam frames and performance have not been checked.
