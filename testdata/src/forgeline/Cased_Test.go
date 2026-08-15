package forgeline

import "testing"

// FuzzCased differs from a test file's name only in letter case. The go tool's
// own check is `strings.HasSuffix(name, "_test.go")` and is case-sensitive, so
// this is ordinary compiled source — verified on a case-INSENSITIVE darwin
// filesystem, where `go list` still reports it in GoFiles and never in
// TestGoFiles. `go test -fuzz` will never run it, so nothing is reported.
//
// Case is the third dimension this package's names would otherwise hold
// constant. Folding the name before matching is the ordinary instinct of anyone
// who has been bitten by a Windows or macOS path.
func FuzzCased(f *testing.F) {
	f.Fuzz(func(t *testing.T, b []byte) { _ = Parse(b) })
}
