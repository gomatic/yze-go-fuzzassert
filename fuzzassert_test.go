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
