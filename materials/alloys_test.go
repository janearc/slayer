package materials

import "testing"

func TestReflectancePagesRefuseAmbiguousOrUnattributedData(t *testing.T) {
	original := alloys["nickel-aluminium-bronze"]
	for _, mode := range []string{"empty", "duplicate", "source",
		"reflectance", "band", "wavelength"} {
		leaf := *original.proxy
		leaf.rows = append([]reflectanceRow(nil), original.proxy.rows...)
		switch mode {
		case "empty":
			leaf.rows = nil
		case "duplicate":
			leaf.rows = append(leaf.rows, leaf.rows[0])
		case "source":
			leaf.paper = ""
		case "reflectance":
			leaf.rows[0].reflectance = 1
		case "band":
			leaf.rows[0].band = Band(99)
		case "wavelength":
			leaf.rows[0].wavelength = -1
		}
		if _, err := leaf.optics(); err == nil {
			t.Fatalf("accepted %s reflectance page", mode)
		}
	}
	mixed := original
	mixed.sight = metals["titanium"].sight
	if _, err := mixed.made("ambiguous"); err == nil {
		t.Fatal("silently chose between measured and estimated optics")
	}
	for _, mode := range []string{"density", "coefficient"} {
		mixed = original
		if mode == "density" {
			mixed.dense = &aluminiumDense
		} else {
			mixed.stops = []coefficientRow{aluminiumTaken}
		}
		if _, err := mixed.made("ambiguous"); err == nil {
			t.Fatalf("silently ignored %s records", mode)
		}
	}
}

func TestBulkOnlyRowRefusesBadDensityOrMissingSource(t *testing.T) {
	original := *alloys["nickel-aluminium-bronze"].bulk
	kind, err := ByName("titanium")
	if err != nil {
		t.Fatal(err)
	}
	for _, mode := range []string{"density", "source", "material", "paper", "text"} {
		row := original
		switch mode {
		case "density":
			row.density = 0
		case "source":
			row.release = ""
		case "material":
			row.material = ""
		case "paper":
			row.paper = ""
		case "text":
			row.text = ""
		}
		if _, err := row.attach(kind); err == nil {
			t.Fatalf("accepted invalid %s", mode)
		}
	}
}
