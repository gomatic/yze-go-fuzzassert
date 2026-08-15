//line nottest.go:1
package forgeline

import "testing"

// FuzzJudged is the direction that disables the rule: a real fuzz target in a
// real test file — go list reports forgeline_test.go in TestGoFiles, and `go
// test -fuzz=FuzzJudged` runs it — whose directive claims a source name.
// Reading the claimed name would take it out of scope, so the assertion-free
// callback below is reported here exactly as it would be without that line.
func FuzzJudged(f *testing.F) {
	f.Fuzz(func(t *testing.T, b []byte) { _ = Parse(b) }) // want "fuzz target invokes without asserting a property"
}
