package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"
)

func TestResourcesRetainsDetailAndReportsFailures(t *testing.T) {
	good := measureResources("can", "quick")
	if good.Error != "" || good.Parts == 0 || good.Geometry.StoredFaces == 0 || good.Geometry.OwnedGeometryBytes == 0 || good.Allocated == 0 {
		t.Fatalf("missing model ownership: %+v", good)
	}
	bad := measureResources("not-a-kind", "demo")
	if bad.Error == "" || bad.Kind != "not-a-kind" || bad.Detail != "demo" {
		t.Fatal("failed model disappeared", bad)
	}
	var out, failed bytes.Buffer
	if tool(context.Background()).Run([]string{"resources", "can"}, &out, &failed) != 0 {
		t.Fatal(failed.String())
	}
	var row resourceRow
	if err := json.Unmarshal(out.Bytes(), &row); err != nil {
		t.Fatal(err)
	}
	if row.Kind != "can" || row.Detail != "quick" {
		t.Fatal("changed requested geometry", out.String())
	}
}

func Example_resources() {
	var out, failed bytes.Buffer
	tool(context.Background()).Run([]string{"resources", "can"}, &out, &failed)
	var row resourceRow
	if err := json.Unmarshal(out.Bytes(), &row); err != nil {
		panic(err)
	}
	fmt.Println(row.Kind, row.Geometry.UniqueMeshes > 0)
	// Output: can true
}
