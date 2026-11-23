package gpu

import (
	"fmt"
	"math/bits"
	"os"
	"regexp"
	"strconv"
	"strings"
	"testing"

	"github.com/janearc/slayer/emit"
	"github.com/janearc/slayer/maths"
	"github.com/janearc/slayer/scene/thing"
)

// rowLamps is how many lamps the depth check lines up: far more than a
// street, so a walk that took a step for every lamp would show.
const rowLamps = 1000

// kernelText is the kernel's source, read from its file.
func kernelText(t *testing.T) string {
	t.Helper()
	source, err := os.ReadFile("trace.metal")
	if err != nil {
		t.Fatal(err)
	}
	return string(source)
}

// kernelBody is the text of the kernel's static function of that name:
// from its signature to the first closing brace in the first column. It
// says whether there is such a function.
func kernelBody(text, name string) (string, bool) {
	signature := regexp.MustCompile(`(?m)^static [^\n(]* ` +
		regexp.QuoteMeta(name) + `\(`)
	found := signature.FindStringIndex(text)
	if found == nil {
		return "", false
	}
	end := strings.Index(text[found[0]:], "\n}\n")
	if end < 0 {
		return "", false
	}
	return text[found[0] : found[0]+end], true
}

// doctored is the text with its one place of old made new, which is a
// fault put in on purpose; it stops the test if old is not there.
func doctored(t *testing.T, text, old, now string) string {
	t.Helper()
	if !strings.Contains(text, old) {
		t.Fatalf("the kernel's text lacks %q to doctor", old)
	}
	return strings.Replace(text, old, now, 1)
}

// lampRow is count lamps in a row, each lit and standing somewhere, so
// that every one of them is a leaf of the lamp tree.
func lampRow(count int) []thing.Lamp {
	var lamps []thing.Lamp
	for each := range count {
		at := maths.V(float64(each), 1, 0)
		lamps = append(lamps, emit.Lit{At: at,
			Strength: maths.Grey(1)})
	}
	return lamps
}

// crowdDepth is the most forks a walk down a crowd's tree from the node
// at passes before it reaches a leaf.
func crowdDepth(crowd lampCrowd, at int32) int {
	node := crowd.nodes[at]
	if node.leaf() {
		return 0
	}
	return 1 + max(crowdDepth(crowd, node.Left),
		crowdDepth(crowd, node.Right))
}

// chainCrowd is a tree of that many forks one below the other, each with
// a leaf on its right: as deep as a tree of that many lamps can be.
func chainCrowd(forks int) lampCrowd {
	nodes := make([]lightNode, 2*forks+1)
	for fork := range forks {
		nodes[fork] = lightNode{Left: int32(fork + 1),
			Right: int32(forks + 1 + fork), Lamp: -1}
	}
	for leaf := forks; leaf < len(nodes); leaf++ {
		nodes[leaf] = lightNode{Lamp: int32(leaf - forks)}
	}
	return lampCrowd{nodes: nodes}
}

// treeFaults is what is wrong with a crowd's tree for a kernel whose walk
// takes allowed forks: deeper than the halvings of its lamps, or as deep
// as the walk is allowed.
func treeFaults(crowd lampCrowd, allowed int) []string {
	var faults []string
	standing := (len(crowd.nodes) + 1) / 2
	depth := crowdDepth(crowd, 0)
	if halving := bits.Len(uint(standing - 1)); depth > halving {
		faults = append(faults, fmt.Sprintf("a tree of %d lamps is %d "+
			"deep, more than the %d halvings", standing, depth, halving))
	}
	if depth >= allowed {
		faults = append(faults, fmt.Sprintf("a tree %d deep, and the "+
			"kernel's walk takes %d forks", depth, allowed))
	}
	return faults
}

// treeDepthPattern finds the most forks the kernel's walk takes.
var treeDepthPattern = regexp.MustCompile(`lamp_tree_depth = (\d+);`)

