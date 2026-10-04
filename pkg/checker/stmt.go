package checker

import (
	"fmt"
	"sort"
	"strings"

	"liaf/pkg/ast"
)

func (c *Checker) block(body []ast.Stmt) { c.push(); c.statements(body); c.pop() }

func (c *Checker) statements(body []ast.Stmt) {
	for _, st := range body {
		switch s := st.(type) {
		case *ast.LetStmt:
			if s.Type == nil {
				inferred := c.expr(s.Value, nil)
				if inferred == nil || name(inferred) == "void" {
					c.error(s, "E_TYPE_INFERENCE_FAILED", fmt.Sprintf("Não foi possível inferir o tipo da variável %q", s.Name))
					s.Type = primitive("void")
				} else {
					s.Type = inferred
				}
			} else {
				c.validType(s.Type, false)
				c.require(s, c.expr(s.Value, s.Type), s.Type)
			}
			c.define(s.Name, s.Type, s, IsResult(s.Type))
		case *ast.SetStmt:
			b := c.lookup(s.Name)
			if b == nil {
				c.error(s, "E_UNDEFINED_SYMBOL", s.Name)
				c.expr(s.Value, nil)
			} else {
				if b.pending {
					c.error(s, "E_UNHANDLED_RESULT", s.Name)
				}
				c.require(s, c.expr(s.Value, b.typ), b.typ)
				b.pending = IsResult(b.typ)
			}
		case *ast.ReturnStmt:
			c.require(s, c.expr(s.Value, c.fn.ReturnType), c.fn.ReturnType)
			c.consume(s.Value)
			for _, scope := range c.scopes {
				for id, b := range scope {
					if b.pending {
						c.error(s, "E_UNHANDLED_RESULT", id+" is unhandled on this return path")
					}
				}
			}
		case *ast.ExprStmt:
			if IsResult(c.expr(s.Expr, nil)) {
				c.error(s, "E_UNHANDLED_RESULT", "Use match or return for fallible operations")
			}
		case *ast.IfStmt:
			c.require(s, c.expr(s.Condition, nil), primitive("bool"))
			before := c.pending()
			c.block(s.Then)
			afterThen := c.pending()
			restore(before)
			c.block(s.Else)
			if returns(s.Then) { /* only the else path continues */
			} else if returns(s.Else) {
				restore(afterThen)
			} else {
				merge(afterThen, c.pending())
			}
		case *ast.LoopStmt:
			before := c.pending()
			c.push()
			switch s.Kind {
			case "while":
				c.require(s, c.expr(s.Condition, nil), primitive("bool"))
			case "for-range":
				c.require(s, c.expr(s.Start, nil), primitive("int"))
				c.require(s, c.expr(s.End, nil), primitive("int"))
				c.define(s.Name, primitive("int"), s, false)
			case "for-each":
				a := parts(c.expr(s.Collection, nil), "list", 1)
				if a == nil {
					c.error(s, "E_TYPE_MISMATCH", "for-each requires list")
				} else {
					c.define(s.Name, a[0], s, false)
				}
			}
			c.loops++
			c.statements(s.Body)
			c.loops--
			c.pop()
			merge(before, c.pending())
		case *ast.LoopControl:
			if c.loops == 0 {
				c.error(s, "E_LOOP_CONTROL", s.Kind+" outside loop")
			}
		case *ast.MatchStmt:
			valType := c.expr(s.Value, nil)
			if len(s.Cases) > 0 {
				c.checkMatchStmtCases(s, valType)
				continue
			}
			if s.IsOption {
				a := parts(valType, "option", 1)
				if a == nil {
					c.error(s, "E_TYPE_MISMATCH", "match requires option")
					continue
				}
				c.consume(s.Value)
				before := c.pending()
				c.push()
				c.define(s.OKName, a[0], s, false)
				c.statements(s.OK)
				c.pop()
				afterOK := c.pending()
				restore(before)
				c.push()
				if s.ErrName != "" {
					c.define(s.ErrName, primitive("void"), s, false)
				}
				c.statements(s.Err)
				c.pop()
				if returns(s.OK) {
				} else if returns(s.Err) {
					restore(afterOK)
				} else {
					merge(afterOK, c.pending())
				}
				continue
			}
			a := parts(valType, "result", 2)
			if a == nil {
				c.error(s, "E_TYPE_MISMATCH", "match requires result")
				continue
			}
			c.consume(s.Value)
			before := c.pending()
			c.push()
			c.define(s.OKName, a[0], s, IsResult(a[0]))
			c.statements(s.OK)
			c.pop()
			afterOK := c.pending()
			restore(before)
			c.push()
			c.define(s.ErrName, a[1], s, IsResult(a[1]))
			c.statements(s.Err)
			c.pop()
			if returns(s.OK) {
			} else if returns(s.Err) {
				restore(afterOK)
			} else {
				merge(afterOK, c.pending())
			}
		case *ast.DBTransactionStmt:
			c.dbTransaction(s)
		case *ast.SpawnStmt:
			c.effect(s, "spawn")
			c.require(s, c.expr(s.Call, nil), primitive("void"))
		case *ast.SendStmt:
			a := parts(c.expr(s.Channel, nil), "chan", 1)
			v := c.expr(s.Value, nil)
			if a == nil {
				c.error(s, "E_CHANNEL_TYPE_MISMATCH", "send requires channel")
			} else {
				c.require(s, v, a[0])
			}
		}
	}
}

