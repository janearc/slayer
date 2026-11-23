package main

import (
	"archive/zip"
	"bytes"
	"encoding/binary"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// viewerFiles is an entirely synthetic sky and earth, a few kilobytes on disk.
// The names exercise the real readers; none of these values are survey data.
func viewerFiles(t *testing.T) (string, string, string, string) {
	t.Helper()
	root := t.TempDir()
	data, earth, sky, themes := filepath.Join(root, "data"), filepath.Join(root, "earth"), filepath.Join(root, "sky"), filepath.Join(root, "themes")
	files := []string{"data-irac/GLM_00000+0000_mosaic_I4.fits", "data-irac/GLM_00000+0000_mosaic_I1.fits", "data-meerkat/MeerKAT_Galactic_Centre_1284MHz-StokesI.fits", "data-ppmap-l000_cdens/l000_cdens.fits"}
	for i, path := range files {
		writeSkyFixture(t, filepath.Join(data, path), i)
	}
	for i, path := range []string{"skyview-Mellinger-G-3600.fits", "skyview-H-Alpha-Comp-3600.fits", "skyview-IRIS-100-1800.fits", "skyview-408MHz-3600.fits"} {
		writeSkyFixture(t, filepath.Join(sky, path), i+4)
	}
	if err := os.MkdirAll(earth, 0700); err != nil {
		t.Fatal(err)
	}
	land := `{"type":"FeatureCollection","features":[{"type":"Feature","properties":{},"geometry":{"type":"Polygon","coordinates":[[[-80,-80],[80,-80],[80,80],[-80,80],[-80,-80]]]}}]}`
	writeViewerFile(t, filepath.Join(earth, "ne_50m_land.geojson"), []byte(land))
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	for name, text := range map[string]string{
		"splib07a_Wavelengths_BECK_Beckman_0.2-3.0_microns.txt": "synthetic wavelengths in micrometres\n0.36\n0.83\n",
		"splib07a_Seawater_Open_Ocean_SW2_lwch_BECKa_AREF.txt":  "synthetic uniform sea reflectance\n0.2\n0.2\n",
		"splib07a_Grass_dry.5+.5green_AMX28_BECKa_AREF.txt":     "synthetic uniform land reflectance\n0.6\n0.6\n",
	} {
		file, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err = file.Write([]byte(text)); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	writeViewerFile(t, filepath.Join(earth, "ASCIIdata_splib07a.zip"), archive.Bytes())
	writeMarbleFixture(t, filepath.Join(earth, "world.200407.3x5400x2700.jpg"))
	writeMarbleFixture(t, filepath.Join(earth, "BlackMarble_2016_01deg.jpg"))
	for _, name := range []string{"a", "twilight"} {
		writeViewerFile(t, filepath.Join(themes, name+".css"), []byte(":root { --ink: #ffffff; --ground: #101010; }\n"))
	}
	return data, earth, sky, themes
}

// writeViewerFile writes a fixture and creates its parent directories.
func writeViewerFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}

// writeSkyFixture writes a full-sky 8 by 4 CAR image in FITS block format.
func writeSkyFixture(t *testing.T, path string, shift int) {
	t.Helper()
	cards := [][2]string{{"SIMPLE", "T"}, {"BITPIX", "-32"}, {"NAXIS", "2"}, {"NAXIS1", "8"}, {"NAXIS2", "4"}, {"CTYPE1", "'RA---CAR'"}, {"CTYPE2", "'DEC--CAR'"}, {"CRPIX1", "4.5"}, {"CRPIX2", "2.5"}, {"CRVAL1", "180"}, {"CRVAL2", "0"}, {"CDELT1", "45"}, {"CDELT2", "45"}, {"CUNIT1", "'deg'"}, {"CUNIT2", "'deg'"}, {"EQUINOX", "2000"}}
	var header strings.Builder
	for _, card := range cards {
		fmt.Fprintf(&header, "%-80s", fmt.Sprintf("%-8s= %20s", card[0], card[1]))
	}
	fmt.Fprintf(&header, "%-80s", "END")
	for header.Len()%2880 != 0 {
		header.WriteByte(' ')
	}
	var image bytes.Buffer
	image.WriteString(header.String())
	for y := range 4 {
		for x := range 8 {
			if err := binary.Write(&image, binary.BigEndian, float32((x+2*y+shift)%16+1)); err != nil {
				t.Fatal(err)
			}
		}
	}
	for image.Len()%2880 != 0 {
		image.WriteByte(0)
	}
	writeViewerFile(t, path, image.Bytes())
}

// writeMarbleFixture makes a local synthetic photograph with two hemispheres.
func writeMarbleFixture(t *testing.T, path string) {
	t.Helper()
	picture := image.NewRGBA(image.Rect(0, 0, 16, 8))
	for y := range 8 {
		for x := range 16 {
			shade := color.RGBA{R: 30, G: 60, B: 200, A: 255}
			if x >= 8 {
				shade = color.RGBA{R: 180, G: 150, B: 80, A: 255}
			}
			picture.SetRGBA(x, y, shade)
		}
	}
	var out bytes.Buffer
	if err := jpeg.Encode(&out, picture, &jpeg.Options{Quality: 95}); err != nil {
		t.Fatal(err)
	}
	writeViewerFile(t, path, out.Bytes())
}
