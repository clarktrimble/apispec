package apispec

import (
	"go/types"
	"reflect"
	"strings"

	"github.com/pkg/errors"
)

// deps tracks named struct types discovered during schema generation.
type deps map[string]*types.Named

// schemaEntry pairs a generated schema with the type that produced it,
// so name collisions between different types can be detected.
type schemaEntry struct {
	schema *schema
	source *types.Named
}

// resolveAll generates schemas for all deps, chasing transitive deps
// until no new ones appear. Returns an error if two different types
// from different packages produce the same schema name.
func resolveAll(schemas map[string]schemaEntry, pending deps, df *docFinder) error {
	for len(pending) > 0 {
		next := deps{}
		for name, t := range pending {
			if existing, done := schemas[name]; done {
				if existing.source != t {
					return errors.Errorf("schema name collision: %q from %s and %s",
						name, existing.source.Obj().Pkg().Path(), t.Obj().Pkg().Path())
				}
				continue
			}
			s, discovered := schemaFrom(t, df)
			schemas[name] = schemaEntry{schema: s, source: t}
			for dname, dt := range discovered {
				if _, done := schemas[dname]; !done {
					next[dname] = dt
				}
			}
		}
		pending = next
	}
	return nil
}

func schemaFrom(t types.Type, df *docFinder) (*schema, deps) {
	d := deps{}

	// For named struct types, generate the full schema directly
	// rather than emitting a $ref (which is for nested fields).
	if named, ok := t.(*types.Named); ok {
		if st, ok := named.Underlying().(*types.Struct); ok {
			schema := structSchema(st, named.Obj(), d, df)
			// type-level doc comment as schema description
			if df != nil && schema.Description == "" {
				schema.Description = df.typeDoc(named.Obj())
			}
			return schema, d
		}
	}

	return typeToSchema(t, d, df), d
}

func typeToSchema(t types.Type, d deps, df *docFinder) *schema {

	switch t := t.(type) {
	case *types.Named:
		return namedSchema(t, d, df)
	case *types.Pointer:
		return typeToSchema(t.Elem(), d, df)
	case *types.Slice:
		return &schema{
			Type:  "array",
			Items: typeToSchema(t.Elem(), d, df),
		}
	case *types.Map:
		return &schema{
			Type:                 "object",
			AdditionalProperties: typeToSchema(t.Elem(), d, df),
		}
	case *types.Struct:
		return structSchema(t, nil, d, df)
	case *types.Basic:
		return basicSchema(t)
	case *types.Interface:
		return &schema{Type: "object"}
	default:
		return &schema{Type: "object"}
	}
}

func namedSchema(t *types.Named, d deps, df *docFinder) *schema {

	obj := t.Obj()
	pkg := obj.Pkg()
	name := obj.Name()

	// well-known types: inline rather than $ref
	if s, ok := wellKnownSchema(pkg, name); ok {
		return s
	}

	// non-struct named types: use underlying
	st, ok := t.Underlying().(*types.Struct)
	if !ok {
		return typeToSchema(t.Underlying(), d, df)
	}

	// named struct: register as dep and emit $ref
	if pkg != nil && name != "" {
		d[name] = t
		return ref(name)
	}

	return structSchema(st, nil, d, df)
}

// structSchema generates a schema from a struct type.
// obj is the named type's object (for field doc lookup), nil for anonymous structs.
func structSchema(st *types.Struct, obj types.Object, d deps, df *docFinder) *schema {

	s := &schema{Type: "object"}

	// resolve package path for field doc lookup
	var pkgPath string
	if obj != nil && obj.Pkg() != nil {
		pkgPath = obj.Pkg().Path()
	}

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

		name, omitempty := parseJSONTag(jsonTag, field.Name())
		prop := typeToSchema(field.Type(), d, df)

		// desc tag wins, then doc comment as fallback
		if desc := tag.Get("desc"); desc != "" {
			prop.Description = desc
		} else if df != nil && pkgPath != "" {
			if doc := df.fieldDoc(pkgPath, field.Pos()); doc != "" {
				prop.Description = doc
			}
		}

		if ex := tag.Get("example"); ex != "" {
			prop.Example = ex
		}

		s.Properties = append(s.Properties,
			property{Name: name, Schema: prop})

		if !omitempty && !isPointer(field.Type()) {
			s.Required = append(s.Required, name)
		}
	}

	return s
}

func basicSchema(t *types.Basic) *schema {

	switch info := t.Info(); {
	case info&types.IsString != 0:
		return &schema{Type: "string"}
	case info&types.IsBoolean != 0:
		return &schema{Type: "boolean"}
	case info&types.IsInteger != 0:
		return &schema{Type: "integer"}
	case info&types.IsFloat != 0:
		return &schema{Type: "number"}
	default:
		return &schema{Type: "object"}
	}
}

func isPointer(t types.Type) bool {
	_, ok := t.(*types.Pointer)
	return ok
}

func wellKnownSchema(pkg *types.Package, name string) (*schema, bool) {
	if pkg == nil {
		return nil, false
	}
	switch path := pkg.Path(); {
	case path == "time" && name == "Time":
		return &schema{Type: "string", Format: "date-time"}, true
	case path == "time" && name == "Duration":
		return &schema{Type: "string"}, true
	case path == "encoding/json" && name == "RawMessage":
		return &schema{Type: "object", Description: "raw JSON"}, true
	}
	return nil, false
}

func parseJSONTag(tag, fieldName string) (name string, omitempty bool) {
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
