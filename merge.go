package apispec

import (
	"encoding/json"
	"fmt"

	"github.com/pkg/errors"
	"go.yaml.in/yaml/v3"
)

// SpecFunc provides an API spec fragment: YAML paths and Go types for schema generation.
type SpecFunc func() ([]byte, map[string]any)

// Merge assembles spec contributors into the document.
// Each contributor provides YAML with a top-level "paths" key and a type map
// for schema generation. Merge adds an Error schema and deduplicates by key.
func (doc *Document) Merge(specFuncs ...SpecFunc) error {

	// Clone shared slices/maps to avoid mutating caller's data.
	doc.Paths = append(Paths{}, doc.Paths...)

	schemas := map[string]*Schema{}
	if doc.Components != nil {
		for k, v := range doc.Components.Schemas {
			schemas[k] = v
		}
	}
	doc.Components = &Components{Schemas: schemas}

	for _, fn := range specFuncs {
		yamlBytes, types := fn()

		if len(yamlBytes) > 0 {
			fragment, err := unmarshalFragment(yamlBytes)
			if err != nil {
				return err
			}

			doc.Paths, err = mergePaths(doc.Paths, fragment.Paths)
			if err != nil {
				return err
			}

			doc.Tags = mergeTags(doc.Tags, fragment.Tags)
		}

		for name, val := range types {
			schema, deps, err := SchemaFrom(val)
			if err != nil {
				return err
			}
			doc.Components.Schemas[name] = schema

			generated, err := GenerateSchemas(deps)
			if err != nil {
				return err
			}
			for dname, dschema := range generated {
				doc.Components.Schemas[dname] = dschema
			}
		}
	}

	// Todo: Error schema should live in boiler (or delish/respond) since that's
	// where the error response convention is defined. Move when apispec is a module.
	doc.Components.Schemas["Error"] = &Schema{
		Type: "object",
		Properties: Properties{
			{Name: "error", Schema: &Schema{Type: "string", Description: "Error message"}},
		},
	}

	return nil
}

// Marshal renders the Document as JSON bytes.
func (doc Document) Marshal() ([]byte, error) {
	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal api spec")
	}
	return data, nil
}

// MarshalYaml renders the Document as YAML bytes.
func (doc Document) MarshalYaml() ([]byte, error) {
	data, err := yaml.Marshal(doc)
	if err != nil {
		return nil, errors.Wrap(err, "failed to marshal api spec")
	}
	return data, nil
}

type fragment struct {
	Tags  []Tag `yaml:"tags"`
	Paths Paths `yaml:"paths"`
}

func unmarshalFragment(yamlBytes []byte) (fragment, error) {
	var f fragment
	err := yaml.Unmarshal(yamlBytes, &f)
	if err != nil {
		return f, errors.Wrap(err, "failed to unmarshal spec fragment")
	}
	return f, nil
}

func mergePaths(dst Paths, src Paths) (Paths, error) {
	for _, kv := range src {
		if dst.Has(kv.Key) {
			return dst, fmt.Errorf("duplicate path: %s", kv.Key)
		}
		dst = append(dst, kv)
	}
	return dst, nil
}

func mergeTags(dst []Tag, src []Tag) []Tag {
	seen := map[string]bool{}
	for _, t := range dst {
		seen[t.Name] = true
	}
	for _, t := range src {
		if !seen[t.Name] {
			dst = append(dst, t)
			seen[t.Name] = true
		}
	}
	return dst
}
