package forgeline

import "testing"

// FuzzKit sits at the matcher's boundary. Its file's name CONTAINS "_test" and
// does not END in "_test.go", so `go test -fuzz` would never run it and nothing
// is reported. A matcher widened from a suffix to a substring would claim it.
func FuzzKit(f *testing.F) {
	f.Fuzz(func(t *testing.T, b []byte) { _ = Parse(b) })
}
