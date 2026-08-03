package a

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// FuzzEmpty invokes without asserting: the counter-satisfying shape that
// verifies nothing.
func FuzzEmpty(f *testing.F) {
	f.Add([]byte("seed"))
	f.Fuzz(func(t *testing.T, b []byte) { // want `invokes without asserting`
		_, _ = Parse(b)
	})
}

// FuzzSkipAndLog only skips and logs — neither can fail the test.
func FuzzSkipAndLog(f *testing.F) {
	f.Fuzz(func(t *testing.T, b []byte) { // want `invokes without asserting`
		decoded, err := Parse(b)
		if err != nil {
			t.Skip()
		}
		t.Log(decoded)
	})
}

// FuzzRoundTrip asserts the round-trip property with t.Fatalf.
func FuzzRoundTrip(f *testing.F) {
	f.Fuzz(func(t *testing.T, b []byte) {
		decoded, err := Parse(b)
		if err != nil {
			t.Skip()
		}
		if string(Render(decoded)) != string(b) {
			t.Fatalf("round-trip mismatch for %q", b)
		}
	})
}

// FuzzTestify asserts through testify.
func FuzzTestify(f *testing.F) {
	f.Fuzz(func(t *testing.T, b []byte) {
		decoded, err := Parse(b)
		assert.NoError(t, err)
		assert.Equal(t, string(b), decoded)
	})
}

// FuzzHelper delegates to a helper that receives the T; the analyzer fails
// open across the call boundary.
func FuzzHelper(f *testing.F) {
	f.Fuzz(func(t *testing.T, b []byte) {
		assertRoundTrip(t, b)
	})
}

// assertRoundTrip asserts on behalf of its callers.
func assertRoundTrip(t *testing.T, b []byte) {
	if decoded, err := Parse(b); err == nil && string(Render(decoded)) != string(b) {
		t.Fatalf("round-trip mismatch for %q", b)
	}
}

// FuzzNamedAsserting passes an asserting callback by name.
func FuzzNamedAsserting(f *testing.F) {
	f.Fuzz(assertRoundTripCallback)
}

// assertRoundTripCallback is the named form of the asserting callback.
func assertRoundTripCallback(t *testing.T, b []byte) {
	if decoded, err := Parse(b); err == nil && string(Render(decoded)) != string(b) {
		t.Fatalf("round-trip mismatch for %q", b)
	}
}

// FuzzNamedEmpty passes an assertion-free callback by name; resolution reaches
// its body.
func FuzzNamedEmpty(f *testing.F) {
	f.Fuzz(emptyCallback) // want `invokes without asserting`
}

// emptyCallback invokes without asserting.
func emptyCallback(t *testing.T, b []byte) {
	_, _ = Parse(b)
	_ = t
}

// callbacks holds an out-of-reach callback: an index expression is not a
// resolvable reference, so the analyzer fails open.
var callbacks = []func(*testing.T, []byte){emptyCallback}

// cb is a package-level var of function type — a var, not a declaration, so
// resolution fails open.
var cb = emptyCallback

// FuzzIndexedCallback passes an unresolvable callback; silence beats a false
// positive on a gate.
func FuzzIndexedCallback(f *testing.F) {
	f.Fuzz(callbacks[0])
}

// FuzzVarCallback passes a var-held callback; equally unresolvable.
func FuzzVarCallback(f *testing.F) {
	f.Fuzz(cb)
}

// validator carries a func-typed field, which is neither a testing call nor
// testify.
type validator struct {
	check func([]byte) bool
}

// FuzzFieldCall discriminates through a func field and never fails the test.
func FuzzFieldCall(f *testing.F) {
	v := validator{check: func(b []byte) bool { return len(b) >= 0 }}
	f.Fuzz(func(t *testing.T, b []byte) { // want `invokes without asserting`
		_ = v.check(b)
	})
}

// Fuzz is exactly "Fuzz" — the go rule accepts the empty rest, so this IS a
// target and its assertion-free callback is judged.
func Fuzz(f *testing.F) {
	f.Fuzz(func(t *testing.T, b []byte) { // want `invokes without asserting`
		_, _ = Parse(b)
	})
}

// FuzzyDriver fails the go name rule (lowercase rune after "Fuzz") even in a
// _test.go file: a helper, not a target, so its callback goes unjudged.
func FuzzyDriver(f *testing.F) {
	f.Fuzz(func(t *testing.T, b []byte) {
		_, _ = Parse(b)
	})
}

// FuzzEngineInside is a REAL target: its f.Fuzz callback asserts, and the
// engine.Fuzz call inside is a domain method on a custom type, not the
// fuzzer — it must not be judged as a fuzz entry point.
func FuzzEngineInside(f *testing.F) {
	e := Engine{rounds: 1}
	e.Fuzz(func() {})
	f.Fuzz(func(t *testing.T, b []byte) {
		if decoded, err := Parse(b); err == nil && string(Render(decoded)) != string(b) {
			t.Fatalf("round-trip mismatch for %q", b)
		}
	})
}
