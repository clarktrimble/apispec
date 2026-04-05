// Package fixture provides test types for schema generation.
package fixture

import "time"

type ServerConfig struct {
	Version string        `json:"version" ignored:"true"`
	Host    string        `json:"host" desc:"hostname or ip to bind"`
	Port    int           `json:"port" desc:"port to listen on" required:"true"`
	Timeout time.Duration `json:"timeout" desc:"request timeout" default:"10s"`
}

// Widget represents a mechanical component in the inventory.
type Widget struct {
	Name      string    `json:"name" desc:"widget name" example:"sprocket"`
	Count     int       `json:"count" desc:"number of widgets"`
	Weight    float64   `json:"weight" desc:"weight in grams"`
	CreatedAt time.Time `json:"created_at" desc:"when the widget was created"`
	// Associated part, if any.
	Part *Part `json:"part,omitempty"`
}

// Part is a sub-component of a widget.
type Part struct {
	ID string `json:"id" example:"p-123"`
	// Human-readable label for the part.
	Label string `json:"label"`
}
