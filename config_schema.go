package apispec

import (
	"go/types"
	"reflect"
)

// configSchemaFrom generates an OpenAPI Schema from a config struct type.
// Uses envconfig-style tags (ignored, default, required) and always inlines
// nested structs (no $ref).
func configSchemaFrom(t types.Type) *schema {

	if named, ok := t.(*types.Named); ok {
		if st, ok := named.Underlying().(*types.Struct); ok {
			return configStructSchema(st)
		}
	}
	// fallback: inline type schema with no deps
	return typeToSchema(t, nil, nil)
}

func configStructSchema(st *types.Struct) *schema {

	s := &schema{Type: "object"}

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

		s.Properties = append(s.Properties,
			property{Name: name, Schema: prop})

		if tag.Get("required") == "true" {
			s.Required = append(s.Required, name)
		}
	}

	return s
}

// configTypeSchema is like typeToSchema but always inlines nested structs.
func configTypeSchema(t types.Type) *schema {

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
		return &schema{Type: "array", Items: configTypeSchema(t.Elem())}
	case *types.Map:
		return &schema{Type: "object", AdditionalProperties: configTypeSchema(t.Elem())}
	case *types.Struct:
		return configStructSchema(t)
	case *types.Basic:
		return basicSchema(t)
	case *types.Interface:
		return &schema{Type: "object"}
	default:
		return &schema{Type: "object"}
	}
}
