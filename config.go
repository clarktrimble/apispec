package apispec

// config represents an apispec generation configuration file.
type config struct {
	Title       string   `yaml:"title"`
	Description []string `yaml:"description"`
	Config      *typeRef `yaml:"config"`
	Specs       []spec   `yaml:"specs"`
}

// typeRef identifies a type to generate a schema from.
type typeRef struct {
	Name    string `yaml:"name"`
	Package string `yaml:"package"`
	Type    string `yaml:"type"`
}

// spec identifies a package that contributes paths and types.
type spec struct {
	Package string   `yaml:"package"`
	Types   []string `yaml:"types"`
}
