package codegen

import (
	"fmt"
	"liaf/pkg/ast"
)

type liafTestCase struct {
	name string
	fn   string
}

func (g *Generator) genTest(t *ast.TestDecl, index int) {
	fnName := fmt.Sprintf("_liaf_test_%d", index)
	g.sb.WriteString(fmt.Sprintf("func %s() {\n", fnName))
	g.indent++
	for _, stmt := range t.Body {
		g.genStmt(stmt)
	}
	g.indent--
	g.sb.WriteString("}\n\n")
	g.testCases = append(g.testCases, liafTestCase{name: t.Name, fn: fnName})
}
