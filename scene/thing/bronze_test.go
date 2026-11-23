package thing

import (
	"testing"

	"github.com/janearc/slayer/materials"
)

// Nickel aluminium bronze reads the shared material's optics band for
// band, reflects the three normal reflectances of its source face on, and
// takes its finish independently of its alloy. The anti-test: it is not
// titanium.
func TestNickelAluminiumBronzeUsesSharedMaterialAndIndependentFinish(t *testing.T) {
	material := madeMaterial(t, "nickel-aluminium-bronze")
	bronze := NickelAluminiumBronze()
	proxy := metalFaceOn(t, "bronze")
	proxies := [metalColours]float64{proxy.R, proxy.G, proxy.B}
	bands := [metalColours]materials.Band{materials.Red, materials.Green,
		materials.Blue}
	for place, band := range bands {
		got := Conductor(unity, bronze.N[place], bronze.K[place])
		if !nearly(got, proxies[place], picked(t, "made-tolerance")) {
			t.Fatalf("band %d normal reflection %v differs from proxy %v",
				place, got, proxies[place])
		}
		read := material.Optics().At(band)
		if bronze.N[place] != read.N() || bronze.K[place] != read.K() {
			t.Fatal("thing invented separate bronze constants")
		}
	}
	satin, err := Made(material, worn(t, "satin"))
	if err != nil {
		t.Fatal(err)
	}
	brushed, err := Made(material, worn(t, "brushed"))
	if err != nil {
		t.Fatal(err)
	}
	if satin.Name != "nickel-aluminium-bronze" ||
		satin.Satin == brushed.Satin || *satin.Metal != *brushed.Metal {
		t.Fatal("finish changed alloy optics or ignored roughness")
	}
	if *bronze == *Titanium() {
		t.Fatal("bronze reads as titanium")
	}
}
