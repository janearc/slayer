package manufacture

import (
	"crypto/sha256"
	"fmt"
	"math/rand/v2"
	"slices"
	"strings"
	"testing"

	"github.com/janearc/slayer/manufacture/cities"
)

// Population captured from 9fabd006, before any animal or plant kind
// retirement. The order matters because the PRNG draws by index.
func TestScavengedCandidatePopulationKeepsPreRetirementOrder(t *testing.T) {
	models, err := scavengedModels()
	if err != nil {
		t.Fatal(err)
	}
	names := kindNames()
	for name := range models {
		if slices.Contains(names, name) {
			t.Errorf("retired model %q became a public kind", name)
		}
		names = append(names, name)
	}
	names = slices.DeleteFunc(names, func(name string) bool { return name == ClusterKind })
	slices.Sort(names)
	digest := fmt.Sprintf("%x", sha256.Sum256([]byte(strings.Join(names, "\n"))))
	if len(names) != 282 || digest != "993d9acaaa98393646779329fd70e15dcf1bef5d6a14c69ba6b7585fb78c53cd" {
		t.Fatalf("candidate population %d %s", len(names), digest)
	}
	for _, tc := range []struct {
		seed uint64
		want string
	}{
		{5, "california-barracuda-glide"}, {6, "saguaro"},
		{12, "walrus-fresh-5"}, {15, "bird-grey-heron-standing"},
	} {
		got, err := drawKind(ClusterAsk{}, rand.New(rand.NewPCG(tc.seed, scavengedStream)))
		if err != nil || got != tc.want {
			t.Errorf("seed %d: %q, %v; want %q", tc.seed, got, err, tc.want)
		}
		if _, err := KindNamed(tc.want); err == nil {
			t.Errorf("public kind accepts retired %q", tc.want)
		}
		if _, err := scavengedModel(tc.want); err != nil {
			t.Errorf("private candidate %q: %v", tc.want, err)
		}
	}
	if _, err := KindNamed("can"); err != nil {
		t.Fatalf("ordinary kind: %v", err)
	}
}

func TestSuperblockPrivateDefaultDrawAndExplicitPartRefusal(t *testing.T) {
	// Seed 6's first draw was saguaro in the independently captured table.
	rng := rand.New(rand.NewPCG(6, scavengedStream))
	berth := cities.Volume{Low: [3]float64{-100, 0, -100}, High: [3]float64{100, 100, 100}}
	made, fits, err := (superblockRow{}).partFor(berth, rng, "hull")
	if err != nil || !fits || made.Bounds().Max.Y <= 0 {
		t.Fatalf("private default draw: fits=%t bounds=%v err=%v", fits, made.Bounds(), err)
	}
	row, err := KindNamed(SuperblockKind)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Superblocked(row, SuperblockAsk{Parts: map[string]float64{"saguaro": 1}}); err == nil {
		t.Fatal("explicit retired plant part was accepted")
	}
	if _, err := Superblocked(row, SuperblockAsk{Parts: map[string]float64{"can": 1}}); err != nil {
		t.Fatalf("ordinary authored part: %v", err)
	}
}

func TestScavengedCandidateCollisionsAreRefused(t *testing.T) {
	models := map[string]func() (Kind, error){}
	first := func() (Kind, error) { return Kind{name: "first"}, nil }
	second := func() (Kind, error) { return Kind{name: "second"}, nil }
	if err := addScavengedModel(models, "same", first); err != nil {
		t.Fatal(err)
	}
	if err := addScavengedModel(models, "same", second); err == nil {
		t.Fatal("duplicate private candidate accepted")
	}
	if row, _ := models["same"](); row.name != "first" {
		t.Fatal("duplicate replaced the original candidate")
	}
	if _, err := scavengedCandidateNames([]string{"same", "can"}, models); err == nil {
		t.Fatal("public/private collision accepted")
	}
	names, err := scavengedCandidateNames([]string{ClusterKind, "can"}, models)
	if err != nil || !slices.Equal(names, []string{"can", "same"}) {
		t.Fatalf("distinct candidates %v: %v", names, err)
	}
}
