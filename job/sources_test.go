package job

import (
	"bytes"
	"testing"
)

func TestSourcesSurviveAdmissionAndOverrides(t *testing.T) {
	text := heading + "scene: {named: honopu-beach}\n" +
		"sources:\n" +
		"  - name: NOAA Hawaiʻi CUDEM\n" +
		"    uri: https://coast.noaa.gov/dataviewer/#/lidar/search/where:ID=9428\n" +
		"    note: 4 m samples of the retained survey\n"
	job := read(t, text)
	sources := job.Sources()
	if len(sources) != 1 || sources[0].Name != "NOAA Hawaiʻi CUDEM" ||
		sources[0].URI != "https://coast.noaa.gov/dataviewer/#/lidar/search/where:ID=9428" {
		t.Fatalf("sources lost: %+v", sources)
	}
	sources[0].Name = "changed"
	if job.Sources()[0].Name != "NOAA Hawaiʻi CUDEM" {
		t.Fatal("caller changed the admitted job")
	}
	if !bytes.Contains(job.Canonical(), []byte("NOAA Hawaiʻi CUDEM")) {
		t.Fatal("canonical job lost the source")
	}
	changed, err := job.With(Width(600))
	if err != nil || len(changed.Sources()) != 1 || changed.Sources()[0] != job.Sources()[0] {
		t.Fatalf("override lost sources: %v, %+v", err, changed.Sources())
	}
}

func TestSourceNeedsNameAndURI(t *testing.T) {
	for _, text := range []string{
		heading + "scene: {named: honopu-beach}\nsources: [{name: NOAA}]\n",
		heading + "scene: {named: honopu-beach}\nsources: [{name: NOAA, uri: ''}]\n",
	} {
		if _, err := reading(t, text); err == nil {
			t.Fatalf("accepted invalid source: %s", text)
		}
	}
}
