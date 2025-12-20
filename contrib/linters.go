package main

import (
	"go/ast"
	"go/token"
	"go/types"
	"reflect"
	"strings"

	"golang.org/x/tools/go/analysis"
	"golang.org/x/tools/go/analysis/multichecker"
)

type indirectCalls struct {
	methods   map[token.Pos]string
	functions map[string]token.Pos
}

var PanicAnalyzer = &analysis.Analyzer{
	Name:       "panic",
	Doc:        "finds goroutines that do not initialize the panic handler",
	Run:        runPanic,
	ResultType: reflect.TypeFor[*indirectCalls](),
}

var PanicIndirectAnalyzer = &analysis.Analyzer{
	Name:     "panicindirect",
	Doc:      "finds functions called as goroutines that do not initialize the panic handler",
	Run:      runPanicIndirect,
	Requires: []*analysis.Analyzer{PanicAnalyzer},
}

func runPanic(pass *analysis.Pass) (any, error) {
	var calls indirectCalls

	calls.methods = make(map[token.Pos]string)
	calls.functions = make(map[string]token.Pos)

	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			g, ok := n.(*ast.GoStmt)
			if !ok {
				return true
			}

			var block *ast.BlockStmt

			expr := g.Call.Fun
			if e, ok := expr.(*ast.ParenExpr); ok {
				expr = e.X
			}

			switch e := expr.(type) {
			case *ast.FuncLit:
				block = e.Body
			case *ast.SelectorExpr:
				sel, ok := pass.TypesInfo.Selections[e]
				if ok {
					f, ok := sel.Obj().(*types.Func)
					if ok {
						calls.methods[f.Pos()] = f.Name()
					}
				}
			case *ast.Ident:
				block = inlineFuncBody(e)
				if block == nil {
					calls.functions[e.Name] = e.NamePos
				}
			}

			if block == nil {
				return true
			}

			if !isPanicHandlerInstall(block.List[0]) {
				path := pass.Fset.File(block.Pos()).Name()
				if !strings.HasSuffix(path, "_test.go") {
					pass.Report(panicDiag(block.Pos()))
				}
			}

			return true
		})
	}

	return &calls, nil
}

func runPanicIndirect(pass *analysis.Pass) (any, error) {
	calls := pass.ResultOf[PanicAnalyzer].(*indirectCalls)

	for _, file := range pass.Files {
		if strings.HasSuffix(file.Name.Name, "_test") {
			continue
		}
		for _, decl := range file.Decls {
			if f, ok := decl.(*ast.FuncDecl); ok {
				if _, ok := calls.methods[f.Name.Pos()]; ok {
					delete(calls.methods, f.Name.Pos())
				} else if _, ok := calls.functions[f.Name.Name]; ok {
					delete(calls.functions, f.Name.Name)
				} else {
					continue
				}
				if !isPanicHandlerInstall(f.Body.List[0]) {
					path := pass.Fset.File(f.Body.Pos()).Name()
					if !strings.HasSuffix(path, "_test.go") {
						pass.Report(panicDiag(f.Body.Pos()))
					}
				}
			}
		}
	}

	return nil, nil
}

func panicDiag(pos token.Pos) analysis.Diagnostic {
	return analysis.Diagnostic{
		Pos:      pos,
		Category: "panic",
		Message:  "missing defer log.PanicHandler() as first statement",
	}
}

func inlineFuncBody(s *ast.Ident) *ast.BlockStmt {
	if s.Obj == nil || s.Obj.Decl == nil {
		return nil
	}
	d, ok := s.Obj.Decl.(*ast.AssignStmt)
	if !ok {
		return nil
	}
	for _, r := range d.Rhs {
		if f, ok := r.(*ast.FuncLit); ok {
			return f.Body
		}
	}
	return nil
}

func isPanicHandlerInstall(stmt ast.Stmt) bool {
	d, ok := stmt.(*ast.DeferStmt)
	if !ok {
		return false
	}
	s, ok := d.Call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	i, ok := s.X.(*ast.Ident)
	if !ok {
		return false
	}
	return i.Name == "log" && s.Sel.Name == "PanicHandler"
}

func main() {
	multichecker.Main(
		PanicAnalyzer,
		PanicIndirectAnalyzer,
	)
}
