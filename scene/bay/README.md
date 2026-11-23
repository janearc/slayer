# Aquatic Park sightline and Alcatraz groundwork

`Alcatraz` loads the included USGS 3DEP crop and returns the land surface
plus original geometry for the Cellhouse, Building 64, New Industries,
Model Industries, powerhouse, lighthouse, Warden's House ruins, water
tower and landing apron. `Piers` independently builds Municipal Pier and Hyde
Street Pier. These helpers do not set lights, camera, or sea conditions.

```go
grid := bay.AquaticPark()
parts, err := bay.Alcatraz("scene/bay/alcatraz-3dep.tif", grid)
// Handle err before adding the parts to the scene.
piers, err := bay.Piers(grid)
// Add piers if the shot needs the cove's edges.
```

## Coordinates and scale

Grid is NAD83 / UTM zone 10N (EPSG:26910) in metres. Local x is east, y
is up and z is south. The selected swimmer is at 37.810500° N,
122.424220° W (550682.05 E, 4184945.93 N). It is about 23 m east and 8 m
south of the recorded concession structure on Municipal Pier.

NOAA's light coordinates, 37°49′34″ N, 122°25′20″ W, project to 550847.18 E,
4186679.06 N. The resulting grid bearing is 5.44° east of north and
the distance is 1.741 km. The horizontal look vector is
`[165.13, 0, -1733.13]` in local metres. The coordinates describe a
rendering viewpoint. They are not a swim route or current access advice.

The DEM is treated as a locally flat projected patch. The GeoTIFF
declares horizontal EPSG:26910, but has no vertical CRS GeoKey. The
default sea plane is the source's elevation zero. There is no tidal
datum conversion.

`Grid.SeaElevation` lets a shot choose its water
plane relative to this elevation data and shift the island accordingly.
The pier decks are visual estimates of 3.1 m and 2.7 m above the shot's
sea plane. No distant shoreline, boats or harbor furniture are included.

The separate Golden Gate view includes a coastal USGS 3DEP crop at 20 m
spacing. The crop covers the Presidio and the visible Marin Headlands.
The bridge axis starts from the [Yacht Racing Association tower marks](https://github.com/nkiesel/YRA-Marks/blob/master/README.md)
projected into EPSG:26910 and extended about their midpoint to the
1,280 m span from the [Golden Gate Bridge District's design tables](https://www.goldengate.org/bridge/history-research/statistics-data/design-construction-stats/).
The same tables set the side spans, tower height, width and cable size.

The deck profile, tower bracing, truss and parabolic cable curves are
visual approximations. The crop uses the same assumed mean plane as the
Alcatraz study. Its SHA-256 is
`211c0fd3b2fa75aae984b2617ad2a8bd40e3d0a2f6ec693fc824df266970a4ac`.

```text
https://elevation.nationalmap.gov/arcgis/rest/services/3DEPElevation/ImageServer/exportImage?bbox=545300,4184300,550420,4187700&bboxSR=26910&imageSR=26910&size=256,170&format=tiff&pixelType=F32&compression=LZ77&f=image
```

## Sources

- [USGS National Map 3DEP ImageServer](https://elevation.nationalmap.gov/arcgis/rest/services/3DEPElevation/ImageServer).
  Retrieved 2026-11-17. Exact export request is below. The file is
  Float32 LZW, 169 by 169 area cells, 4.970414 m between pixel centres.
  SHA-256 is
  `eab6d67a28cec596143e3baa9d4d8c31db9a8eb5e9461af179bc7f80e634d8fe`.

  `terrain.Raster` decodes the file and provides the pixel-centre
  registration. Faces use the decoded heights without exaggeration.
  A face needs all three samples and at least one vertex more than
  0.1 m above the selected sea plane. Water rendering is owned by
  the shot.
- [USGS National Map aerial service](https://basemap.nationalmap.gov/arcgis/rest/services/USGSImageryOnly/MapServer)
  supplied plan views for approximate building rectangles and pier
  centre lines. Read the island from the same UTM bbox at 840 by 840
  pixels and the cove from bbox `550300,4184480,551200,4185380` at
  900 by 900 pixels. Used to pick coordinates only. No imagery files
  are shipped as textures.
- [NPS Alcatraz map](https://www.nps.gov/goga/planyourvisit/upload/Alcatraz-map-Access-508.pdf)
  names and locates the landmark structures. Footprints and façade
  dimensions are approximations for the Aquatic Park sightline.
- The Cellhouse's flat stepped roof, concrete cornices, repeated dark
  window openings, and raised terrace are based on the south elevation
  in the [NPS Alcatraz Cultural Landscapes Inventory](https://irma.nps.gov/DataStore/FileSource/Get/2186800?filename=GOGA_Alcatraz_Island_CLI.pdf)
  and [HABS Cellhouse survey](https://www.loc.gov/item/ca1498/).
  Window widths and pilaster spacing are simplified for a background at
  1.74 km.

  Building 64 uses the hipped roof, brick casemates and three
  upper levels documented by the [NPS Building 64 structure report](https://www.nps.gov/goga/learn/historyculture/upload/Alcatraz-Bldg-64-HSR-for-web.pdf).
  The Warden's House is modeled as a roofless shell from the
  [NPS Warden's House record](https://home.nps.gov/places/000/alcatraz-wardens-house.htm).
- [NOAA Coast Pilot, chapter 7](https://nauticalcharts.noaa.gov/publications/coast-pilot/files/cp7/CPB7_C07_WEB.pdf)
  anchors the lighthouse position and describes its octagonal profile.
  The pale shaft, gallery and dark lantern rise to the 95 ft documented
  in the [NPS lighthouse record](https://home.nps.gov/places/000/alcatraz-lighthouse.htm).
  The 94 ft tank and its four-legged lattice use the
  [NPS water tower record](https://www.nps.gov/places/000/alcatraz-water-tower.htm).
- [NPS HALS CA-113-A field report](https://tile.loc.gov/storage-services/master/pnp/habshaer/ca/ca4300/ca4327/data/ca4327data.pdf)
  gives the Municipal Pier end coordinates, curved plan, approximate
  60 ft deck width, and pile construction. The deck and piles are
  simplified. The cove package omits the relocated historic ships
  listed in the [NPS Hyde Street project overview](https://www.nps.gov/safr/getinvolved/planning.htm).

```text
https://elevation.nationalmap.gov/arcgis/rest/services/3DEPElevation/ImageServer/exportImage?bbox=550440,4186200,551280,4187040&bboxSR=26910&imageSR=26910&size=169,169&pixelType=F32&format=tiff&compression=LZ77&f=image
```
