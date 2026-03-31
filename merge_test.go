package apispec_test

import (
	_ "embed"
	"fmt"
	"testing"

	"github.com/clarktrimble/apispec"
)

//go:embed testdata/paths.yaml
var testPaths []byte

type Widget struct {
	Name  string `json:"name" desc:"widget name" example:"sprocket"`
	Count int    `json:"count" desc:"number of widgets"`
	Part  *Part  `json:"part,omitempty"`
}

type Part struct {
	ID    string `json:"id" example:"p-123"`
	Label string `json:"label"`
}

func testSpec() ([]byte, map[string]any) {
	return testPaths, map[string]any{"Widget": Widget{}}
}

func TestMerge(t *testing.T) {

	doc := apispec.NewDocument("test.1.abc1234", "untagged", "http://localhost:3031",
		"Test API")

	err := doc.Merge(testSpec)
	if err != nil {
		t.Fatalf("merge: %v", err)
	}

	// check paths came through
	if len(doc.Paths) != 2 {
		t.Errorf("expected 2 paths, got %d", len(doc.Paths))
	}
	if doc.Paths.Get("/widgets") == nil {
		t.Error("missing /widgets")
	}
	if doc.Paths.Get("/widgets/{id}") == nil {
		t.Error("missing /widgets/{id}")
	}

	// check tags merged from fragment
	found := false
	for _, tag := range doc.Tags {
		if tag.Name == "widgets" {
			found = true
		}
	}
	if !found {
		t.Error("missing widgets tag")
	}

	// check schema generated from type map
	if doc.Components.Schemas["Widget"] == nil {
		t.Error("missing Widget schema")
	}
	if doc.Components.Schemas["Widget"].Type != "object" {
		t.Errorf("expected Widget type object, got %s", doc.Components.Schemas["Widget"].Type)
	}

	// check transitive dep resolved
	if doc.Components.Schemas["Part"] == nil {
		t.Error("missing Part schema (transitive dep)")
	}

	// check Error always present
	if doc.Components.Schemas["Error"] == nil {
		t.Error("missing Error schema")
	}

	data, err := doc.Marshal()
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	fmt.Println(string(data))
}
