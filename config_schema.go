package apispec

import (
	"go/types"
	"reflect"
)

// configSchemaFrom generates an OpenAPI Schema from a config struct type.
// Uses envconfig-style tags (ignored, default, required) and always inlines
// nested structs (no $ref).
func configSchemaFrom(t types.Type) *Schema {

	if named, ok := t.(*types.Named); ok {
		if st, ok := named.Underlying().(*types.Struct); ok {
			return configStructSchema(st)
		}
	}
	// fallback: inline type schema with no deps
	return typeToSchema(t, nil, nil)
}

func configStructSchema(st *types.Struct) *Schema {

	schema := &Schema{Type: "object"}

	for i := range st.NumFields() {
		field := st.Field(i)
		if !field.Exported() {
			continue
		}

		tag := reflect.StructTag(st.Tag(i))
		jsonTag := tag.Get("json")
		if jsonTag == "-" {
			continue
		}
		if tag.Get("ignored") == "true" {
			continue
		}

		name, _ := parseJSONTag(jsonTag, field.Name())
		prop := configTypeSchema(field.Type())

		if desc := tag.Get("desc"); desc != "" {
			prop.Description = desc
		}
		if def := tag.Get("default"); def != "" {
			prop.Example = def
		}

		schema.Properties = append(schema.Properties,
			Property{Name: name, Schema: prop})

		if tag.Get("required") == "true" {
			schema.Required = append(schema.Required, name)
		}
	}

	return schema
}

// configTypeSchema is like typeToSchema but always inlines nested structs.
func configTypeSchema(t types.Type) *Schema {

	switch t := t.(type) {
	case *types.Named:
		obj := t.Obj()
		pkg := obj.Pkg()
		name := obj.Name()

		if s, ok := wellKnownSchema(pkg, name); ok {
			return s
		}

		if st, ok := t.Underlying().(*types.Struct); ok {
			return configStructSchema(st)
		}
		return configTypeSchema(t.Underlying())
	case *types.Pointer:
		return configTypeSchema(t.Elem())
	case *types.Slice:
		return &Schema{Type: "array", Items: configTypeSchema(t.Elem())}
	case *types.Map:
		return &Schema{Type: "object", AdditionalProperties: configTypeSchema(t.Elem())}
	case *types.Struct:
		return configStructSchema(t)
	case *types.Basic:
		return basicSchema(t)
	case *types.Interface:
		return &Schema{Type: "object"}
	default:
		return &Schema{Type: "object"}
	}
}
