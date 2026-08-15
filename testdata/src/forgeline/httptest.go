package forgeline

import "testing"

// FuzzHelper sits at the matcher's LEFT edge — the underscore that separates a
// base name from the "_test.go" suffix. Its file's name ends in "test.go" and
// not in "_test.go", so `go test -fuzz` would never run it and nothing is
// reported.
//
// This edge is not hypothetical and not latent: `find ~/src/github.com -name
// '*test.go' -not -name '*_test.go'` returns 39 files, among them
// net/http/httptest/httptest.go and gomatic/go-wofl/internal/pgtest/pgtest.go.
// A matcher that dropped the underscore would report a fuzz-shaped function in
// production source as an unasserted fuzz target — a false positive on code the
// go tool will never fuzz, and a false positive is answered with a baseline.
func FuzzHelper(f *testing.F) {
	f.Fuzz(func(t *testing.T, b []byte) { _ = Parse(b) })
}
