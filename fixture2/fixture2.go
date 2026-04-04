// Package fixture2 provides types for collision testing.
package fixture2

// Gadget has a Part field that collides with fixture.Part.
type Gadget struct {
	Size string `json:"size"`
	Part Part   `json:"part"`
}

// Part is a different type with the same name as fixture.Part.
type Part struct {
	Serial string `json:"serial"`
}
