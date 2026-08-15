//line zz_test.go:1
package forgeline

import "testing"

// Parse is the code a fuzz target would drive.
func Parse(b []byte) int { return len(b) }

// FuzzSource is ordinary compiled source — go list reports forgeline.go in
// GoFiles, and `go test -fuzz` will never run this — and the directive above is
// the only thing claiming a test name for it. A target is what the GO TOOL
// would run, so nothing is reported here.
func FuzzSource(f *testing.F) {
	f.Fuzz(func(t *testing.T, b []byte) { _ = Parse(b) })
}
