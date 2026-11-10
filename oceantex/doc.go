// Package oceantex draws the sea from any point of view, a beach to orbit,
// using only published formulas and measurements. Colour goes through
// github.com/janearc/libtheme-css: its CIE 1931 observer, its D65 and
// black body, and its sRGB space.
//
// View is a picture from any camera. Render is a square, tiling texture
// from straight above. Both show the waves at least as long as the sea a
// pixel covers, and turn the shorter ones into roughness: their slope
// variance is what is left of the Cox and Munk measurement after
// subtracting what the picture resolves, and it widens the sun's glint.
//
// Near the camera that is sharp facets; toward the horizon and from orbit
// it is the smooth glitter pattern, with no switch between.
//
// # Numbers
//
// Every measured or chosen number is in sources.yaml, with the
// publication it belongs to, the URL it was read on and the date. The
// code reads each by name and holds none of its own; Sources lists them.
//
// # Sources
//
// The keys in square brackets through the code are:
//
//	[CT65]  Cooley & Tukey, Math. Comp. 19, 297-301 (1965). The FFT.
//	[MWM87] Mastin, Watterberg & Mareda, IEEE CG&A 7(3), 16-23 (1987).
//	        Fourier synthesis of ocean scenes.
//	[H73]   Hasselmann et al., Dtsch. Hydrogr. Z. Suppl. A8(12) (1973).
//	        JONSWAP spectrum.
//	[PM64]  Pierson & Moskowitz, J. Geophys. Res. 69, 5181-5190 (1964).
//	        The fully developed sea.
//	[D85]   Donelan, Hamilton & Hui, Phil. Trans. R. Soc. A 315, 509-562
//	        (1985). Directional spreading.
//	[Ho15]  Horvath, Proc. DigiPro 2015, 29-39. The above as used in
//	        graphics, the TMA depth factor, the amplitude normalisation.
//	[T01]   Tessendorf, Simulating Ocean Water, SIGGRAPH 2001 course.
//	        Choppy waves. Course notes, not peer reviewed.
//	[CM54]  Cox & Munk, J. Opt. Soc. Am. 44, 838-850 (1954). Slope variance
//	        from sun glitter; the glint formula.
//	[S67]   Smith, IEEE Trans. Antennas Propag. 15, 668-671 (1967), as
//	        given for the sea by Ross & Dion, Proc. SPIE 5572 (2004).
//	        Shadowing.
//	[QF95]  Quan & Fry, Appl. Opt. 34, 3477-3480 (1995). Refractive index.
//	[PF97]  Pope & Fry, Appl. Opt. 36, 8710-8723 (1997). Water absorption.
//	[M74]   Morel, in Optical Aspects of Oceanography (1974), 1-24.
//	        Scattering by pure seawater.
//	[PS81]  Prieur & Sathyendranath, Limnol. Oceanogr. 26, 671-689 (1981).
//	        Pigment absorption.
//	[GM83]  Gordon & Morel (1983); [MM01] Morel & Maritorena, J. Geophys.
//	        Res. 106, 7163-7180 (2001); [U94] Ulloa et al., Appl. Opt. 33
//	        (1994). The case 1 water model, as given in Mobley's Ocean
//	        Optics Web Book.
//	[G88]   Gordon et al., J. Geophys. Res. 93, 10909-10924 (1988);
//	[L02]   Lee, Carder & Arnone, Appl. Opt. 41, 5755-5772 (2002).
//	        Reflectance.
//	[MO80]  Monahan & O'Muircheartaigh, J. Phys. Oceanogr. 10, 2094-2099
//	        (1980). Whitecap coverage.
//	[K84]   Koepke, Appl. Opt. 23, 1816-1824 (1984). Whitecap reflectance.
//	[USNO]  U.S. Naval Observatory, Computing Approximate Solar
//	        Coordinates; [Me] Meeus, Astronomical Algorithms. The sun.
//	[KY89]  Kasten & Young, Appl. Opt. 28, 4735-4738 (1989). Air mass.
//	[HT74]  Hansen & Travis, Space Sci. Rev. 16, 527-610 (1974). Rayleigh
//	        optical depth.
//
//	[TG82]  Thornton & Guza, J. Geophys. Res. 87, 9499-9508 (1982). Wave
//	        height held to the depth in the surf.
//	[Mc94]  McCowan, Phil. Mag. 38, 351-357 (1894). The breaking limit.
//	[W82]   Whitlock, Bartlett & Gurganus, Geophys. Res. Lett. 9, 719-722
//	        (1982). Reflectance of fresh foam.
//
// The WMO sea state code, the Beaufort relation and NOAA's tidal speeds
// are in sources.yaml with theirs.
//
// README.md lists what joins these that is not from any of them, each
// marked GLUE in the code, and what is left out: a sky, the atmosphere,
// the Earth's curve, a sloping shore and its surf, and a map of the
// world's chlorophyll.
package oceantex
