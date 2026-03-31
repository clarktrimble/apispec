// Package apispec provides a lightweight OpenAPI 3.0 document builder
// with reflection-based schema generation from Go structs.
package apispec

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"
)

// Schema represents an OpenAPI schema object.
type Schema struct {
	Ref                  string     `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Type                 string     `json:"type,omitempty" yaml:"type,omitempty"`
	Format               string     `json:"format,omitempty" yaml:"format,omitempty"`
	Description          string     `json:"description,omitempty" yaml:"description,omitempty"`
	Properties           Properties `json:"properties,omitempty" yaml:"properties,omitempty"`
	Items                *Schema    `json:"items,omitempty" yaml:"items,omitempty"`
	Required             []string   `json:"required,omitempty" yaml:"required,omitempty"`
	Enum                 []any      `json:"enum,omitempty" yaml:"enum,omitempty"`
	Example              any        `json:"example,omitempty" yaml:"example,omitempty"`
	AdditionalProperties *Schema    `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`
}

// Deps collects named struct types discovered during schema generation.
type Deps map[string]reflect.Type

// SchemaFrom generates an OpenAPI Schema from a Go value using json tags.
// Named struct fields become $ref entries; their types are collected in deps
// so the caller can generate component schemas for them.
func SchemaFrom(v any) (*Schema, Deps, error) {
	t := deref(reflect.TypeOf(v))
	deps := Deps{}
	var err error
	schema := schemaFromStruct(t, apiTags, deps, &err)
	return schema, deps, err
}

// GenerateSchemas generates component schemas for all types in deps,
// recursively resolving any new deps discovered along the way.
func GenerateSchemas(deps Deps) (map[string]*Schema, error) {
	schemas := map[string]*Schema{}

	for len(deps) > 0 {
		current := deps
		deps = Deps{}

		var err error
		for name, t := range current {
			if _, done := schemas[name]; done {
				continue
			}
			schemas[name] = schemaFromStruct(t, apiTags, deps, &err)
		}
		if err != nil {
			return nil, err
		}

		for name := range schemas {
			delete(deps, name)
		}
	}

	return schemas, nil
}

// ConfigSchema generates an OpenAPI Schema from a config struct using
// envconfig-style tags: desc, required, default, ignored.
// Config schemas always inline nested structs (no $ref).
func ConfigSchema(v any) *Schema {
	t := deref(reflect.TypeOf(v))
	return schemaFromStruct(t, configTags, nil, nil)
}

// tagReader extracts per-field schema metadata from struct tags.
type tagReader struct {
	skip        func(reflect.StructField) bool
	description func(reflect.StructField) string
	example     func(reflect.StructField) string
	required    func(reflect.StructField, bool) bool
}

var apiTags = tagReader{
	skip: func(f reflect.StructField) bool { return false },
	description: func(f reflect.StructField) string {
		return f.Tag.Get("desc")
	},
	example: func(f reflect.StructField) string {
		return f.Tag.Get("example")
	},
	required: func(f reflect.StructField, omitempty bool) bool {
		return !omitempty && f.Type.Kind() != reflect.Ptr
	},
}

var configTags = tagReader{
	skip: func(f reflect.StructField) bool {
		return f.Tag.Get("ignored") == "true"
	},
	description: func(f reflect.StructField) string {
		return f.Tag.Get("desc")
	},
	example: func(f reflect.StructField) string {
		return f.Tag.Get("default")
	},
	required: func(f reflect.StructField, _ bool) bool {
		return f.Tag.Get("required") == "true"
	},
}

// well-known types that should not be treated as named structs
var wellKnown = map[reflect.Type]bool{
	reflect.TypeOf(time.Time{}):       true,
	reflect.TypeOf(time.Duration(0)):  true,
	reflect.TypeOf(json.RawMessage{}): true,
}

// Describer provides a schema-level description for a type.
type Describer interface {
	Description() string
}

func schemaFromStruct(t reflect.Type, tags tagReader, deps Deps, errp *error) *Schema {
	schema := &Schema{
		Type: "object",
	}

	if v, ok := reflect.New(t).Interface().(Describer); ok {
		schema.Description = v.Description()
	}

	for i := range t.NumField() {
		field := t.Field(i)
		if !field.IsExported() {
			continue
		}

		jsonTag := field.Tag.Get("json")
		if jsonTag == "-" {
			continue
		}
		if tags.skip(field) {
			continue
		}

		name, omitempty := parseJsonTag(jsonTag, field.Name)
		prop := typeSchema(field.Type, tags, deps, errp)

		if desc := tags.description(field); desc != "" {
			prop.Description = desc
		}
		if ex := tags.example(field); ex != "" {
			prop.Example = ex
		}

		schema.Properties = append(schema.Properties, Property{Name: name, Schema: prop})

		if tags.required(field, omitempty) {
			schema.Required = append(schema.Required, name)
		}
	}

	return schema
}

func typeSchema(t reflect.Type, tags tagReader, deps Deps, errp *error) *Schema {
	t = deref(t)

	// well-known types
	switch t {
	case reflect.TypeOf(time.Time{}):
		return &Schema{Type: "string", Format: "date-time"}
	case reflect.TypeOf(time.Duration(0)):
		return &Schema{Type: "string"}
	case reflect.TypeOf(json.RawMessage{}):
		return &Schema{Type: "object", Description: "raw JSON"}
	}

	switch t.Kind() {
	case reflect.String:
		return &Schema{Type: "string"}
	case reflect.Bool:
		return &Schema{Type: "boolean"}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return &Schema{Type: "integer"}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return &Schema{Type: "integer"}
	case reflect.Float32, reflect.Float64:
		return &Schema{Type: "number"}
	case reflect.Slice:
		return &Schema{Type: "array", Items: typeSchema(t.Elem(), tags, deps, errp)}
	case reflect.Map:
		return &Schema{Type: "object", AdditionalProperties: typeSchema(t.Elem(), tags, deps, errp)}
	case reflect.Struct:
		// named struct with deps tracking: emit $ref
		if deps != nil && t.Name() != "" && !wellKnown[t] {
			if existing, ok := deps[t.Name()]; ok && existing != t {
				if errp != nil {
					*errp = fmt.Errorf("schema name collision: %q from %s and %s",
						t.Name(), existing.PkgPath(), t.PkgPath())
				}
				return Ref(t.Name())
			}
			deps[t.Name()] = t
			return Ref(t.Name())
		}
		// inline (config mode or anonymous struct)
		return schemaFromStruct(t, tags, deps, errp)
	case reflect.Interface:
		return &Schema{Type: "object"}
	default:
		return &Schema{Type: "object"}
	}
}

func deref(t reflect.Type) reflect.Type {
	if t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	return t
}

func parseJsonTag(tag, fieldName string) (name string, omitempty bool) {
	name = fieldName
	if tag == "" {
		return
	}

	parts := strings.Split(tag, ",")
	if parts[0] != "" {
		name = parts[0]
	}
	for _, part := range parts[1:] {
		if part == "omitempty" {
			omitempty = true
		}
	}
	return
}
