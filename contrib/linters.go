package main

import (
	"go/ast"
	"go/token"
	"go/types"

	"golang.org/x/tools/go/analysis"
)

var PanicAnalyzer = &analysis.Analyzer{
	Name: "panic",
	Doc:  "finds goroutines that do not initialize the panic handler",
	Run:  runPanic,
}

func runPanic(pass *analysis.Pass) (interface{}, error) {
	methods := make(map[token.Pos]string)
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
						methods[f.Pos()] = f.Name()
					}
				}
			case *ast.Ident:
				block = inlineFuncBody(e)
			}

			if block == nil {
				return true
			}

			if !isPanicHandlerInstall(block.List[0]) {
				pass.Report(panicDiag(block.Pos()))
			}

			return true
		})
	}
	for _, file := range pass.Files {
		ast.Inspect(file, func(n ast.Node) bool {
			f, ok := n.(*ast.FuncDecl)
			if !ok {
				return false
			}
			_, found := methods[f.Name.Pos()]
			if !found {
				return false
			}
			delete(methods, f.Name.Pos())
			if !isPanicHandlerInstall(f.Body.List[0]) {
				pass.Report(panicDiag(f.Body.Pos()))
			}
			return false
		})
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

// golang-lint required plugin api
type analyzerPlugin struct{}

// This must be implemented
func (*analyzerPlugin) GetAnalyzers() []*analysis.Analyzer {
	return []*analysis.Analyzer{
		PanicAnalyzer,
	}
}

// This must be defined and named 'AnalyzerPlugin'
var AnalyzerPlugin analyzerPlugin
