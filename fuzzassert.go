// Package fuzzassert provides a go/analysis analyzer enforcing that a fuzz
// target asserts a property. A `func Fuzz*` whose f.Fuzz callback merely
// invokes the code under test — `f.Fuzz(func(t *testing.T, b []byte) { _, _ =
// Parse(b) })` — executes on every input and can fail on none: it verifies
// nothing while reading as fuzz coverage. That shape is precisely how a
// fuzz-target mandate gets gamed, which is why this analyzer must exist
// BEFORE any rule counts fuzz targets.
//
// A callback asserts when it fails the test itself (t.Error/t.Errorf/
// t.Fatal/t.Fatalf/t.Fail/t.FailNow, or a testify assertion), or when it
// hands its *testing.T to another function — a helper that receives the T is
// presumed to assert through it, and the analyzer fails open rather than
// judge the helper across call boundaries. t.Skip and t.Log fail nothing and
// therefore assert nothing. A callback passed by NAME is resolved when it is
// declared in the same package; an unresolvable reference fails open.
//
// A fuzz target is what the go tool itself would run, nothing looser: a free
// function in a _test.go file, named "Fuzz" followed by a non-lowercase rune
// (or nothing at all), taking exactly one *testing.F parameter — and only a
// Fuzz call made ON that *testing.F is judged. Production code that merely
// resembles fuzzing (a FuzzDrive free function, an engine.Fuzz method on a
// custom type, a Fuzzy-prefixed name) is invisible to this analyzer.
package fuzzassert

import (
	"go/ast"
	"go/types"
	"strings"
	"unicode"
	"unicode/utf8"

	goyze "github.com/gomatic/go-yze"
	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/passes/inspect"
	"golang.org/x/tools/go/ast/inspector"
)

// message is the diagnostic for an assertion-free fuzz callback.
const message = "fuzz target invokes without asserting a property; a callback that cannot fail verifies nothing"

// testifyPath is the module whose assertions count as assertions.
const testifyPath = "github.com/stretchr/testify"

// failingMethods are the *testing.T methods that FAIL a test; a callback
// calling none of them (and delegating to no helper) has no way to fail.
var failingMethods = map[string]bool{
	"Error": true, "Errorf": true, "Fatal": true, "Fatalf": true, "Fail": true, "FailNow": true,
}

// Analyzer reports fuzz targets whose callbacks assert nothing.
var Analyzer = &analysis.Analyzer{
	Name:     "fuzzassert",
	Doc:      "reports a Fuzz target whose f.Fuzz callback invokes the code under test without asserting a property",
	Requires: []*analysis.Analyzer{inspect.Analyzer},
	Run:      run,
}

// Registration declares this analyzer to the yze framework.
var Registration = goyze.Registration{
	Name:       "fuzzassert",
	Categories: []goyze.Category{"tests"},
	URL:        "https://docs.gomatic.dev/yze/fuzzassert",
	Analyzer:   Analyzer,
}

// run inspects every f.Fuzz call in every fuzz target.
func run(pass *analysis.Pass) (any, error) {
	ins, _ := pass.ResultOf[inspect.Analyzer].(*inspector.Inspector)
	ins.Preorder([]ast.Node{(*ast.FuncDecl)(nil)}, func(n ast.Node) {
		decl, _ := n.(*ast.FuncDecl)
		if isFuzzTarget(pass, decl) {
			checkTarget(pass, decl)
		}
	})
	return nil, nil
}

// isFuzzTarget reports what the go tool would run as a fuzz target: a free
// function named per the go rule, taking exactly one *testing.F parameter,
// declared in a _test.go file.
func isFuzzTarget(pass *analysis.Pass, decl *ast.FuncDecl) bool {
	return decl.Recv == nil && decl.Body != nil &&
		isFuzzName(funcName(decl.Name.Name)) &&
		takesTestingF(pass, decl) &&
		inTestFile(pass, decl)
}

// inTestFile reports a declaration located in a _test.go file.
func inTestFile(pass *analysis.Pass, decl *ast.FuncDecl) bool {
	return strings.HasSuffix(pass.Fset.Position(decl.Pos()).Filename, "_test.go")
}

// funcName is the identifier of a declared function.
type funcName string

// isFuzzName applies the go tool's rule for fuzz identifiers: "Fuzz" followed
// by a non-lowercase rune, or exactly "Fuzz" — so FuzzyMatch is an ordinary
// function, not a target.
func isFuzzName(name funcName) bool {
	rest, ok := strings.CutPrefix(string(name), "Fuzz")
	if !ok {
		return false
	}
	r, _ := utf8.DecodeRuneInString(rest)
	return rest == "" || !unicode.IsLower(r)
}

// takesTestingF reports a signature of exactly one *testing.F parameter.
func takesTestingF(pass *analysis.Pass, decl *ast.FuncDecl) bool {
	params := decl.Type.Params.List
	if len(params) != 1 || len(params[0].Names) > 1 {
		return false
	}
	return isTestingF(pass.TypesInfo.TypeOf(params[0].Type))
}

