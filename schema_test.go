package apispec_test

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/clarktrimble/apispec"
)

// API types — only json tags matter

type Event struct {
	Id      string          `json:"id,omitempty"`
	Type    string          `json:"event_type"`
	Webhook string          `json:"webhook_name,omitempty"`
	Remote  string          `json:"remote,omitempty"`
	Payload json.RawMessage `json:"payload"`
}

// Nested types for $ref testing

type Network struct {
	Name string `json:"name"`
}

type Emitter struct {
	Id       string  `json:"transmitter_id"`
	Protocol string  `json:"protocol"`
	Vendor   string  `json:"vendor"`
	Network  Network `json:"network"`
}

type Area struct {
	SiteId         string `json:"site_id"`
	ConcentratorId string `json:"concentrator_id"`
	MapId          string `json:"map_id"`
}

type DeviceInfo struct {
	Manufacturer string `json:"manufacturer"`
	User         string `json:"user"`
	Model        string `json:"model"`
	Name         string `json:"name"`
}

type Observation struct {
	Timestamp  int64      `json:"time_s"`
	FirstSeen  int64      `json:"first_seen_lifetime"`
	Tags       []string   `json:"tags"`
	Position   []float64  `json:"position"`
	Area       Area       `json:"area"`
	AreaId     string     `json:"area_id"`
	Emitter    Emitter    `json:"emitter"`
	DeviceInfo DeviceInfo `json:"device_info"`
}

// Config types — desc, required, default, ignored tags matter

type TopConfig struct {
	Version string         `json:"version" ignored:"true"`
	Release string         `json:"release" ignored:"true"`
	Logger  *LoggerConfig  `json:"logger"`
	S3      *S3Config      `json:"s3"`
	Sender  *SenderConfig  `json:"event_sender"`
	Svc     *SvcConfig     `json:"svc"`
	Webhook *WebhookConfig `json:"webhook"`
	Server  *ServerConfig  `json:"http_server"`
}

type LoggerConfig struct {
	MaxLen      int  `json:"max_len" default:"999" desc:"maximum length that will be logged for any field"`
	EnableDebug bool `json:"enable_debug" default:"false" desc:"log debug messages"`
	EnableTrace bool `json:"enable_trace" default:"false" desc:"log trace messages"`
}

type S3Config struct {
	Region    string `json:"region" desc:"provider region" required:"true"`
	Scheme    string `json:"scheme" desc:"http or https" default:"https"`
	Host      string `json:"host" desc:"endpoint hostname" required:"true"`
	Bucket    string `json:"bucket" desc:"bucket name" required:"true"`
	AccessKey string `json:"access_key" desc:"credential identifier" required:"true"`
	SecretKey string `json:"secret_key" desc:"credential secret" required:"true"`
}

type SenderConfig struct {
	ObjectName string `json:"object_name" desc:"object naming template" default:"events_{ts}_{rand}.ndjson"`
}

type SvcConfig struct {
	Webhook  bool   `json:"include_webhook" desc:"include webhook name" default:"false"`
	Remote   bool   `json:"include_remote" desc:"include event sender ip" default:"false"`
	Raw      bool   `json:"include_raw" desc:"include original event" default:"false"`
	DryRun   bool   `json:"dry_run" desc:"skip sending downstream" default:"false"`
	Customer string `json:"customer" desc:"customer-specific formatting" default:""`
}

type WebhookConfig struct {
	Interval    time.Duration `json:"interval" desc:"period to process buffered events" default:"60s"`
	BufferSize  int           `json:"buffer_size" desc:"size of event buffer" default:"999"`
	BufferTypes []string      `json:"buffer_types" desc:"event types to buffer" default:"observations"`
}

type ServerConfig struct {
	Host    string        `json:"host" desc:"hostname or ip for which to bind"`
	Port    int           `json:"port" desc:"port on which to listen" required:"true"`
	Timeout time.Duration `json:"timeout" desc:"characteristic timeout" default:"10s"`
}

