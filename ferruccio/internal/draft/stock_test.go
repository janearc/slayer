package draft

import (
	"strings"
	"testing"
)

func TestJobStockEditPreservesParametersAndRefusesPrintConflict(t *testing.T) {
	source := string(exampleSource(t, "first-picture.yaml")) + "stock: {name: dusk, seed: 7, rated: -1} # Keep development\n"
	doc, err := Open("job.yaml", []byte(source))
	if err != nil {
		t.Fatal(err)
	}
	if name, ok := doc.StockChoice(); !ok || name != "dusk" {
		t.Fatalf("stock %q %v", name, ok)
	}
	if err := doc.SetStock("invented"); err == nil || doc.Text() != source {
		t.Fatal("unknown stock changed source")
	}
	if err := doc.SetStock("ash"); err != nil {
		t.Fatal(err)
	}
	if name, ok := doc.StockChoice(); !ok || name != "ash" || !strings.Contains(doc.Text(), "seed: 7") || !strings.Contains(doc.Text(), "rated: -1") || !strings.Contains(doc.Text(), "Keep development") {
		t.Fatalf("stock %q %v", name, ok)
	}
	printed := string(exampleSource(t, "first-picture.yaml")) + "print: {knee: 0.5, top: 8}\n"
	doc, err = Open("job.yaml", []byte(printed))
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := doc.StockChoice(); ok {
		t.Fatal("print conflict offered stock")
	}
	if err := doc.SetStock("ash"); err == nil || doc.Text() != printed {
		t.Fatal("print curve overwritten")
	}
}
