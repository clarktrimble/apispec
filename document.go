package apispec

// Document represents an OpenAPI 3.0.3 document.
type Document struct {
	OpenAPI    string      `json:"openapi" yaml:"openapi"`
	Info       Info        `json:"info" yaml:"info"`
	Servers    []Server    `json:"servers,omitempty" yaml:"servers,omitempty"`
	Tags       []Tag       `json:"tags,omitempty" yaml:"tags,omitempty"`
	Paths      Paths       `json:"paths" yaml:"paths"`
	Components *Components `json:"components,omitempty" yaml:"components,omitempty"`
}

// Info provides metadata about the API.
type Info struct {
	Title       string   `json:"title" yaml:"title"`
	Version     string   `json:"version" yaml:"version"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Contact     *Contact `json:"contact,omitempty" yaml:"contact,omitempty"`
	License     *License `json:"license,omitempty" yaml:"license,omitempty"`
}

// Contact information for the API.
type Contact struct {
	Name  string `json:"name,omitempty" yaml:"name,omitempty"`
	Email string `json:"email,omitempty" yaml:"email,omitempty"`
}

// License information for the API.
type License struct {
	Name string `json:"name" yaml:"name"`
	URL  string `json:"url,omitempty" yaml:"url,omitempty"`
}

// Server represents an API server.
type Server struct {
	URL         string `json:"url" yaml:"url"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// Tag adds metadata to a group of operations.
type Tag struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// Paths is an ordered collection of URL patterns to their operations.
type Paths = OrderedMap[*PathItem]

// PathItem describes operations on a single path.
type PathItem struct {
	Get    *Operation `json:"get,omitempty" yaml:"get,omitempty"`
	Post   *Operation `json:"post,omitempty" yaml:"post,omitempty"`
	Put    *Operation `json:"put,omitempty" yaml:"put,omitempty"`
	Delete *Operation `json:"delete,omitempty" yaml:"delete,omitempty"`
	Patch  *Operation `json:"patch,omitempty" yaml:"patch,omitempty"`
}

// Operation describes a single API operation on a path.
type Operation struct {
	Summary     string       `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string       `json:"description,omitempty" yaml:"description,omitempty"`
	OperationID string       `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Tags        []string     `json:"tags,omitempty" yaml:"tags,omitempty"`
	Parameters  []Parameter  `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody *RequestBody `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Responses   Responses    `json:"responses" yaml:"responses"`
}

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

// Parameter describes a single operation parameter.
type Parameter struct {
	Name        string  `json:"name" yaml:"name"`
	In          string  `json:"in" yaml:"in"`
	Required    bool    `json:"required,omitempty" yaml:"required,omitempty"`
	Description string  `json:"description,omitempty" yaml:"description,omitempty"`
	Schema      *Schema `json:"schema,omitempty" yaml:"schema,omitempty"`
	Example     any     `json:"example,omitempty" yaml:"example,omitempty"`
}

// RequestBody describes a request body.
type RequestBody struct {
	Required bool    `json:"required,omitempty" yaml:"required,omitempty"`
	Content  Content `json:"content" yaml:"content"`
}

// Content maps media types to their schema.
type Content map[string]*MediaType

// MediaType describes a media type with a schema.
type MediaType struct {
	Schema *Schema `json:"schema" yaml:"schema"`
}

// Responses is an ordered collection of HTTP status codes to their response definition.
type Responses = OrderedMap[*Response]

// Response describes a single response from an API operation.
type Response struct {
	Description string  `json:"description" yaml:"description"`
	Content     Content `json:"content,omitempty" yaml:"content,omitempty"`
}

// Components holds reusable schema definitions.
type Components struct {
	Schemas map[string]*Schema `json:"schemas,omitempty" yaml:"schemas,omitempty"`
}

// OpenAPIVersion is the OpenAPI specification version.
const OpenAPIVersion = "3.0.3"

// Ref creates a schema reference.
func Ref(name string) *Schema {
	return &Schema{Ref: "#/components/schemas/" + name}
}