func TestSchemaFrom(t *testing.T) {

	schema, deps, err := apispec.SchemaFrom(Event{})
	if err != nil {
		t.Fatalf("SchemaFrom: %v", err)
	}
	dump(t, "Event", schema)

	if schema.Type != "object" {
		t.Errorf("expected object, got %s", schema.Type)
	}
	if len(schema.Properties) != 5 {
		t.Errorf("expected 5 properties, got %d", len(schema.Properties))
	}
	if len(schema.Required) != 2 {
		t.Errorf("expected 2 required, got %d: %v", len(schema.Required), schema.Required)
	}
	// Event has no named struct fields, so no deps
	if len(deps) != 0 {
		t.Errorf("expected 0 deps, got %d", len(deps))
	}
}

func TestSchemaFromNested(t *testing.T) {

	schema, deps, err := apispec.SchemaFrom(Observation{})
	if err != nil {
		t.Fatalf("SchemaFrom: %v", err)
	}
	dump(t, "Observation", schema)

	// nested named structs should be $ref, not inlined
	area := schema.Properties.Get("area")
	if area == nil {
		t.Fatal("missing area property")
	}
	if area.Ref != "#/components/schemas/Area" {
		t.Errorf("expected $ref for area, got: %+v", area)
	}

	emitter := schema.Properties.Get("emitter")
	if emitter == nil {
		t.Fatal("missing emitter property")
	}
	if emitter.Ref != "#/components/schemas/Emitter" {
		t.Errorf("expected $ref for emitter, got: %+v", emitter)
	}

	// deps should contain the nested types
	if len(deps) != 3 {
		t.Errorf("expected 3 deps (Area, Emitter, DeviceInfo), got %d: %v", len(deps), depsNames(deps))
	}

	// generate component schemas from deps
	schemas, err := apispec.GenerateSchemas(deps)
	if err != nil {
		t.Fatalf("GenerateSchemas: %v", err)
	}
	fmt.Println("--- Component Schemas ---")
	for name, s := range schemas {
		dump(t, name, s)
	}

	// Emitter has Network, so it should be in generated schemas
	if schemas["Network"] == nil {
		t.Error("expected Network in generated schemas (transitive dep of Emitter)")
	}
	if schemas["Area"] == nil {
		t.Error("expected Area in generated schemas")
	}
}

func TestConfigSchema(t *testing.T) {

	schema := apispec.ConfigSchema(TopConfig{})
	dump(t, "TopConfig", schema)

	// ignored fields should be absent
	if schema.Properties.Get("version") != nil {
		t.Error("version should be ignored")
	}
	if schema.Properties.Get("release") != nil {
		t.Error("release should be ignored")
	}

	// nested config should have descriptions and be inlined
	logger := schema.Properties.Get("logger")
	if logger == nil {
		t.Fatal("missing logger property")
	}
	if logger.Ref != "" {
		t.Error("config nested structs should be inlined, not $ref")
	}
	if logger.Properties.Get("max_len").Description != "maximum length that will be logged for any field" {
		t.Error("expected desc tag on max_len")
	}
	if logger.Properties.Get("max_len").Example != "999" {
		t.Error("expected default tag as example on max_len")
	}

	// required from tag
	s3 := schema.Properties.Get("s3")
	if s3 == nil {
		t.Fatal("missing s3 property")
	}
	if len(s3.Required) != 5 {
		t.Errorf("expected 5 required s3 fields, got %d: %v", len(s3.Required), s3.Required)
	}

	// no required on top-level (all pointer fields, no required tags)
	if len(schema.Required) != 0 {
		t.Errorf("expected 0 required on top-level, got %d: %v", len(schema.Required), schema.Required)
	}
}

func dump(t *testing.T, name string, schema *apispec.Schema) {
	t.Helper()
	data, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	fmt.Printf("--- %s ---\n%s\n\n", name, data)
}

func depsNames(deps apispec.Deps) []string {
	names := make([]string, 0, len(deps))
	for name := range deps {
		names = append(names, name)
	}
	return names
}
