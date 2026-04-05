package apispec

import (
	"go/types"
	"testing"

	"golang.org/x/tools/go/packages"
)

func loadTestPackage(t *testing.T, path string) *packages.Package {
	t.Helper()

	cfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedSyntax | packages.NeedTypesInfo |
			packages.NeedName | packages.NeedFiles,
	}
	pkgs, err := packages.Load(cfg, path)
	if err != nil {
		t.Fatalf("loading package: %v", err)
	}
	if len(pkgs) == 0 {
		t.Fatal("no packages loaded")
	}
	if len(pkgs[0].Errors) > 0 {
		t.Fatalf("package errors: %v", pkgs[0].Errors)
	}
	return pkgs[0]
}

func TestSchemaFromServer(t *testing.T) {

	pkg := loadTestPackage(t, "github.com/clarktrimble/apispec")

	obj := pkg.Types.Scope().Lookup("server")
	if obj == nil {
		t.Fatal("server type not found")
	}

	s, deps := schemaFrom(obj.Type(), nil)

	if s.Type != "object" {
		t.Errorf("expected object, got %s", s.Type)
	}
	if len(s.Properties) != 2 {
		t.Errorf("expected 2 properties, got %d", len(s.Properties))
	}

	url := s.Properties.Get("url")
	if url == nil {
		t.Fatal("missing url property")
	}
	if url.Type != "string" {
		t.Errorf("expected string for url, got %s", url.Type)
	}

	desc := s.Properties.Get("description")
	if desc == nil {
		t.Fatal("missing description property")
	}

	// url is required (no omitempty), description is not (has omitempty)
	if len(s.Required) != 1 || s.Required[0] != "url" {
		t.Errorf("expected required [url], got %v", s.Required)
	}

	if len(deps) != 0 {
		t.Errorf("expected 0 deps for Server, got %d", len(deps))
	}
}

func TestSchemaFromOperation(t *testing.T) {

	pkg := loadTestPackage(t, "github.com/clarktrimble/apispec")

	obj := pkg.Types.Scope().Lookup("operation")
	if obj == nil {
		t.Fatal("operation type not found")
	}

	s, deps := schemaFrom(obj.Type(), nil)

	if s.Type != "object" {
		t.Errorf("expected object, got %s", s.Type)
	}

	// Operation has nested named types: Parameter, RequestBody
	// These should be $ref with deps registered
	params := s.Properties.Get("parameters")
	if params == nil {
		t.Fatal("missing parameters property")
	}
	if params.Type != "array" {
		t.Errorf("expected array for parameters, got %s", params.Type)
	}

	reqBody := s.Properties.Get("requestBody")
	if reqBody == nil {
		t.Fatal("missing requestBody property")
	}
	if reqBody.Ref == "" {
		t.Error("expected $ref for requestBody")
	}

	if len(deps) == 0 {
		t.Error("expected deps for Operation (Parameter, RequestBody, etc.)")
	}
	t.Logf("deps: %v", depsNames(deps))
}

func TestResolveAll(t *testing.T) {

	pkg := loadTestPackage(t, "github.com/clarktrimble/apispec")

	obj := pkg.Types.Scope().Lookup("operation")
	if obj == nil {
		t.Fatal("operation type not found")
	}

	named, ok := obj.Type().(*types.Named)
	if !ok {
		t.Fatal("operation is not a named type")
	}
	schemas := map[string]schemaEntry{}
	s, discovered := schemaFrom(obj.Type(), nil)
	schemas["operation"] = schemaEntry{schema: s, source: named}
	if err := resolveAll(schemas, discovered, nil); err != nil {
		t.Fatalf("resolveAll: %v", err)
	}

	// operation -> requestBody -> mediaType -> schema (transitive chain)
	for _, name := range []string{"requestBody", "parameter"} {
		if schemas[name].schema == nil {
			t.Errorf("missing direct dep: %s", name)
		}
	}
	for _, name := range []string{"mediaType", "schema"} {
		if schemas[name].schema == nil {
			t.Errorf("missing transitive dep: %s", name)
		}
	}

	t.Logf("resolved %d schemas: %v", len(schemas), entryNames(schemas))
}