func returns(body []ast.Stmt) bool {
	for _, st := range body {
		switch s := st.(type) {
		case *ast.ReturnStmt:
			return true
		case *ast.IfStmt:
			if returns(s.Then) && returns(s.Else) {
				return true
			}
		case *ast.MatchStmt:
			if len(s.Cases) > 0 {
				allReturn := true
				for _, c := range s.Cases {
					if !returns(c.Body) {
						if len(c.Body) > 0 {
							if _, isExpr := c.Body[len(c.Body)-1].(*ast.ExprStmt); isExpr {
								continue
							}
						}
						allReturn = false
						break
					}
				}
				if allReturn {
					return true
				}
			} else if returns(s.OK) && returns(s.Err) {
				return true
			}
		case *ast.DBTransactionStmt:
			// Ao contrario de um loop, o corpo da transacao sempre executa,
			// entao um return la dentro satisfaz o retorno da funcao.
			if returns(s.Body) {
				return true
			}
		}
	}
	return false
}

func (c *Checker) checkMatchStmtCases(s *ast.MatchStmt, valType ast.Type) {
	c.consume(s.Value)
	typeName := name(valType)

	if enumDecl := c.findEnum(typeName); enumDecl != nil {
		variantsMap := map[string]ast.EnumVariant{}
		unhandled := map[string]bool{}
		for _, v := range enumDecl.Variants {
			variantsMap[v.Name] = v
			unhandled[v.Name] = true
		}

		hasWildcard := false
		for _, cs := range s.Cases {
			if cs.Pattern == "_" {
				hasWildcard = true
				c.push()
				c.statements(cs.Body)
				if c.fn != nil && name(c.fn.ReturnType) != "void" && len(cs.Body) > 0 {
					if es, ok := cs.Body[len(cs.Body)-1].(*ast.ExprStmt); ok {
						exprT := c.expr(es.Expr, c.fn.ReturnType)
						c.require(es, exprT, c.fn.ReturnType)
					}
				}
				c.pop()
				continue
			}

			varName := cs.Pattern
			if strings.Contains(varName, ".") {
				parts := strings.Split(varName, ".")
				if parts[0] != enumDecl.Name {
					c.error(s, "E_TYPE_MISMATCH", fmt.Sprintf("Pattern %s does not belong to enum %s", cs.Pattern, enumDecl.Name))
					continue
				}
				varName = parts[1]
			}

			variant, ok := variantsMap[varName]
			if !ok {
				c.error(s, "E_UNKNOWN_VARIANT", fmt.Sprintf("Enum %s has no variant %s", enumDecl.Name, varName))
				continue
			}

			if !unhandled[varName] {
				c.error(s, "E_DUPLICATE_CASE", fmt.Sprintf("Duplicate case for variant %s", varName))
			}
			delete(unhandled, varName)

			c.push()
			if len(cs.Variables) != len(variant.Fields) {
				c.error(s, "E_ARITY_MISMATCH", fmt.Sprintf("Variant %s expects %d variables, got %d", varName, len(variant.Fields), len(cs.Variables)))
			} else {
				for i, f := range variant.Fields {
					c.define(cs.Variables[i], f.Type, s, false)
				}
			}
			c.statements(cs.Body)
			if c.fn != nil && name(c.fn.ReturnType) != "void" && len(cs.Body) > 0 {
				if es, ok := cs.Body[len(cs.Body)-1].(*ast.ExprStmt); ok {
					exprT := c.expr(es.Expr, c.fn.ReturnType)
					c.require(es, exprT, c.fn.ReturnType)
				}
			}
			c.pop()
		}

		if !hasWildcard && len(unhandled) > 0 {
			var missing []string
			for m := range unhandled {
				missing = append(missing, m)
			}
			sort.Strings(missing)
			c.error(s, "E_NONEXHAUSTIVE_MATCH", fmt.Sprintf("Match on enum %s is not exhaustive; missing: %s", enumDecl.Name, strings.Join(missing, ", ")))
		}
		return
	}

	if optParts := parts(valType, "option", 1); optParts != nil {
		unhandled := map[string]bool{"some": true, "none": true}
		hasWildcard := false
		for _, cs := range s.Cases {
			if cs.Pattern == "_" {
				hasWildcard = true
				c.push()
				c.statements(cs.Body)
				c.pop()
				continue
			}
			if cs.Pattern != "some" && cs.Pattern != "none" {
				c.error(s, "E_UNKNOWN_VARIANT", fmt.Sprintf("Option has no variant %s", cs.Pattern))
				continue
			}
			if !unhandled[cs.Pattern] {
				c.error(s, "E_DUPLICATE_CASE", fmt.Sprintf("Duplicate case for %s", cs.Pattern))
			}
			delete(unhandled, cs.Pattern)

			c.push()
			if cs.Pattern == "some" {
				if len(cs.Variables) > 0 && cs.Variables[0] != "_" {
					c.define(cs.Variables[0], optParts[0], s, false)
				}
			} else if cs.Pattern == "none" {
				if len(cs.Variables) > 0 && cs.Variables[0] != "_" {
					c.define(cs.Variables[0], primitive("void"), s, false)
				}
			}
			c.statements(cs.Body)
			c.pop()
		}
		if !hasWildcard && len(unhandled) > 0 {
			c.error(s, "E_NONEXHAUSTIVE_MATCH", "Match on option is not exhaustive")
		}
		return
	}

	if resParts := parts(valType, "result", 2); resParts != nil {
		unhandled := map[string]bool{"ok": true, "err": true}
		hasWildcard := false
		for _, cs := range s.Cases {
			if cs.Pattern == "_" {
				hasWildcard = true
				c.push()
				c.statements(cs.Body)
				c.pop()
				continue
			}
			if cs.Pattern != "ok" && cs.Pattern != "err" {
				c.error(s, "E_UNKNOWN_VARIANT", fmt.Sprintf("Result has no variant %s", cs.Pattern))
				continue
			}
			if !unhandled[cs.Pattern] {
				c.error(s, "E_DUPLICATE_CASE", fmt.Sprintf("Duplicate case for %s", cs.Pattern))
			}
			delete(unhandled, cs.Pattern)

			c.push()
			if cs.Pattern == "ok" {
				if len(cs.Variables) > 0 && cs.Variables[0] != "_" {
					c.define(cs.Variables[0], resParts[0], s, IsResult(resParts[0]))
				}
			} else if cs.Pattern == "err" {
				if len(cs.Variables) > 0 && cs.Variables[0] != "_" {
					c.define(cs.Variables[0], resParts[1], s, IsResult(resParts[1]))
				}
			}
			c.statements(cs.Body)
			c.pop()
		}
		if !hasWildcard && len(unhandled) > 0 {
			c.error(s, "E_NONEXHAUSTIVE_MATCH", "Match on result is not exhaustive")
		}
		return
	}

	c.error(s, "E_TYPE_MISMATCH", fmt.Sprintf("match requires enum, option or result, received %s", typeName))
}
