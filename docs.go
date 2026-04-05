package apispec

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/packages"
)

// docFinder looks up doc comments from the AST for types and struct fields.
type docFinder struct {
	pkgs map[string]*packages.Package
}

func newDocFinder(pkgs map[string]*packages.Package) *docFinder {
	return &docFinder{pkgs: pkgs}
}

// typeDoc returns the doc comment for a named type.
func (df *docFinder) typeDoc(obj types.Object) string {

	pkg := df.pkgs[obj.Pkg().Path()]
	if pkg == nil {
		return ""
	}

	pos := obj.Pos()
	for _, file := range pkg.Syntax {
		for _, decl := range file.Decls {
			gd, ok := decl.(*ast.GenDecl)
			if !ok || gd.Tok != token.TYPE {
				continue
			}
			for _, spec := range gd.Specs {
				ts, ok := spec.(*ast.TypeSpec)
				if !ok {
					continue
				}
				if ts.Name.Pos() == pos {
					if ts.Doc != nil {
						return clean(ts.Doc.Text())
					}
					// single-spec GenDecl: doc lives on the GenDecl
					if len(gd.Specs) == 1 && gd.Doc != nil {
						return clean(gd.Doc.Text())
					}
				}
			}
		}
	}
	return ""
}

// fieldDoc returns the doc comment for a struct field at the given position.
func (df *docFinder) fieldDoc(pkgPath string, pos token.Pos) string {

	pkg := df.pkgs[pkgPath]
	if pkg == nil {
		return ""
	}

	for _, file := range pkg.Syntax {
		field := findField(file, pos)
		if field == nil {
			continue
		}
		if field.Doc != nil {
			return clean(field.Doc.Text())
		}
		if field.Comment != nil {
			return clean(field.Comment.Text())
		}
	}
	return ""
}

func findField(file *ast.File, pos token.Pos) *ast.Field {
	var found *ast.Field
	ast.Inspect(file, func(n ast.Node) bool {
		if found != nil {
			return false
		}
		f, ok := n.(*ast.Field)
		if !ok {
			return true
		}
		for _, name := range f.Names {
			if name.Pos() == pos {
				found = f
				return false
			}
		}
		return true
	})
	return found
}

func clean(s string) string {
	return strings.TrimSpace(s)
}
