package static

import (
	"testing"

	"github.com/clarktrimble/apispec"
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

	obj := pkg.Types.Scope().Lookup("Server")
	if obj == nil {
		t.Fatal("Server type not found")
	}

	schema, deps := schemaFrom(obj.Type(), nil)

	if schema.Type != "object" {
		t.Errorf("expected object, got %s", schema.Type)
	}
	if len(schema.Properties) != 2 {
		t.Errorf("expected 2 properties, got %d", len(schema.Properties))
	}

	url := schema.Properties.Get("url")
	if url == nil {
		t.Fatal("missing url property")
	}
	if url.Type != "string" {
		t.Errorf("expected string for url, got %s", url.Type)
	}

	desc := schema.Properties.Get("description")
	if desc == nil {
		t.Fatal("missing description property")
	}

	// url is required (no omitempty), description is not (has omitempty)
	if len(schema.Required) != 1 || schema.Required[0] != "url" {
		t.Errorf("expected required [url], got %v", schema.Required)
	}

	if len(deps) != 0 {
		t.Errorf("expected 0 deps for Server, got %d", len(deps))
	}
}

func TestSchemaFromOperation(t *testing.T) {

	pkg := loadTestPackage(t, "github.com/clarktrimble/apispec")

	obj := pkg.Types.Scope().Lookup("Operation")
	if obj == nil {
		t.Fatal("Operation type not found")
	}

	schema, deps := schemaFrom(obj.Type(), nil)

	if schema.Type != "object" {
		t.Errorf("expected object, got %s", schema.Type)
	}

	// Operation has nested named types: Parameter, RequestBody
	// These should be $ref with deps registered
	params := schema.Properties.Get("parameters")
	if params == nil {
		t.Fatal("missing parameters property")
	}
	if params.Type != "array" {
		t.Errorf("expected array for parameters, got %s", params.Type)
	}

	reqBody := schema.Properties.Get("requestBody")
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

	obj := pkg.Types.Scope().Lookup("Operation")
	if obj == nil {
		t.Fatal("Operation type not found")
	}

	schemas := map[string]*apispec.Schema{}
	schema, discovered := schemaFrom(obj.Type(), nil)
	schemas["Operation"] = schema
	resolveAll(schemas, discovered, nil)

	// Operation -> RequestBody -> MediaType -> Schema (transitive chain)
	for _, name := range []string{"RequestBody", "Parameter"} {
		if schemas[name] == nil {
			t.Errorf("missing direct dep: %s", name)
		}
	}
	for _, name := range []string{"MediaType", "Schema"} {
		if schemas[name] == nil {
			t.Errorf("missing transitive dep: %s", name)
		}
	}

	t.Logf("resolved %d schemas: %v", len(schemas), schemaNames(schemas))
}

func TestConfigSchema(t *testing.T) {

	pkg := loadTestPackage(t, "github.com/clarktrimble/apispec/static/fixture")

	obj := pkg.Types.Scope().Lookup("ServerConfig")
	if obj == nil {
		t.Fatal("ServerConfig type not found")
	}

	schema := configSchemaFrom(obj.Type())

	if schema.Type != "object" {
		t.Errorf("expected object, got %s", schema.Type)
	}

	// "version" has ignored:"true", should be absent
	if schema.Properties.Get("version") != nil {
		t.Error("version should be ignored")
	}

	// "port" has required:"true"
	if len(schema.Required) != 1 || schema.Required[0] != "port" {
		t.Errorf("expected required [port], got %v", schema.Required)
	}

	// "timeout" has default:"10s" which becomes example
	timeout := schema.Properties.Get("timeout")
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
	pkg := loadTestPackage(t, "github.com/clarktrimble/apispec/static/fixture")

	obj := pkg.Types.Scope().Lookup("Widget")
	if obj == nil {
		t.Fatal("Widget type not found")
	}

	schema := configSchemaFrom(obj.Type())

	part := schema.Properties.Get("part")
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

	pkg := loadTestPackage(t, "github.com/clarktrimble/apispec/static/fixture")
	df := newDocFinder(map[string]*packages.Package{
		pkg.PkgPath: pkg,
	})

	// type-level doc comment
	obj := pkg.Types.Scope().Lookup("Widget")
	if obj == nil {
		t.Fatal("Widget type not found")
	}

	schema, _ := schemaFrom(obj.Type(), df)

	if schema.Description != "Widget represents a mechanical component in the inventory." {
		t.Errorf("expected type doc comment, got %q", schema.Description)
	}

	// field-level: "desc" tag wins over doc comment
	name := schema.Properties.Get("name")
	if name == nil {
		t.Fatal("missing name property")
	}
	if name.Description != "widget name" {
		t.Errorf("desc tag should win, got %q", name.Description)
	}

	// field-level: doc comment as fallback when no desc tag
	part := schema.Properties.Get("part")
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

func schemaNames(schemas map[string]*apispec.Schema) []string {
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
