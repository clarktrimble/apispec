package apispec

// document represents an OpenAPI 3.0.3 document.
type document struct {
	OpenAPI    string      `json:"openapi" yaml:"openapi"`
	Info       info        `json:"info" yaml:"info"`
	Servers    []server    `json:"servers,omitempty" yaml:"servers,omitempty"`
	Tags       []tag       `json:"tags,omitempty" yaml:"tags,omitempty"`
	Paths      paths       `json:"paths" yaml:"paths"`
	Components *components `json:"components,omitempty" yaml:"components,omitempty"`
}

// info provides metadata about the API.
type info struct {
	Title       string   `json:"title" yaml:"title"`
	Version     string   `json:"version" yaml:"version"`
	Description string   `json:"description,omitempty" yaml:"description,omitempty"`
	Contact     *contact `json:"contact,omitempty" yaml:"contact,omitempty"`
	License     *license `json:"license,omitempty" yaml:"license,omitempty"`
}

// contact information for the API.
type contact struct {
	Name  string `json:"name,omitempty" yaml:"name,omitempty"`
	Email string `json:"email,omitempty" yaml:"email,omitempty"`
}

// license information for the API.
type license struct {
	Name string `json:"name" yaml:"name"`
	URL  string `json:"url,omitempty" yaml:"url,omitempty"`
}

// server represents an API server.
type server struct {
	URL         string `json:"url" yaml:"url"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// tag adds metadata to a group of operations.
type tag struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

// paths is an ordered collection of URL patterns to their operations.
type paths = orderedMap[*pathItem]

// pathItem describes operations on a single path.
type pathItem struct {
	Get    *operation `json:"get,omitempty" yaml:"get,omitempty"`
	Post   *operation `json:"post,omitempty" yaml:"post,omitempty"`
	Put    *operation `json:"put,omitempty" yaml:"put,omitempty"`
	Delete *operation `json:"delete,omitempty" yaml:"delete,omitempty"`
	Patch  *operation `json:"patch,omitempty" yaml:"patch,omitempty"`
}

// operation describes a single API operation on a path.
type operation struct {
	Summary     string       `json:"summary,omitempty" yaml:"summary,omitempty"`
	Description string       `json:"description,omitempty" yaml:"description,omitempty"`
	OperationID string       `json:"operationId,omitempty" yaml:"operationId,omitempty"`
	Tags        []string     `json:"tags,omitempty" yaml:"tags,omitempty"`
	Parameters  []parameter  `json:"parameters,omitempty" yaml:"parameters,omitempty"`
	RequestBody *requestBody `json:"requestBody,omitempty" yaml:"requestBody,omitempty"`
	Responses   responses    `json:"responses" yaml:"responses"`
}

// schema represents an OpenAPI schema object.
type schema struct {
	Ref                  string     `json:"$ref,omitempty" yaml:"$ref,omitempty"`
	Type                 string     `json:"type,omitempty" yaml:"type,omitempty"`
	Format               string     `json:"format,omitempty" yaml:"format,omitempty"`
	Description          string     `json:"description,omitempty" yaml:"description,omitempty"`
	Properties           properties `json:"properties,omitempty" yaml:"properties,omitempty"`
	Items                *schema    `json:"items,omitempty" yaml:"items,omitempty"`
	Required             []string   `json:"required,omitempty" yaml:"required,omitempty"`
	Enum                 []any      `json:"enum,omitempty" yaml:"enum,omitempty"`
	Example              any        `json:"example,omitempty" yaml:"example,omitempty"`
	AdditionalProperties *schema    `json:"additionalProperties,omitempty" yaml:"additionalProperties,omitempty"`
}

// parameter describes a single operation parameter.
type parameter struct {
	Name        string  `json:"name" yaml:"name"`
	In          string  `json:"in" yaml:"in"`
	Required    bool    `json:"required,omitempty" yaml:"required,omitempty"`
	Description string  `json:"description,omitempty" yaml:"description,omitempty"`
	Schema      *schema `json:"schema,omitempty" yaml:"schema,omitempty"`
	Example     any     `json:"example,omitempty" yaml:"example,omitempty"`
}

// requestBody describes a request body.
type requestBody struct {
	Required bool    `json:"required,omitempty" yaml:"required,omitempty"`
	Content  content `json:"content" yaml:"content"`
}

// content maps media types to their schema.
type content map[string]*mediaType

// mediaType describes a media type with a schema.
type mediaType struct {
	Schema *schema `json:"schema" yaml:"schema"`
}

// responses is an ordered collection of HTTP status codes to their response definition.
type responses = orderedMap[*response]

// response describes a single response from an API operation.
type response struct {
	Description string  `json:"description" yaml:"description"`
	Content     content `json:"content,omitempty" yaml:"content,omitempty"`
}

// components holds reusable schema definitions.
type components struct {
	Schemas map[string]*schema `json:"schemas,omitempty" yaml:"schemas,omitempty"`
}

// openAPIVersion is the OpenAPI specification version.
const openAPIVersion = "3.0.3"

// ref creates a schema reference.
func ref(name string) *schema {
	return &schema{Ref: "#/components/schemas/" + name}
}
