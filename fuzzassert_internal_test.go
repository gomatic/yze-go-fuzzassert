package fuzzassert

import (
	"go/ast"
	"go/token"
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

// testingFType fabricates the *testing.F type without loading the real
// package: isTestingF discriminates on package path and type name alone.
func testingFType() types.Type {
	pkg := types.NewPackage("testing", "testing")
	name := types.NewTypeName(token.NoPos, pkg, "F", nil)
	return types.NewPointer(types.NewNamed(name, types.NewStruct(nil, nil), nil))
}

// TestIsFuzzNameFollowsTheGoBoundaryRule pins the go tool's identifier rule:
// "Fuzz" followed by a non-lowercase rune, or exactly "Fuzz" — never a bare
// prefix match.
func TestIsFuzzNameFollowsTheGoBoundaryRule(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name funcName
		want bool
	}{
		{name: "Fuzz", want: true},
		{name: "FuzzParse", want: true},
		{name: "Fuzz_underscore", want: true},
		{name: "FuzzÉclair", want: true},
		{name: "Fuzzy", want: false},
		{name: "Fuzzédaux", want: false},
		{name: "fuzz", want: false},
		{name: "Parse", want: false},
	}
	for _, test := range tests {
		assert.Equal(t, test.want, isFuzzName(test.name), string(test.name))
	}
}

// param is one declared parameter field for a fabricated signature.
func param(kind ast.Expr, names ...string) *ast.Field {
	idents := make([]*ast.Ident, 0, len(names))
	for _, name := range names {
		idents = append(idents, ast.NewIdent(name))
	}
	return &ast.Field{Names: idents, Type: kind}
}

// declOf fabricates a free function declaration with the given parameters.
func declOf(name string, at token.Pos, params ...*ast.Field) *ast.FuncDecl {
	return &ast.FuncDecl{
		Name: ast.NewIdent(name),
		Type: &ast.FuncType{Func: at, Params: &ast.FieldList{List: params}},
		Body: &ast.BlockStmt{},
	}
}

// TestTakesTestingFRequiresExactlyOneTestingFParameter pins the signature
// rule: one parameter field, one name at most, typed *testing.F.
func TestTakesTestingFRequiresExactlyOneTestingFParameter(t *testing.T) {
	t.Parallel()

	fType := ast.NewIdent("F")
	sType := ast.NewIdent("string")
	pass := &analysis.Pass{TypesInfo: &types.Info{Types: map[ast.Expr]types.TypeAndValue{
		fType: {Type: testingFType()},
		sType: {Type: types.Typ[types.String]},
	}}}

	assert.True(t, takesTestingF(pass, declOf("Fuzz", token.NoPos, param(fType, "f"))))
	assert.False(t, takesTestingF(pass, declOf("Fuzz", token.NoPos)), "no parameters")
	assert.False(
		t,
		takesTestingF(pass, declOf("Fuzz", token.NoPos, param(fType, "f"), param(sType, "s"))),
		"two fields",
	)
	assert.False(t, takesTestingF(pass, declOf("Fuzz", token.NoPos, param(fType, "f", "g"))), "two names in one field")
	assert.False(t, takesTestingF(pass, declOf("Fuzz", token.NoPos, param(sType, "s"))), "not a *testing.F")
}

// TestIsTestingFAcceptsOnlyTheTestingFPointer pins the type rule: exactly a
// pointer to testing's F — never a non-pointer, an unnamed or foreign
// pointee, a package-less universe type, or a missing type.
func TestIsTestingFAcceptsOnlyTheTestingFPointer(t *testing.T) {
	t.Parallel()

	foreign := types.NewPackage("example.com/engine", "engine")
	foreignF := types.NewNamed(types.NewTypeName(token.NoPos, foreign, "F", nil), types.NewStruct(nil, nil), nil)

	assert.True(t, isTestingF(testingFType()))
	assert.False(t, isTestingF(nil), "unresolved type")
	assert.False(t, isTestingF(types.Typ[types.String]), "non-pointer")
	assert.False(t, isTestingF(types.NewPointer(types.Typ[types.String])), "unnamed pointee")
	assert.False(t, isTestingF(types.NewPointer(types.Universe.Lookup("error").Type())), "package-less universe type")
	assert.False(t, isTestingF(types.NewPointer(foreignF)), "foreign package")
}

// TestIsFuzzTargetRequiresATestFileDeclaration pins the whole shape: only a
// bodied free function, go-named, *testing.F-typed, in a _test.go file.
func TestIsFuzzTargetRequiresATestFileDeclaration(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	inProduction := fset.AddFile("a.go", -1, 100).Pos(1)
	inTest := fset.AddFile("a_test.go", -1, 100).Pos(1)
	fType := ast.NewIdent("F")
	pass := &analysis.Pass{Fset: fset, TypesInfo: &types.Info{Types: map[ast.Expr]types.TypeAndValue{
		fType: {Type: testingFType()},
	}}}

	assert.True(t, isFuzzTarget(pass, declOf("Fuzz", inTest, param(fType, "f"))))
	assert.False(t, isFuzzTarget(pass, declOf("Fuzz", inProduction, param(fType, "f"))), "production file")
	assert.False(t, isFuzzTarget(pass, declOf("Fuzzy", inTest, param(fType, "f"))), "boundary-failing name")

	method := declOf("Fuzz", inTest, param(fType, "f"))
	method.Recv = &ast.FieldList{List: []*ast.Field{param(ast.NewIdent("Engine"), "e")}}
	assert.False(t, isFuzzTarget(pass, method), "method")

	bodiless := declOf("Fuzz", inTest, param(fType, "f"))
	bodiless.Body = nil
	assert.False(t, isFuzzTarget(pass, bodiless), "no body")
}

// TestInTestFileReadsTheNameTheFileCannotRewrite pins the "what the go tool would run" test to the FileSet's own
// entry for a file. A //line directive compiles to exactly the
// AddLineColumnInfo calls made here: fset.Position reads that alternative
// information and token.File.Name() ignores it, so the disagreement below is
// the one a directive produces, built directly rather than parsed.
//
// Both directions are asserted because reading the rewritten name is wrong in
// both: it drops a target `go test -fuzz` really does run, and claims one it never would.
func TestInTestFileReadsTheNameTheFileCannotRewrite(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	shipped := fset.AddFile("shipped.go", -1, 100)
	shipped.AddLineColumnInfo(0, "zz_test.go", 1, 1)
	tested := fset.AddFile("shipped_test.go", -1, 100)
	tested.AddLineColumnInfo(0, "nottest.go", 1, 1)
	pass := &analysis.Pass{Fset: fset}

	assert.Equal(t, "zz_test.go", fset.Position(shipped.Pos(1)).Filename,
		"the position machinery does adopt the claimed name — without this the rest asserts nothing")
	assert.Equal(t, "nottest.go", fset.Position(tested.Pos(1)).Filename,
		"and in the other direction too")

	assert.False(t, inTestFile(pass, declOf("Fuzz", shipped.Pos(1))),
		"compiled source claiming a test name holds no fuzz target")
	assert.True(t, inTestFile(pass, declOf("Fuzz", tested.Pos(1))),
		"a test file claiming a source name still holds one")
}
