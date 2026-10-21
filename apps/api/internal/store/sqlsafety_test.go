package store_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// sqlFragmentMethods は第 1 引数が SQL 断片として解釈される GORM のメソッド。
// ここに動的な文字列(fmt.Sprintf や + 連結、変数)を渡すと SQL インジェクションになる。
// gosec は database/sql しか見ないので、GORM の呼び出しはこのテストで検査する。
var sqlFragmentMethods = map[string]bool{
	"Where": true, "Or": true, "Not": true, "Order": true, "Select": true,
	"Joins": true, "InnerJoins": true, "Group": true, "Having": true,
	"Distinct": true, "Omit": true, "Pluck": true,
}

// TestSQLFragmentsAreConstant は store パッケージ内で上記メソッドの第 1 引数が
// 文字列リテラル、定数、または非文字列の値(構造体や clause 式)であることを検査する。
// 並び順のように実行時に選ぶ必要があるものは、許可リストの map から定数を引いて渡す。
func TestSQLFragmentsAreConstant(t *testing.T) {
	files, err := filepath.Glob("*.go")
	if err != nil {
		t.Fatal(err)
	}
	fset := token.NewFileSet()
	for _, name := range files {
		if strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, err := os.ReadFile(name)
		if err != nil {
			t.Fatal(err)
		}
		file, err := parser.ParseFile(fset, name, src, 0)
		if err != nil {
			t.Fatal(err)
		}
		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) == 0 {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || !sqlFragmentMethods[sel.Sel.Name] {
				return true
			}
			if isDynamicString(call.Args[0]) {
				t.Errorf("%s: %s の第 1 引数が動的な文字列です。リテラルか定数にし、値は ? で渡してください",
					fset.Position(call.Pos()), sel.Sel.Name)
			}
			return true
		})
	}
}

// isDynamicString は SQL 断片として危険な式(文字列の連結、fmt.Sprintf 等の関数呼び出し)を検出する。
// 識別子は定数か構造体変数かを AST だけでは区別できないので許可し、
// 代わりに命名規則(SQL 断片の定数は sql で始める)をレビューで確認する。
func isDynamicString(expr ast.Expr) bool {
	switch e := expr.(type) {
	case *ast.BasicLit, *ast.Ident, *ast.CompositeLit, *ast.UnaryExpr:
		return false
	case *ast.SelectorExpr:
		return false
	case *ast.BinaryExpr:
		return e.Op == token.ADD
	case *ast.CallExpr:
		// clause.Expr{...} や gorm.Expr("? + 1", x) のような安全な構築は許す
		if sel, ok := e.Fun.(*ast.SelectorExpr); ok {
			if pkg, ok := sel.X.(*ast.Ident); ok && (pkg.Name == "clause" || pkg.Name == "gorm") {
				return false
			}
		}
		return true
	case *ast.IndexExpr:
		// 許可リストの map からの取り出し(sqlOrderBy[key])は許す
		return false
	default:
		return true
	}
}