// isTestingF reports the type *testing.F exactly.
func isTestingF(t types.Type) bool {
	ptr, ok := types.Unalias(t).(*types.Pointer)
	if !ok {
		return false
	}
	named, ok := types.Unalias(ptr.Elem()).(*types.Named)
	return ok && named.Obj().Pkg() != nil &&
		named.Obj().Pkg().Path() == "testing" && named.Obj().Name() == "F"
}

// checkTarget reports each f.Fuzz callback in the target that asserts nothing.
func checkTarget(pass *analysis.Pass, decl *ast.FuncDecl) {
	ast.Inspect(decl.Body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if ok && isFuzzCall(pass, call) && len(call.Args) == 1 {
			checkCallback(pass, call, call.Args[0])
		}
		return true
	})
}

// isFuzzCall reports a Fuzz(...) call made on a *testing.F — a Fuzz method
// on any other type is somebody's domain verb, not the fuzzer.
func isFuzzCall(pass *analysis.Pass, call *ast.CallExpr) bool {
	sel, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr)
	return ok && sel.Sel.Name == "Fuzz" && isTestingF(pass.TypesInfo.TypeOf(sel.X))
}

// checkCallback resolves the callback expression to a body and judges it. An
// unresolvable callback fails open: silence beats a false positive on a gate.
func checkCallback(pass *analysis.Pass, at *ast.CallExpr, callback ast.Expr) {
	body := callbackBody(pass, callback)
	if body == nil {
		return
	}
	if !asserts(pass, body) {
		pass.Reportf(at.Pos(), "%s", message)
	}
}

// callbackBody finds the function body behind the callback: a literal
// directly, or the same-package declaration a plain identifier names.
func callbackBody(pass *analysis.Pass, callback ast.Expr) *ast.BlockStmt {
	switch fn := ast.Unparen(callback).(type) {
	case *ast.FuncLit:
		return fn.Body
	case *ast.Ident:
		return declaredBody(pass, fn)
	}
	return nil
}

// declaredBody finds the body of the same-package function ident names.
func declaredBody(pass *analysis.Pass, ident *ast.Ident) *ast.BlockStmt {
	target := pass.TypesInfo.ObjectOf(ident)
	if target == nil {
		return nil
	}
	for _, file := range pass.Files {
		if body := bodyDeclaredAt(pass, file, target); body != nil {
			return body
		}
	}
	return nil
}

// bodyDeclaredAt finds the function declaration in file whose object is target.
func bodyDeclaredAt(pass *analysis.Pass, file *ast.File, target types.Object) *ast.BlockStmt {
	for _, decl := range file.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if ok && pass.TypesInfo.ObjectOf(fn.Name) == target {
			return fn.Body
		}
	}
	return nil
}

// asserts reports whether the body can fail the test: a failing *testing.T
// method, a testify assertion, or its T handed to another function.
func asserts(pass *analysis.Pass, body *ast.BlockStmt) bool {
	var found bool
	ast.Inspect(body, func(n ast.Node) bool {
		call, ok := n.(*ast.CallExpr)
		if ok && !found {
			found = isAssertion(pass, call)
		}
		return !found
	})
	return found
}

// isAssertion reports whether one call can fail the test.
func isAssertion(pass *analysis.Pass, call *ast.CallExpr) bool {
	if sel, ok := ast.Unparen(call.Fun).(*ast.SelectorExpr); ok {
		if isFailingTestingCall(pass, sel) || isTestifyCall(pass, sel) {
			return true
		}
	}
	return delegatesT(pass, call)
}

// isFailingTestingCall reports a failing method called on a *testing.T (or
// *testing.F) receiver.
func isFailingTestingCall(pass *analysis.Pass, sel *ast.SelectorExpr) bool {
	return failingMethods[sel.Sel.Name] && isTestingType(pass.TypesInfo.TypeOf(sel.X))
}

// isTestifyCall reports a call into the testify module (assert/require in
// function or method form).
func isTestifyCall(pass *analysis.Pass, sel *ast.SelectorExpr) bool {
	fn, ok := pass.TypesInfo.ObjectOf(sel.Sel).(*types.Func)
	if !ok || fn.Pkg() == nil {
		return false
	}
	path := fn.Pkg().Path()
	return path == testifyPath || strings.HasPrefix(path, testifyPath+"/")
}

// delegatesT reports whether the call passes a testing.T/F/B value onward —
// a helper receiving the T is presumed to assert through it.
func delegatesT(pass *analysis.Pass, call *ast.CallExpr) bool {
	for _, arg := range call.Args {
		if isTestingType(pass.TypesInfo.TypeOf(arg)) {
			return true
		}
	}
	return false
}

// isTestingType reports a (pointer to a) type declared by package testing.
func isTestingType(t types.Type) bool {
	if ptr, ok := types.Unalias(t).(*types.Pointer); ok {
		t = ptr.Elem()
	}
	named, ok := types.Unalias(t).(*types.Named)
	if !ok || named.Obj().Pkg() == nil {
		return false
	}
	return named.Obj().Pkg().Path() == "testing"
}
