package env

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

// TestEnvReadsOnlyInEnvPackage enforces golden principle 1: production code
// outside internal/env must not read the process environment directly.
func TestEnvReadsOnlyInEnvPackage(t *testing.T) {
	forbidden := map[string]bool{"Getenv": true, "LookupEnv": true, "Environ": true}
	var violations []string
	for _, root := range []string{"../../cmd", "../../internal"} {
		err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				if filepath.Base(path) == "env" && filepath.Base(filepath.Dir(path)) == "internal" {
					return filepath.SkipDir
				}
				return nil
			}
			if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return err
			}
			ast.Inspect(f, func(n ast.Node) bool {
				sel, ok := n.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				if pkg, ok := sel.X.(*ast.Ident); ok && pkg.Name == "os" && forbidden[sel.Sel.Name] {
					violations = append(violations, fset.Position(sel.Pos()).String()+": os."+sel.Sel.Name)
				}
				return true
			})
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", root, err)
		}
	}
	for _, v := range violations {
		t.Errorf("direct env read outside internal/env (use env.Get/env.Debug): %s", v)
	}
}
