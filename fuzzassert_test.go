package fuzzassert_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/tools/go/analysis/analysistest"

	fuzzassert "github.com/gomatic/yze-go-fuzzassert"
)

// TestFuzzTargetsMustAssert pins both directions against the fixtures: the
// invoke-only, skip-and-log, and named-empty callbacks are reported; the
// t.Fatalf, testify, helper-delegating, and named-asserting callbacks are not.
func TestFuzzTargetsMustAssert(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), fuzzassert.Analyzer, "a")
}

// TestRegistrationIsWellFormed pins the yze wiring.
func TestRegistrationIsWellFormed(t *testing.T) {
	assert.NoError(t, fuzzassert.Registration.Validate())
	assert.Equal(t, "yze/fuzzassert", fuzzassert.Registration.RuleID())
	assert.Same(t, fuzzassert.Analyzer, fuzzassert.Registration.Analyzer)
}

// TestADirectiveDoesNotRenameAFile pins "a _test.go file" to the name the
// FileSet holds, which the judged file cannot rewrite. A `//line` directive is a
// compiler feature for generated code: it changes what fset.Position reports and
// nothing else — the go tool still compiles the file, go list still names it in
// GoFiles or TestGoFiles, and `go test -fuzz` still runs the target. So a scope
// resolved through Position is decided by the very file being judged.
//
// Both directions are pinned, because this matcher includes rather than exempts.
// Package forgeline's real fuzz target claims a source name and is judged
// anyway; the Fuzz-shaped function in its ordinary source claims a test name and
// is still invisible, because the go tool would never fuzz it.
func TestADirectiveDoesNotRenameAFile(t *testing.T) {
	analysistest.Run(t, analysistest.TestData(), fuzzassert.Analyzer, "forgeline")
}
