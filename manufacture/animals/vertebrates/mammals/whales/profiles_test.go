package whales

import "testing"

func TestProfileLookupDoesNotExposeBookStations(t *testing.T) {
	first, found := Default().Get("orca")
	if !found || len(first.Stations) == 0 {
		t.Fatal("missing embedded orca")
	}
	before := first.Stations[0]
	first.Stations[0][1]++
	again, found := Default().Get("orca")
	if !found || again.Stations[0] != before {
		t.Fatal("mutating a returned profile changed its book")
	}
}
