package static

import (
	"os"
	"path/filepath"

	"github.com/clarktrimble/apispec"
	"github.com/pkg/errors"
	"go.yaml.in/yaml/v3"
	"golang.org/x/tools/go/packages"
)

type fragment struct {
	Tags  []apispec.Tag  `yaml:"tags"`
	Paths apispec.Paths  `yaml:"paths"`
}

// loadFragment finds and parses paths.yaml in a package's directory.
func loadFragment(pkg *packages.Package) (apispec.Paths, []apispec.Tag, error) {

	if len(pkg.GoFiles) == 0 {
		return nil, nil, errors.Errorf("package %s has no Go files", pkg.PkgPath)
	}

	dir := filepath.Dir(pkg.GoFiles[0])
	path := filepath.Join(dir, "paths.yaml")

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, errors.Wrapf(err, "reading %s", path)
	}

	var f fragment
	if err := yaml.Unmarshal(data, &f); err != nil {
		return nil, nil, errors.Wrapf(err, "parsing %s", path)
	}

	return f.Paths, f.Tags, nil
}
