# Moonlight entering water

`world.ocean.direct-moon: mean-plane` explicitly admits the staged Moon's
distant light as a water source. It is independent of diffuse sky scattering
and clear-cloud sky samples. Existing scenes remain unchanged without it.

The source is the actual Moon emitter inserted by hillside, with its authored
direction and existing atmospheric attenuation. It is never a replacement
spotlight or a synthetic fill. The CPU uses the existing horizontal mean-plane
Snell/Fresnel source approximation and applies incoming water attenuation,
opaque visibility, local-cloud extinction, and outgoing water attenuation.
Wave-resolved focusing and caustics are not included. Metal refuses this route.

The reader refuses simultaneous direct-sun and direct-moon declarations.
Staging requires `light: moon`, an above-horizon authored Moon, and the exact
inserted distant emitter. Initial underwater medium state remains mandatory.
Validation rejects additional directed lamps and synthetic fill tags. Both
world-only sets and sets with manufactured objects use the same source rule.
