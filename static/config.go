package static

// Config represents an apispec generation configuration file.
type Config struct {
	Title       string   `yaml:"title"`
	Description []string `yaml:"description"`
	Config      *TypeRef `yaml:"config"`
	Specs       []Spec   `yaml:"specs"`
}

// TypeRef identifies a type to generate a schema from.
type TypeRef struct {
	Package string `yaml:"package"`
	Type    string `yaml:"type"`
}

// Spec identifies a package that contributes paths and types.
type Spec struct {
	Package string   `yaml:"package"`
	Types   []string `yaml:"types"`
}
