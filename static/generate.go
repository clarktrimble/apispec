package static

import (
	"go/types"
	"os"
	"strings"

	"github.com/clarktrimble/apispec"
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

	doc := apispec.Document{
		OpenAPI: apispec.OpenAPIVersion,
		Info: apispec.Info{
			Title:       cfg.Title,
			Version:     "${RELEASE}",
			Description: strings.Join(cfg.Description, "\n\n"),
		},
		Servers: []apispec.Server{
			{URL: "${PUBLISHED_URL}", Description: "API server"},
		},
	}

	schemas := map[string]schemaEntry{}
	df := newDocFinder(pkgs)

	for _, spec := range cfg.Specs {
		pkg, ok := pkgs[spec.Package]
		if !ok {
			return errors.Errorf("package %s not loaded", spec.Package)
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

		for _, typeName := range spec.Types {
			obj := pkg.Types.Scope().Lookup(typeName)
			if obj == nil {
				return errors.Errorf("type %s not found in %s", typeName, spec.Package)
			}
			named, ok := obj.Type().(*types.Named)
			if !ok {
				return errors.Errorf("type %s in %s is not a named type", typeName, spec.Package)
			}
			if existing, exists := schemas[typeName]; exists {
				if existing.source != named {
					return errors.Errorf("schema name collision: %q from %s and %s",
						typeName, existing.source.Obj().Pkg().Path(), spec.Package)
				}
				continue
			}
			schema, discovered := schemaFrom(obj.Type(), df)
			schemas[typeName] = schemaEntry{schema: schema, source: named}
			if err := resolveAll(schemas, discovered, df); err != nil {
				return err
			}
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

	schemas["Error"] = schemaEntry{schema: &apispec.Schema{
		Type: "object",
		Properties: apispec.Properties{
			{Name: "error", Schema: &apispec.Schema{Type: "string", Description: "Error message"}},
		},
	}}

	componentSchemas := map[string]*apispec.Schema{}
	for name, entry := range schemas {
		componentSchemas[name] = entry.schema
	}
	doc.Components = &apispec.Components{Schemas: componentSchemas}

	data, err := yaml.Marshal(doc)
	if err != nil {
		return errors.Wrap(err, "marshaling document")
	}

	return os.WriteFile(outPath, data, 0o644)
}

func loadConfig(path string) (Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, errors.Wrap(err, "reading config")
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, errors.Wrap(err, "parsing config")
	}
	return cfg, nil
}

func loadPackages(cfg Config) (map[string]*packages.Package, error) {

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
