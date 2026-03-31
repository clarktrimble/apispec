package apispec_test

import (
	"encoding/json"
	"fmt"
	"testing"

	"github.com/clarktrimble/apispec"
)

func TestBuildDocument(t *testing.T) {

	// what fwd's spec might look like, assembled from contributors

	doc := apispec.Document{
		OpenAPI: "3.0.3",
		Info: apispec.Info{
			Title:       "Fwd Event Forwarding API",
			Version:     "0.1.0",
			Description: "Forwards BFC webhook events to object storage.",
			Contact:     &apispec.Contact{Name: "Bastille Integration Team"},
			License:     &apispec.License{Name: "Proprietary"},
		},
		Servers: []apispec.Server{
			{URL: "http://localhost:3031", Description: "API server"},
		},
		Tags: []apispec.Tag{
			{Name: "webhooks", Description: "Fusion Center webhook event processing"},
			{Name: "operations", Description: "Service health and operational controls"},
		},
		Paths: apispec.Paths{
			// webhook routes
			{Key: "/fc-events", Val: &apispec.PathItem{
				Post: &apispec.Operation{
					Summary:     "Receive webhook events",
					OperationID: "receiveWebhookEvents",
					Tags:        []string{"webhooks"},
					RequestBody: &apispec.RequestBody{
						Required: true,
						Content: apispec.JsonContent(&apispec.Schema{
							Type:  "array",
							Items: apispec.Ref("Event"),
						}),
					},
					Responses: apispec.Responses{
						{Key: "200", Val: apispec.StatusResponse("Events received and buffered")},
						{Key: "400", Val: apispec.ErrorResponse("Invalid event format")},
						{Key: "500", Val: apispec.ErrorResponse("Internal server error")},
					},
				},
			}},
			{Key: "/fc-events/flush", Val: &apispec.PathItem{
				Post: &apispec.Operation{
					Summary:     "Flush event buffer",
					OperationID: "flushWebhookEvents",
					Tags:        []string{"webhooks"},
					Responses: apispec.Responses{
						{Key: "200", Val: apispec.StatusResponse("Events processed")},
					},
				},
			}},
			{Key: "/fc-events/dump", Val: &apispec.PathItem{
				Get: &apispec.Operation{
					Summary:     "Dump event buffer",
					OperationID: "dumpWebhookEvents",
					Tags:        []string{"webhooks"},
					Responses: apispec.Responses{
						{Key: "200", Val: &apispec.Response{
							Description: "Buffered events returned",
							Content: apispec.JsonContent(&apispec.Schema{
								Type:  "array",
								Items: apispec.Ref("Event"),
							}),
						}},
					},
				},
			}},
			{Key: "/fc-events/stats", Val: &apispec.PathItem{
				Get: &apispec.Operation{
					Summary:     "Get webhook event stats",
					OperationID: "getWebhookStats",
					Tags:        []string{"webhooks"},
					Responses: apispec.Responses{
						{Key: "200", Val: &apispec.Response{
							Description: "Event stats retrieved",
							Content: func() apispec.Content {
								s, _, _ := apispec.SchemaFrom(eventCounts{})
								return apispec.JsonContent(s)
							}(),
						}},
					},
				},
			}},
			// boiler routes
			{Key: "/config", Val: &apispec.PathItem{
				Get: &apispec.Operation{
					Summary:     "Get service configuration",
					OperationID: "getConfig",
					Tags:        []string{"operations"},
					Responses: apispec.Responses{
						{Key: "200", Val: &apispec.Response{
							Description: "Configuration retrieved",
							Content:     apispec.JsonContent(apispec.ConfigSchema(TopConfig{})),
						}},
					},
				},
			}},
			{Key: "/monitor", Val: &apispec.PathItem{
				Get: &apispec.Operation{
					Summary:     "Health check",
					OperationID: "healthCheck",
					Tags:        []string{"operations"},
					Responses: apispec.Responses{
						{Key: "200", Val: apispec.StatusResponse("Service is healthy")},
					},
				},
			}},
			// stats route
			{Key: "/stats", Val: &apispec.PathItem{
				Get: &apispec.Operation{
					Summary:     "Get service stats",
					OperationID: "getStats",
					Tags:        []string{"operations"},
					Responses: apispec.Responses{
						{Key: "200", Val: &apispec.Response{
							Description: "Stats retrieved",
							Content: apispec.JsonContent(&apispec.Schema{
								Type: "object",
							}),
						}},
					},
				},
			}},
		},
		Components: &apispec.Components{
			Schemas: map[string]*apispec.Schema{
				"Event": func() *apispec.Schema { s, _, _ := apispec.SchemaFrom(Event{}); return s }(),
				"Error": {
					Type: "object",
					Properties: apispec.Properties{
						{Name: "error", Schema: &apispec.Schema{Type: "string", Description: "Error message"}},
					},
				},
			},
		},
	}

	data, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	fmt.Println(string(data))
}

// stand-in for webhook eventCounts
type eventCounts struct {
	Buffered int `json:"buffered"`
	Rejected int `json:"rejected"`
	Ignored  int `json:"ignored"`
}
