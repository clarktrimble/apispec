package apispec

import "strings"

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
	// Todo: OrderedMap[*Schema] with alphabetical sort in Merge, like Paths and Responses.
	Schemas map[string]*Schema `json:"schemas,omitempty" yaml:"schemas,omitempty"`
}

// OpenAPIVersion is the OpenAPI specification version.
const OpenAPIVersion = "3.0.3"

// NewDocument creates a Document with the OpenAPI version set and info populated.
// Version and release follow the build convention: release is a git tag (e.g. "1.2.3")
// or "untagged"; version is a branch.revcount.revhash fallback. Descriptions are
// joined with double newlines.
func NewDocument(version, release, url, title string, descriptions ...string) Document {

	apiVersion := release
	if release == "untagged" {
		apiVersion = "_" + version
	}
	if apiVersion == "" {
		apiVersion = "_unreleased"
	}

	return Document{
		OpenAPI: OpenAPIVersion,
		Info: Info{
			Title:       title,
			Version:     apiVersion,
			Description: strings.Join(descriptions, "\n\n"),
		},
		Servers: []Server{
			{URL: url, Description: "API server"},
		},
	}
}

// Helpers for common patterns.

// Ref creates a schema reference.
func Ref(name string) *Schema {
	return &Schema{Ref: "#/components/schemas/" + name}
}

// JsonContent creates a Content with a single application/json media type.
func JsonContent(schema *Schema) Content {
	return Content{"application/json": &MediaType{Schema: schema}}
}

// StatusResponse creates a simple {"status": "ok"} response.
func StatusResponse(desc string) *Response {
	return &Response{
		Description: desc,
		Content: JsonContent(&Schema{
			Type: "object",
			Properties: Properties{
				{Name: "status", Schema: &Schema{Type: "string", Example: "ok"}},
			},
		}),
	}
}

// ErrorResponse creates a response referencing the Error schema.
func ErrorResponse(desc string) *Response {
	return &Response{
		Description: desc,
		Content:     JsonContent(Ref("Error")),
	}
}

// ObjResponse creates a response with a named object wrapper,
// mirroring respond.WriteObjects(ctx, map[string]any{name: ...}).
func ObjResponse(desc, name string, schema *Schema) *Response {
	return &Response{
		Description: desc,
		Content: JsonContent(&Schema{
			Type: "object",
			Properties: Properties{
				{Name: name, Schema: schema},
			},
		}),
	}
}
