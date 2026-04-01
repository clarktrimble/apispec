package static

import (
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

	schemas := map[string]*apispec.Schema{}
	df := newDocFinder(pkgs)

	for _, spec := range cfg.Specs {
		pkg, ok := pkgs[spec.Package]
		if !ok {
			return errors.Errorf("package %s not loaded", spec.Package)
		}

		paths, tags, err := loadFragment(pkg)
		if err != nil {
			return err
		}
		// Todo: detect duplicate paths (runtime mergePaths errors on dupes)
		doc.Paths = append(doc.Paths, paths...)
		doc.Tags = append(doc.Tags, tags...)

		for _, typeName := range spec.Types {
			obj := pkg.Types.Scope().Lookup(typeName)
			if obj == nil {
				return errors.Errorf("type %s not found in %s", typeName, spec.Package)
			}
			schema, discovered := schemaFrom(obj.Type(), df)
			schemas[typeName] = schema
			resolveAll(schemas, discovered, df)
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
		schemas[name] = configSchemaFrom(obj.Type())
	}

	schemas["Error"] = &apispec.Schema{
		Type: "object",
		Properties: apispec.Properties{
			{Name: "error", Schema: &apispec.Schema{Type: "string", Description: "Error message"}},
		},
	}

	doc.Components = &apispec.Components{Schemas: schemas}

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
