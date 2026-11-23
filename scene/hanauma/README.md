# Hānauma Bay elevation and water

The overview uses a single continuous terrain surface across the shoreline.
It is a resampled subsection of NOAA's
[Hawaiʻi CUDEM topography and bathymetry](https://www.ncei.noaa.gov/access/metadata/landing-page/bin/iso?id=gov.noaa.ngdc.mgg.dem%3A299919).

The source is the [1/9 arc-second NOAA tile](https://noaa-nos-coastal-lidar-pds.s3.amazonaws.com/dem/NCEI_ninth_Topobathy_Hawaii_9428/tiles/ncei19_n21x50_w157x75_2021v1.tif),
SHA-256 `bc5912b3009f0edb71ee283a373af32b520295977820ec8e878b0e031d3d42ef`.
The original shows WGS 84 horizontal coordinates and MSL height, with
pixel-is-area registration. The local origin is
21.271100° N, 157.694500° W. Local X runs east and Z runs south.

The source Float32 samples were losslessly transcoded to tiled LZW for
Slayer's reader. The crop takes a bilinear sample every five metres on a
361 × 445 grid starting at X=-850 m and Z=-620 m. The sampling uses
103,800 metres per longitude degree and 110,720 metres per latitude
degree at this origin, with the original GeoTIFF pixel-centre offsets.

Heights were rounded to 0.01 m before serializing as a gzipped
`terrain.Window`. Its SHA-256 is
`dc978b20ebdd86e15dabd521ba149c7b49c346d9dba873ec2ea7bdecc422150a`.
The crop spans -204.27 m to 197.52 m. The scene's Y=0 uses the source
mean sea level, with no tide offset.

The water and relief jobs share an elevated camera and the same slope
lighting. A second aligned grid rasterizes NOAA's
[2007 geomorphological structure polygons](https://gis.ngdc.noaa.gov/arcgis/rest/services/nccos/BenthicMapping_Hawaii_Dynamic/MapServer/10).

A WGS 84 envelope query returned 33 features. Sand, pavement, rock and two
reef classes map to visual hardbottom weights of 0 to 192 on a 200-point
scale. Unmapped cells use a depth- and slope-driven material. The GIS
categories describe broad substrate, with no inference about present-day
live coral. The gzipped grid's SHA-256 is
`c9c073d6d426a33f3a20b0643cc892d6584f762884035831ee29f30e0b1b41cb`.

The water proof uses OceanTex ripples, dielectric refraction and
scene-chosen RGB absorption. A depth-weighted seabed light return
approximates diffuse illumination below the surface.

The tagged
refracted-sun route failed a boundary-intersection check on both the
waved and exact-flat variants. Land cover and detail within seabed
patches are procedural. Vegetation geometry and breaking surf are
outside this pass.

The original NOAA/USACE 2013 lidar products covering the inner bay were
also fetched via `celestial-data`'s fixed-product downloader. The point
cloud and 1 m DEM use tile `4QFJ3552`, datasets 5034 and 9365 respectively.
The overview uses the wider CUDEM mosaic at 5 m. The 1 m file is a
separate source for closer studies.
