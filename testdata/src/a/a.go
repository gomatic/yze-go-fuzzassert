// Package a is the subject the fuzz fixtures exercise.
package a

import "testing"

// Payload is raw input.
type Payload []byte

// Parse decodes a payload; Render re-encodes it.
func Parse(raw Payload) (string, error) { return string(raw), nil }

// Render inverts Parse.
func Render(decoded string) Payload { return Payload(decoded) }

// Mutation is one perturbation an engine applies.
type Mutation func()

// Engine drives randomized mutations in PRODUCTION code; Fuzz is its domain
// verb, not testing's.
type Engine struct {
	rounds int
}

// Fuzz runs the mutation once per configured round.
func (e Engine) Fuzz(mutate Mutation) {
	for round := 0; round < e.rounds; round++ {
		mutate()
	}
}

// FuzzDrive is production code: Fuzz-prefixed, calling a method named Fuzz
// with a closure — the exact shape a naive prefix match mistakes for an
// assertion-free fuzz target. Not in a _test.go file, so never a target.
func FuzzDrive(engine Engine) {
	engine.Fuzz(func() {})
}

// FuzzyMatcher shares only the prefix: the go rule's boundary (non-lowercase
// rune after "Fuzz") excludes it, and it is production code besides.
func FuzzyMatcher(engine Engine) {
	engine.Fuzz(func() {})
}

// FuzzNoParams is Fuzz-named with no parameters: not a target signature. (In
// a _test.go file the go loader would reject this shape outright, which is
// why the wrong-signature fixtures live in production code.)
func FuzzNoParams() {}

// FuzzPair declares two parameters in one field; still not one *testing.F.
func FuzzPair(a, b Engine) {
	a.Fuzz(func() {})
	b.Fuzz(func() {})
}

// FuzzOtherPointer takes a pointer, but not to testing.F.
func FuzzOtherPointer(engine *Engine) {
	engine.Fuzz(func() {})
}

// FuzzErrPointer takes a pointer to a universe type, whose named type has no
// package — the nil-package guard must hold, not panic.
func FuzzErrPointer(problem *error) {
	_ = problem
}

// FuzzSeed has the EXACT target signature and an assertion-free callback, but
// lives in production code — the go tool runs targets only from _test.go
// files, so it must stay unjudged.
func FuzzSeed(f *testing.F) {
	f.Fuzz(func(t *testing.T, b []byte) {
		_, _ = Parse(b)
	})
}