// kernelTreeDepth is the most forks the kernel's walk down the lamp tree
// takes before it gives the walk up, as the kernel states it.
func kernelTreeDepth(t *testing.T, text string) int {
	t.Helper()
	found := treeDepthPattern.FindStringSubmatch(text)
	if found == nil {
		t.Fatal("the kernel states no depth for the lamp tree's walk")
	}
	depth, err := strconv.Atoi(found[1])
	if err != nil {
		t.Fatal(err)
	}
	return depth
}

// A walk down the lamp tree takes one fork for each halving of the lamps,
// not one for each lamp, and the kernel's walk is allowed more forks than
// that: a tree deeper than the walk's bound would be given up on.
func TestALampTreeFitsTheKernelsWalkInHalvings(t *testing.T) {
	allowed := kernelTreeDepth(t, kernelText(t))
	for name, lamps := range map[string][]thing.Lamp{
		"a street": crowdStreet(t), "a row": lampRow(rowLamps)} {
		for _, fault := range treeFaults(lampCrowdOf(lamps), allowed) {
			t.Errorf("%s: %s", name, fault)
		}
	}
}

// Anti-test: a tree as deep as the kernel's walk is allowed is reported,
// as too deep for the walk and as deeper than halving.
func TestATreeAsDeepAsTheKernelsWalkIsReported(t *testing.T) {
	allowed := kernelTreeDepth(t, kernelText(t))
	if faults := treeFaults(chainCrowd(allowed), allowed); len(faults) != 2 {
		t.Errorf("a chain of %d forks gave %v, want two faults", allowed,
			faults)
	}
}

// pickFaults is what is wrong with how the kernel's text picks lamps from
// the tree: asked giving the tree only the worlds above lamps_every, the
// tree's picks by a walk, and the walk bounded in its forks.
func pickFaults(text string) []string {
	var faults []string
	asked, _ := kernelBody(text, "asked")
	every := strings.Index(asked, "lamps <= lamps_every")
	treed := strings.Index(asked, "asked_treed(")
	if every < 0 || treed < every ||
		!strings.Contains(asked, "lamp_tree.x > 0") {
		faults = append(faults, "asked does not give a world of many "+
			"lamps to the tree after letting lamps_every lamps keep "+
			"their loop")
	}
	if body, _ := kernelBody(text, "asked_treed"); !strings.Contains(body,
		"lamp_tree_walk(") {
		faults = append(faults, "asked_treed does not walk the tree")
	}
	if body, _ := kernelBody(text, "lamp_tree_walk"); !strings.Contains(
		body, "depth < lamp_tree_depth") {
		faults = append(faults, "the walk is not bounded in its forks")
	}
	return faults
}

// The kernel picks lamps from the tree only above lamps_every lamps, and
// then by a walk bounded in its forks, in place of weighing every lamp.
// This reads the text, so a rename shows here.
func TestTheKernelPicksFromTheTreeOnlyAboveLampsEvery(t *testing.T) {
	for _, fault := range pickFaults(kernelText(t)) {
		t.Error(fault)
	}
}

// Anti-test: a walk that is not bounded is reported, and so is an asked
// that has no loop for the worlds of few lamps.
func TestAWalkWithNoBoundIsReported(t *testing.T) {
	text := kernelText(t)
	unbounded := doctored(t, text, "depth < lamp_tree_depth", "true")
	if faults := pickFaults(unbounded); len(faults) != 1 {
		t.Errorf("a walk with no bound gave %v, want one fault", faults)
	}
	early := doctored(t, text, "lamps <= lamps_every", "lamps < 0")
	if len(pickFaults(early)) == 0 {
		t.Error("asked with no loop for few lamps is not reported")
	}
}

// listsOf is each loop of the kernel that walks a list and the list it
// walks.
var listsOf = map[string]string{
	"flooded":              "flood_list",
	"clear_single_scatter": "cone_list",
	"beam_march_boxes":     "beam_list",
	"beam_march_apart":     "beam_list",
}

// loopFaults is each loop of the kernel's text that does not walk its
// list, or reads the world's count of lamps.
func loopFaults(text string) []string {
	var faults []string
	for name, list := range listsOf {
		body, found := kernelBody(text, name)
		if !found || !strings.Contains(body, "lamp_list(world, "+list+")") {
			faults = append(faults, name+" does not walk the "+list)
		}
		if strings.Contains(body, "counts.w") {
			faults = append(faults, name+" reads the count of every lamp")
		}
	}
	return faults
}

