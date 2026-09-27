package checker

import (
	"liaf/pkg/ast"
)

// Implementação dos métodos de CheckContext em Checker
func (c *Checker) Expr(e ast.Expr, want ast.Type) ast.Type { return c.expr(e, want) }
func (c *Checker) Require(n ast.Node, actual, want ast.Type) { c.require(n, actual, want) }
func (c *Checker) Error(n ast.Node, code, msg string) { c.error(n, code, msg) }
func (c *Checker) Arity(v *ast.CallExpr, n int) bool { return c.arity(v, n) }
func (c *Checker) Effect(n ast.Node, eff string) { c.effect(n, eff) }
func (c *Checker) TypeArg(e ast.Expr) ast.Type { return c.typeArg(e) }
func (c *Checker) FindStruct(name string) *ast.StructDecl { return c.Info.Structs[name] }
func (c *Checker) FindFunc(name string) *ast.FuncDecl { return c.Info.Functions[name] }
func (c *Checker) TypeAt(e ast.Expr) ast.Type { return c.Info.Types[e] }
