// Package a is the subject the fuzz fixtures exercise.
package a

// Payload is raw input.
type Payload []byte

// Parse decodes a payload; Render re-encodes it.
func Parse(raw Payload) (string, error) { return string(raw), nil }

// Render inverts Parse.
func Render(decoded string) Payload { return Payload(decoded) }