func TestConfigSchema(t *testing.T) {

	pkg := loadTestPackage(t, "github.com/clarktrimble/apispec/testdata/fixture")

	obj := pkg.Types.Scope().Lookup("ServerConfig")
	if obj == nil {
		t.Fatal("ServerConfig type not found")
	}

	s := configSchemaFrom(obj.Type())

	if s.Type != "object" {
		t.Errorf("expected object, got %s", s.Type)
	}

	// "version" has ignored:"true", should be absent
	if s.Properties.Get("version") != nil {
		t.Error("version should be ignored")
	}

	// "port" has required:"true"
	if len(s.Required) != 1 || s.Required[0] != "port" {
		t.Errorf("expected required [port], got %v", s.Required)
	}

	// "timeout" has default:"10s" which becomes example
	timeout := s.Properties.Get("timeout")
	if timeout == nil {
		t.Fatal("missing timeout property")
	}
	if timeout.Example != "10s" {
		t.Errorf("expected example '10s', got %v", timeout.Example)
	}
	if timeout.Description != "request timeout" {
		t.Errorf("expected desc 'request timeout', got %s", timeout.Description)
	}

	// timeout is time.Duration, should be inlined as string (not $ref)
	if timeout.Type != "string" {
		t.Errorf("expected string for Duration, got %s", timeout.Type)
	}
}

func TestConfigSchemaInlinesNested(t *testing.T) {

	// Widget has a *Part field — in config mode it should be inlined, not $ref
	pkg := loadTestPackage(t, "github.com/clarktrimble/apispec/testdata/fixture")

	obj := pkg.Types.Scope().Lookup("Widget")
	if obj == nil {
		t.Fatal("Widget type not found")
	}

	s := configSchemaFrom(obj.Type())

	part := s.Properties.Get("part")
	if part == nil {
		t.Fatal("missing part property")
	}
	if part.Ref != "" {
		t.Errorf("config mode should inline, got $ref: %s", part.Ref)
	}
	if part.Type != "object" {
		t.Errorf("expected inlined object for Part, got %s", part.Type)
	}
	if len(part.Properties) != 2 {
		t.Errorf("expected 2 properties on inlined Part, got %d", len(part.Properties))
	}
}

func TestDocComments(t *testing.T) {

	pkg := loadTestPackage(t, "github.com/clarktrimble/apispec/testdata/fixture")
	df := newDocFinder(map[string]*packages.Package{
		pkg.PkgPath: pkg,
	})

	// type-level doc comment
	obj := pkg.Types.Scope().Lookup("Widget")
	if obj == nil {
		t.Fatal("Widget type not found")
	}

	s, _ := schemaFrom(obj.Type(), df)

	if s.Description != "Widget represents a mechanical component in the inventory." {
		t.Errorf("expected type doc comment, got %q", s.Description)
	}

	// field-level: "desc" tag wins over doc comment
	name := s.Properties.Get("name")
	if name == nil {
		t.Fatal("missing name property")
	}
	if name.Description != "widget name" {
		t.Errorf("desc tag should win, got %q", name.Description)
	}

	// field-level: doc comment as fallback when no desc tag
	part := s.Properties.Get("part")
	if part == nil {
		t.Fatal("missing part property")
	}
	// part is a $ref so the comment lands on the ref schema...
	// but let's check Part.Label which has a doc comment and no desc tag
	partObj := pkg.Types.Scope().Lookup("Part")
	if partObj == nil {
		t.Fatal("Part type not found")
	}
	partSchema, _ := schemaFrom(partObj.Type(), df)

	label := partSchema.Properties.Get("label")
	if label == nil {
		t.Fatal("missing label property")
	}
	if label.Description != "Human-readable label for the part." {
		t.Errorf("expected doc comment fallback, got %q", label.Description)
	}

	// Part type-level doc
	if partSchema.Description != "Part is a sub-component of a widget." {
		t.Errorf("expected Part type doc, got %q", partSchema.Description)
	}
}

func entryNames(schemas map[string]schemaEntry) []string {
	names := make([]string, 0, len(schemas))
	for name := range schemas {
		names = append(names, name)
	}
	return names
}

func depsNames(d deps) []string {
	names := make([]string, 0, len(d))
	for name := range d {
		names = append(names, name)
	}
	return names
}
