package apispec

import (
	"go/types"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	"go.yaml.in/yaml/v3"
	"golang.org/x/tools/go/packages"
)

// Generate reads a config file, loads the referenced packages and types,
// and writes a complete OpenAPI document.
func Generate(cfgPath, outPath string) error {

	cfg, err := loadConfig(cfgPath)
	if err != nil {
		return err
	}

	pkgs, err := loadPackages(cfg)
	if err != nil {
		return err
	}

	doc := document{
		OpenAPI: openAPIVersion,
		Info: info{
			Title:       cfg.Title,
			Version:     "${RELEASE}",
			Description: strings.Join(cfg.Description, "\n\n"),
		},
		Servers: []server{
			{URL: "${PUBLISHED_URL}", Description: "API server"},
		},
	}

	schemas := map[string]schemaEntry{}
	df := newDocFinder(pkgs)

	for _, sp := range cfg.Specs {
		pkg, ok := pkgs[sp.Package]
		if !ok {
			return errors.Errorf("package %s not loaded", sp.Package)
		}

		paths, err := loadFragment(pkg)
		if err != nil {
			return err
		}
		for _, kv := range paths {
			if doc.Paths.Has(kv.Key) {
				return errors.Errorf("duplicate path: %s", kv.Key)
			}
			doc.Paths = append(doc.Paths, kv)
		}

		if err := resolveSpecTypes(sp, pkg, schemas, df); err != nil {
			return err
		}
	}

	if cfg.Config != nil {
		pkg, ok := pkgs[cfg.Config.Package]
		if !ok {
			return errors.Errorf("config package %s not loaded", cfg.Config.Package)
		}
		obj := pkg.Types.Scope().Lookup(cfg.Config.Type)
		if obj == nil {
			return errors.Errorf("config type %s not found in %s", cfg.Config.Type, cfg.Config.Package)
		}
		name := cfg.Config.Name
		if name == "" {
			name = cfg.Config.Type
		}
		schemas[name] = schemaEntry{schema: configSchemaFrom(obj.Type())}
	}

	schemas["Error"] = schemaEntry{schema: &schema{
		Type: "object",
		Properties: properties{
			{Name: "error", Schema: &schema{Type: "string", Description: "Error message"}},
		},
	}}

	componentSchemas := map[string]*schema{}
	for name, entry := range schemas {
		componentSchemas[name] = entry.schema
	}
	doc.Components = &components{Schemas: componentSchemas}

	data, err := yaml.Marshal(doc)
	if err != nil {
		return errors.Wrap(err, "marshaling document")
	}

	return os.WriteFile(outPath, data, 0o644) //nolint:gosec // spec file should be world-readable
}

func loadConfig(path string) (config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return config{}, errors.Wrap(err, "reading config")
	}

	var cfg config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return config{}, errors.Wrap(err, "parsing config")
	}
	return cfg, nil
}

func loadPackages(cfg config) (map[string]*packages.Package, error) {

	patterns := make([]string, 0, len(cfg.Specs)+1)
	for _, s := range cfg.Specs {
		patterns = append(patterns, s.Package)
	}
	if cfg.Config != nil && cfg.Config.Package != "" {
		patterns = append(patterns, cfg.Config.Package)
	}

	pkgCfg := &packages.Config{
		Mode: packages.NeedTypes | packages.NeedSyntax |
			packages.NeedTypesInfo | packages.NeedName | packages.NeedFiles,
	}

	pkgs, err := packages.Load(pkgCfg, patterns...)
	if err != nil {
		return nil, errors.Wrap(err, "loading packages")
	}

	result := map[string]*packages.Package{}
	for _, pkg := range pkgs {
		if len(pkg.Errors) > 0 {
			return nil, errors.Errorf("package %s: %v", pkg.PkgPath, pkg.Errors[0])
		}
		result[pkg.PkgPath] = pkg
	}

	return result, nil
}

func resolveSpecTypes(sp spec, pkg *packages.Package, schemas map[string]schemaEntry, df *docFinder) error {

	for _, typeName := range sp.Types {
		obj := pkg.Types.Scope().Lookup(typeName)
		if obj == nil {
			return errors.Errorf("type %s not found in %s", typeName, sp.Package)
		}
		named, ok := obj.Type().(*types.Named)
		if !ok {
			return errors.Errorf("type %s in %s is not a named type", typeName, sp.Package)
		}
		if existing, exists := schemas[typeName]; exists {
			if existing.source != named {
				return errors.Errorf("schema name collision: %q from %s and %s",
					typeName, existing.source.Obj().Pkg().Path(), sp.Package)
			}
			continue
		}
		schema, discovered := schemaFrom(obj.Type(), df)
		schemas[typeName] = schemaEntry{schema: schema, source: named}
		if err := resolveAll(schemas, discovered, df); err != nil {
			return err
		}
	}

	return nil
}

type fragment struct {
	Paths paths `yaml:"paths"`
}

// loadFragment finds and parses paths.yaml in a package's directory.
func loadFragment(pkg *packages.Package) (paths, error) {

	if len(pkg.GoFiles) == 0 {
		return nil, errors.Errorf("package %s has no Go files", pkg.PkgPath)
	}

	dir := filepath.Dir(pkg.GoFiles[0])
	path := filepath.Join(dir, "paths.yaml")

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, errors.Wrapf(err, "reading %s", path)
	}

	var f fragment
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, errors.Wrapf(err, "parsing %s", path)
	}

	return f.Paths, nil
}
