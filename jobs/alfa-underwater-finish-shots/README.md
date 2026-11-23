# Moonlit Alfa shot controls

These four native GPU jobs retain the modeled Moon and synthetic water
coefficients from the approved underwater clip. They draw1080×960 Ninjafilm
half-portrait masters at one ray per pixel and ten-stop development.

whole-readable230.yaml frames the complete81m hull and screw-origin cavity
population. whole-readable-no-cavity.yaml removes only that population.
whole-readable-depth90.yaml translates hull, camera and cavities30m deeper,
leaving the relative composition unchanged. detail-backlight301.yaml shows
the five-blade stern from below, with an explicitly different Moon azimuth.
All preserve Moon altitude40°, light strength, exposure and development.

The water is an explicitly calm exact infinite plane, not a wavy GPU ocean.
Cavity density, fragmentation and pressure recovery are authored controls,
not CFD or resolved individual bubbles. Residual gas and independent cavity
phase are not present in these baseline files. Blade geometry remains static
in the base73623e49 renderer; rotating cavity phase does not rotate its mesh.

The ITTC2024 [cavitation descriptions](https://www.ittc.info/media/11816/75-02-03-032.pdf)
distinguish tip-vortex, cloud and bubble regimes. NIST's
[phase-function reference](https://pages.nist.gov/SCATMECH/docs/phase.htm)
explains directional HG scattering. These support the decomposition, not the
chosen numerical coefficients or a claim of measured submarine optics.

Run the named alfa-underwater-whole, alfa-underwater-no-cavity,
alfa-underwater-depth90 and alfa-underwater-detail Game targets. Outputs
are local under out/alfa-underwater-finish-shots. Shared-render receipts
and original approved images remain separate.
