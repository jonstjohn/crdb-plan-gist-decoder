package gistdecoder

import (
	"strings"
	"testing"
)

func TestDecodePlanGist(t *testing.T) {
	// This is a real gist from CockroachDB representing:
	// UPDATE ... SET ... (with render and scan)
	gist := "AgHgAQIA/wMCAAAHFAUUIeABAAAFDAYM"

	node, err := DecodePlanGist(gist, nil, nil)
	if err != nil {
		t.Fatalf("Failed to decode gist: %v", err)
	}

	if node == nil {
		t.Fatal("Expected non-nil node")
	}

	if node.op != updateOp {
		t.Errorf("Expected root to be updateOp, got %v", node.op)
	}

	// Should have one child (the render node)
	if len(node.children) != 1 {
		t.Errorf("Expected 1 child, got %d", len(node.children))
	}

	// Check that table info was decoded
	if _, ok := node.args["table"]; !ok {
		t.Error("Expected table argument in update node")
	}
}

func TestDecodePlanGistInvalidBase64(t *testing.T) {
	_, err := DecodePlanGist("not-valid-base64!", nil, nil)
	if err == nil {
		t.Error("Expected error for invalid base64")
	}
	if !strings.Contains(err.Error(), "base64") {
		t.Errorf("Expected base64 error, got: %v", err)
	}
}

func TestDecodePlanGistWithLookup(t *testing.T) {
	gist := "AgHgAQIA/wMCAAAHFAUUIeABAAAFDAYM"

	tableLookup := func(id int64) string {
		if id == 112 {
			return "users"
		}
		return ""
	}

	indexLookup := func(tableID int64, indexID int64) string {
		if tableID == 112 && indexID == 1 {
			return "users_pkey"
		}
		return ""
	}

	node, err := DecodePlanGist(gist, tableLookup, indexLookup)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	// Navigate to the scan node (update -> simple project -> render -> scan)
	if len(node.children) == 0 {
		t.Fatal("Expected children in update node")
	}

	// The actual tree has a simple project node (which is skipped in formatted output)
	projectNode := node.children[0]
	if len(projectNode.children) == 0 {
		t.Fatal("Expected children in project node")
	}

	renderNode := projectNode.children[0]
	if len(renderNode.children) == 0 {
		t.Fatal("Expected children in render node")
	}

	scanNode := renderNode.children[0]
	if scanNode.op != scanOp {
		t.Errorf("Expected scan node, got %v", scanNode.op)
	}

	// Check that custom lookup was used
	table, ok := scanNode.args["table"]
	if !ok {
		t.Fatal("Expected table arg in scan node")
	}

	if table != "users" {
		t.Errorf("Expected table name 'users', got '%v'", table)
	}

	index, ok := scanNode.args["index"]
	if !ok {
		t.Fatal("Expected index arg in scan node")
	}

	if index != "users_pkey" {
		t.Errorf("Expected index name 'users_pkey', got '%v'", index)
	}
}

func TestFormatPlan(t *testing.T) {
	gist := "AgHgAQIA/wMCAAAHFAUUIeABAAAFDAYM"

	node, err := DecodePlanGist(gist, nil, nil)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	output := FormatPlan(node)

	// Check for expected output elements
	expectedStrings := []string{
		"• update",
		"table: 112",
		"set",
		"• render",
		"• scan",
		"table: 112@1",
		"spans: 1+ spans",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected output to contain '%s', but it didn't.\nOutput:\n%s", expected, output)
		}
	}

	// Check for tree characters
	if !strings.Contains(output, "│") {
		t.Error("Expected output to contain vertical tree characters (│)")
	}

	if !strings.Contains(output, "└──") {
		t.Error("Expected output to contain tree connectors (└──)")
	}
}

func TestFormatPlanNilNode(t *testing.T) {
	output := FormatPlan(nil)
	if output != "" {
		t.Errorf("Expected empty string for nil node, got: %s", output)
	}
}

func BenchmarkDecodePlanGist(b *testing.B) {
	gist := "AgHgAQIA/wMCAAAHFAUUIeABAAAFDAYM"

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := DecodePlanGist(gist, nil, nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkFormatPlan(b *testing.B) {
	gist := "AgHgAQIA/wMCAAAHFAUUIeABAAAFDAYM"
	node, _ := DecodePlanGist(gist, nil, nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = FormatPlan(node)
	}
}
