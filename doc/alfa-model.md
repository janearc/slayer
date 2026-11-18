# Native Alfa model detail

`manufacture/alfa.yaml` and its Go factory make Project 705 geometry from
Slayer's native lofts, turned profiles and rods. The model uses no imported
submarine mesh. Length and beam come from the cited russianships page; the
shape and all undimensioned placements are authored approximations from
photographs, not a measured replica.

## Sourced features and authored dimensions

* *American Submariner*, Third Quarter 2023, “Project 705—Lira,” pp. 22–23,
  identifies a five-blade, non-skewback main propeller and two two-blade
  auxiliary propellers on motor pods at the stern-plane tips. The model draws
  those counts and locations. Their dimensions, pitch, and the simplified
  blade profiles are chosen because the article gives no dimensioned drawing.
* The same article identifies the sail's periscope/radar/radio mast forms and
  a starboard aft condenser cooling scoop. `complexity: demo` adds four
  simple antenna rods, two low sail-crown covers and one shallow scoop plate.
  Those forms, dimensions and positions are approximate; covers are a visual
  choice, not a claim about a specific hatch layout. The scoop plate is not a
  boolean cut or functional intake.
* The surfaced photos show a dark hull. The dedicated `alfa-hero` kind selects
  a near-black paint finish on the coated exterior parts. Ordinary `alfa`
  remains bare titanium by default. The black paint is a shot/model choice;
  the sources do not establish the outer hull's coating.

The article is available at
<https://ufdcimages.uflib.ufl.edu/AA/00/04/07/89/00111/2023-%20Issue%203.pdf>.
The local CIA Thamm 2008 scan and the shelf's russianships text remain the
sources for the existing measurements and rescue-sphere/sail notes.

## Levels

The factory leaves `quick` as the default. The masts and other extra surfaces
are added only at `demo`; main and auxiliary screw blades are present at all
levels. Both the ordinary and black-coated kinds use the same geometry and
the same selectable complexity.

The model establishes geometry and finish only. It does not establish
accurate San Onofre water, a wake, or physical water optics.
