package forgeline

import "testing"

// FuzzGolden sits INSIDE the matcher's literal. Its file's name contains
// "_test.go" and does not end in it, so `go list` reports it in GoFiles, `go
// test -fuzz` will never run it, and nothing is reported. A matcher widened
// from a suffix to a substring would claim it.
//
// The fleet holds no file of this shape — the same find that returns 39 files
// for the left edge returns none for this one — and that absence is the reason
// the case is here rather than the reason to skip it: a widening nothing
// exercises is a widening nothing can fail on.
//
// The sibling escape, a package DIRECTORY named "*_test.go", is declined: it
// kills the same one widening and costs a second package in this corpus.
func FuzzGolden(f *testing.F) {
	f.Fuzz(func(t *testing.T, b []byte) { _ = Parse(b) })
}