// The three loops that used to walk every lamp and skip most of them walk
// the list of the lamps they keep, and no loop of them reads the world's
// count of lamps. This reads the text, so a rename shows here.
func TestTheKernelLoopsWalkTheirListsAndNotEveryLamp(t *testing.T) {
	for _, fault := range loopFaults(kernelText(t)) {
		t.Error(fault)
	}
}

// Anti-test: a flood loop that walks every lamp again is reported, and
// only it.
func TestALoopThatWalksEveryLampIsReported(t *testing.T) {
	again := doctored(t, kernelText(t), "int floods = world.header->"+
		"lamp_lists.x;", "int floods = world.header->counts.w;")
	faults := loopFaults(again)
	if len(faults) != 1 || !strings.HasPrefix(faults[0], "flooded ") {
		t.Errorf("a flood loop over every lamp gave %v", faults)
	}
}

// boxFaults is what is wrong with how the kernel's text lets a shadow ray
// ask a volume's box: clear asks it before the volume's spans, and a
// volume with no box is missed by no ray.
func boxFaults(text string) []string {
	var faults []string
	shadow, _ := kernelBody(text, "clear")
	missed := strings.Index(shadow, "volume_missed(")
	marched := strings.Index(shadow, "spans(")
	if missed < 0 || marched < missed {
		faults = append(faults, "clear does not ask a volume's box "+
			"before its spans")
	}
	if body, _ := kernelBody(text, "volume_missed"); !strings.Contains(body,
		"world_low.w == 0) { return false; }") {
		faults = append(faults, "a volume with no box is missed by a ray")
	}
	return faults
}

// A shadow ray asks a volume's box in the world before it marches the
// volume, and a volume with no box is missed by no ray. This reads the
// text, so a rename shows here.
func TestAShadowRayAsksAVolumesBoxBeforeItMarches(t *testing.T) {
	for _, fault := range boxFaults(kernelText(t)) {
		t.Error(fault)
	}
}

// Anti-test: a shadow ray that never asks the box is reported.
func TestAShadowRayThatNeverAsksTheBoxIsReported(t *testing.T) {
	skipped := doctored(t, kernelText(t), "if (volume_missed(volume, "+
		"point, toward, far)) { continue; }", "")
	if faults := boxFaults(skipped); len(faults) != 1 {
		t.Errorf("a shadow ray with no box check gave %v, want one fault",
			faults)
	}
}

// bendFaults is what is wrong with how the kernel's text reads a bend's
// place: each read by bend_of, none by value, and bend_of taking away the
// bias bendNumber adds.
func bendFaults(text string) []string {
	var faults []string
	bias := fmt.Sprintf("0x%08x", bendBias)
	if body, _ := kernelBody(text, "bend_of"); !strings.Contains(body,
		bias) {
		faults = append(faults, "bend_of does not take away "+bias)
	}
	for _, read := range []string{"met", "face"} {
		place := read + ".corner_a.w)"
		if !strings.Contains(text, "bend_of("+place) {
			faults = append(faults, "a place read of "+read+" is not "+
				"by bend_of")
		}
		if strings.Contains(text, "int("+place) {
			faults = append(faults, "a place read of "+read+" is by value")
		}
	}
	return faults
}

// The kernel reads a bend's place from its float's bits, less the bias
// bendNumber adds, at each place it reads one: a value read as a whole
// number would lose the places above two to the 24th.
func TestTheKernelReadsABendsPlaceFromItsBits(t *testing.T) {
	for _, fault := range bendFaults(kernelText(t)) {
		t.Error(fault)
	}
}

// Anti-test: a place read by value is reported, as not by bend_of and as
// by value.
func TestABendsPlaceReadByValueIsReported(t *testing.T) {
	byValue := doctored(t, kernelText(t), "bend_of(met.corner_a.w)",
		"int(met.corner_a.w)")
	if faults := bendFaults(byValue); len(faults) != 2 {
		t.Errorf("a place read by value gave %v, want two faults", faults)
	}
}
