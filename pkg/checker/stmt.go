package checker

import (
	"fmt"

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
			if returns(s.OK) && returns(s.Err) {
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
