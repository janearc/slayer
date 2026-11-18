package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"strconv"

	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/sky"
	"github.com/janearc/slayer/sources"
)

// gaiaPlan pins a saved Gaia DR3 query. Star radius and zero-magnitude light
// are display choices; the source's positions and G magnitudes are preserved.
type gaiaPlan struct {
	File   string  `json:"file" yaml:"file"`
	SHA256 string  `json:"sha256" yaml:"sha256"`
	Radius float64 `json:"radius-deg" yaml:"radius-deg"`
	Zero   float64 `json:"zero-magnitude" yaml:"zero-magnitude"`
}

// load verifies the exact source bytes before they can affect a render.
// Relative paths are on the source shelf, not inside a repository.
func (plan gaiaPlan) load() (*sky.Catalogue, error) {
	path := plan.File
	if !filepath.IsAbs(path) {
		shelf, err := sources.Here()
		if err != nil {
			return nil, err
		}
		path = filepath.Join(shelf, path)
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	const limit = 4 << 20
	raw, err := io.ReadAll(io.LimitReader(file, limit+1))
	if err != nil {
		return nil, err
	}
	if len(raw) > limit {
		return nil, fmt.Errorf("Gaia presentation catalogue exceeds four MiB")
	}
	if fmt.Sprintf("%x", sha256.Sum256(raw)) != plan.SHA256 {
		return nil, fmt.Errorf("Gaia catalogue %s differs from its sha256 pin", path)
	}
	return readGaia(bytes.NewReader(raw), plan.Radius, plan.Zero)
}

// readGaia uses ICRS RA/Dec at the catalogue epoch, with north along local Y
// and increasing RA toward negative Z. It does not apply proper motion.
func readGaia(input io.Reader, radius, zero float64) (*sky.Catalogue, error) {
	if !(radius > 0 && radius <= 1) || !(zero > 0) || math.IsInf(zero, 0) {
		return nil, fmt.Errorf("Gaia presentation needs a finite positive size and brightness")
	}
	reader := csv.NewReader(input)
	header, err := reader.Read()
	if err != nil {
		return nil, err
	}
	if len(header) != 4 || header[0] != "source_id" || header[1] != "ra" ||
		header[2] != "dec" || header[3] != "phot_g_mean_mag" {
		return nil, fmt.Errorf("Gaia CSV needs source_id,ra,dec,phot_g_mean_mag")
	}
	var stars []sky.Star
	seen := map[uint64]bool{}
	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		star, id, err := gaiaStar(row, zero)
		if err != nil {
			return nil, err
		}
		if seen[id] || len(stars) == 12000 {
			return nil, fmt.Errorf("Gaia catalogue repeats a source or exceeds 12000 stars")
		}
		seen[id] = true
		stars = append(stars, star)
	}
	return sky.NewCatalogue(stars, radius*maths.Degree, maths.Light{})
}

// gaiaStar converts one measured position and magnitude into display light.
func gaiaStar(row []string, zero float64) (sky.Star, uint64, error) {
	id, err := strconv.ParseUint(row[0], 10, 64)
	if err != nil || id == 0 {
		return sky.Star{}, 0, fmt.Errorf("invalid Gaia source identifier %q", row[0])
	}
	var values [3]float64
	for index := range values {
		value, err := strconv.ParseFloat(row[index+1], 64)
		if err != nil || math.IsNaN(value) || math.IsInf(value, 0) {
			return sky.Star{}, 0, fmt.Errorf("invalid Gaia coordinate or magnitude %q", row[index+1])
		}
		values[index] = value
	}
	ra, dec, magnitude := values[0], values[1], values[2]
	if ra < 0 || ra >= 360 || dec < -90 || dec > 90 || magnitude < -30 || magnitude > 30 {
		return sky.Star{}, 0, fmt.Errorf("Gaia source %d has an out-of-range position or magnitude", id)
	}
	alpha, delta := ra*maths.Degree, dec*maths.Degree
	direction := maths.V(math.Cos(delta)*math.Cos(alpha), math.Sin(delta),
		-math.Cos(delta)*math.Sin(alpha))
	return sky.Star{Toward: direction, Light: maths.Grey(zero * math.Pow(10, -.4*magnitude))}, id, nil
}
