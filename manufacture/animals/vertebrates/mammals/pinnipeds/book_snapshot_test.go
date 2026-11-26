package pinnipeds

import (
	"sync"
	"testing"
)

func TestProfileLookupRetainsIndependentData(t *testing.T) {
	first, ok := Default().Get("harbor-seal")
	if !ok {
		t.Fatal("missing seal")
	}
	before := first.Outline[0]
	first.Outline[0][1]++
	*first.Length++
	study := first.Studies["detailed-wet"]
	*study.Wet = 0
	again, ok := Default().Get("harbor-seal")
	if !ok || again.Outline[0] != before || *again.Length == *first.Length || *again.Studies["detailed-wet"].Wet == 0 {
		t.Fatal("returned profile exposed book data")
	}
	walrus, ok := Default().Get("walrus")
	if !ok || len(walrus.WalrusOutline) == 0 || walrus.Evidence["thoracic"].Value == nil {
		t.Fatal("missing walrus profile data")
	}
	walrusBefore := walrus.WalrusOutline[0]
	walrus.WalrusOutline[0][1]++
	*walrus.Evidence["thoracic"].Value = 99
	again, ok = Default().Get("walrus")
	if !ok || again.WalrusOutline[0] != walrusBefore || *again.Evidence["thoracic"].Value != 14 {
		t.Fatal("returned walrus profile exposed book data")
	}
}

func TestConcurrentProfileLookupsOwnTheirData(t *testing.T) {
	const workers = 8
	var group sync.WaitGroup
	for i := 0; i < workers; i++ {
		group.Add(1)
		go func() {
			defer group.Done()
			for repeat := 0; repeat < 20; repeat++ {
				p, ok := Default().Get("harbor-seal")
				if !ok || p.Outline[0][1] != .006 || *p.Studies["detailed-wet"].Wet != .85 {
					t.Error("book data changed between lookups")
					return
				}
				p.Outline[0][1]++
				*p.Studies["detailed-wet"].Wet = 0
			}
		}()
	}
	group.Wait()
}
