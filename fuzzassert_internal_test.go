package fuzzassert

import (
	"go/ast"
	"go/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.org/x/tools/go/analysis"
)

// TestDeclaredBodyFailsOpenOnAnUnresolvedIdent pins the defensive nil-object
// branch: an identifier the type information does not resolve yields no body,
// so the caller stays silent rather than guessing.
func TestDeclaredBodyFailsOpenOnAnUnresolvedIdent(t *testing.T) {
	t.Parallel()

	pass := &analysis.Pass{TypesInfo: &types.Info{}}

	assert.Nil(t, declaredBody(pass, &ast.Ident{Name: "phantom"}))
}
