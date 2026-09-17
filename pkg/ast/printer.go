package ast

import (
	"fmt"
	"sort"
	"strings"
)

// Format produz a representação canônica textual oficial da AST segundo a SPEC_V2
func Format(mod *Module) string {
	var sb strings.Builder
	p := &printer{sb: &sb, indent: 0}
	p.printModule(mod)
	return sb.String()
}

type printer struct {
	sb     *strings.Builder
	indent int
}

func (p *printer) writeIndent() {
	for i := 0; i < p.indent; i++ {
		p.sb.WriteString("  ")
	}
}

func (p *printer) printModule(mod *Module) {
	p.sb.WriteString("(module " + mod.Name)
	if len(mod.Decls) == 0 {
		p.sb.WriteString(")\n")
		return
	}
	p.sb.WriteString("\n")
	p.indent++
	for i, decl := range mod.Decls {
		isLast := i == len(mod.Decls)-1
		p.writeIndent()
		p.printTopLevel(decl, isLast)
		p.sb.WriteString("\n")
	}
	p.indent--
}

func (p *printer) printTopLevel(decl TopLevel, isLastDecl bool) {
	suffix := ""
	if isLastDecl {
		suffix = ")"
	}

	switch d := decl.(type) {
	case *ImportDecl:
		p.sb.WriteString(fmt.Sprintf("(import %q)%s", d.Path, suffix))
	case *StructDecl:
		p.printStruct(d, suffix)
	case *FuncDecl:
		p.printFunc(d, suffix)
	case *RouteDecl:
		p.printRoute(d, suffix)
	case *WSRouteDecl:
		p.printWSRoute(d, suffix)
	}
}

// printWSRoute imprime a forma canonica de (ws-route ...). Blocos opcionais
// vazios sao omitidos: um (on-close) sem instrucao nenhuma seria ruido, e a
// forma canonica precisa ser unica para o fmt ser idempotente.
func (p *printer) printWSRoute(w *WSRouteDecl, suffix string) {
	p.sb.WriteString(fmt.Sprintf("(ws-route %q\n", w.Path))
	p.indent++

	p.writeIndent()
	if len(w.Params) == 0 {
		p.sb.WriteString("(params)\n")
	} else {
		p.sb.WriteString("(params\n")
		p.indent++
		for i, param := range w.Params {
			p.writeIndent()
			if i == len(w.Params)-1 {
				p.sb.WriteString(fmt.Sprintf("(%s %s))\n", param.Name, param.Type.String()))
			} else {
				p.sb.WriteString(fmt.Sprintf("(%s %s)\n", param.Name, param.Type.String()))
			}
		}
		p.indent--
	}

	p.writeIndent()
	p.sb.WriteString("(effects")
	effects := append([]string(nil), w.Effects...)
	sort.Strings(effects)
	for _, eff := range effects {
		p.sb.WriteString(" " + eff)
	}
	p.sb.WriteString(")\n")

	// O ultimo bloco impresso carrega o fechamento da propria ws-route.
	last := "on-message"
	if len(w.OnClose) > 0 {
		last = "on-close"
	}

	if w.OnErrVar != "" {
		p.writeIndent()
		p.sb.WriteString("(on-err " + w.OnErrVar)
		p.printBlock(w.OnErrBody, "\n")
	}
	if len(w.OnOpen) > 0 {
		p.writeIndent()
		p.sb.WriteString("(on-open")
		p.printBlock(w.OnOpen, "\n")
	}

	p.writeIndent()
	p.sb.WriteString("(on-message " + w.MsgVar)
	if last == "on-message" {
		p.printBlock(w.OnMessage, ")"+suffix)
	} else {
		p.printBlock(w.OnMessage, "\n")
		p.writeIndent()
		p.sb.WriteString("(on-close")
		p.printBlock(w.OnClose, ")"+suffix)
	}

	p.indent--
}

func (p *printer) printStruct(s *StructDecl, suffix string) {
	p.sb.WriteString("(struct " + s.Name + "\n")
	p.indent++
	p.writeIndent()
	if len(s.Fields) == 0 {
		p.sb.WriteString("(fields))" + suffix)
		p.indent--
		return
	}
	p.sb.WriteString("(fields\n")
	p.indent++
	for i, f := range s.Fields {
		p.writeIndent()
		if i == len(s.Fields)-1 {
			p.sb.WriteString(fmt.Sprintf("(%s %s)))%s", f.Name, f.Type.String(), suffix))
		} else {
			p.sb.WriteString(fmt.Sprintf("(%s %s)\n", f.Name, f.Type.String()))
		}
	}
	p.indent -= 2
}

func (p *printer) printFunc(f *FuncDecl, suffix string) {
	p.sb.WriteString("(fn " + f.Name + "\n")
	p.indent++

	// Params
	p.writeIndent()
	if len(f.Params) == 0 {
		p.sb.WriteString("(params)\n")
	} else {
		p.sb.WriteString("(params\n")
		p.indent++
		for i, param := range f.Params {
			p.writeIndent()
			if i == len(f.Params)-1 {
				p.sb.WriteString(fmt.Sprintf("(%s %s))\n", param.Name, param.Type.String()))
			} else {
				p.sb.WriteString(fmt.Sprintf("(%s %s)\n", param.Name, param.Type.String()))
			}
		}
		p.indent--
	}

	// Returns
	p.writeIndent()
	p.sb.WriteString(fmt.Sprintf("(returns %s)\n", f.ReturnType.String()))

	// Effects
	p.writeIndent()
	p.sb.WriteString("(effects")
	if len(f.Effects) > 0 {
		effects := append([]string(nil), f.Effects...)
		sort.Strings(effects)
		for _, eff := range effects {
			p.sb.WriteString(" " + eff)
		}
	}
	p.sb.WriteString(")\n")

	// On-Err (opcional)
	if f.OnErrVar != "" {
		p.writeIndent()
		p.sb.WriteString("(on-err " + f.OnErrVar)
		p.printBlock(f.OnErrBody, "\n")
	}

	// Body
	p.writeIndent()
	if len(f.Body) == 0 {
		p.sb.WriteString("(body))" + suffix)
	} else {
		p.sb.WriteString("(body\n")
		p.indent++
		for i, stmt := range f.Body {
			p.writeIndent()
			if i == len(f.Body)-1 {
				p.printStmt(stmt, "))"+suffix)
			} else {
				p.printStmt(stmt, "")
				p.sb.WriteString("\n")
			}
		}
		p.indent--
	}

	p.indent--
}

func (p *printer) printRoute(r *RouteDecl, suffix string) {
	p.sb.WriteString(fmt.Sprintf("(route %s %q\n", r.Method, r.Path))
	p.indent++

	// Params
	p.writeIndent()
	if len(r.Params) == 0 {
		p.sb.WriteString("(params)\n")
	} else {
		p.sb.WriteString("(params\n")
		p.indent++
		for i, param := range r.Params {
			p.writeIndent()
			if i == len(r.Params)-1 {
				p.sb.WriteString(fmt.Sprintf("(%s %s))\n", param.Name, param.Type.String()))
			} else {
				p.sb.WriteString(fmt.Sprintf("(%s %s)\n", param.Name, param.Type.String()))
			}
		}
		p.indent--
	}

	// Returns
	p.writeIndent()
	p.sb.WriteString(fmt.Sprintf("(returns %s)\n", r.ReturnType.String()))

	// Effects
	p.writeIndent()
	p.sb.WriteString("(effects")
	if len(r.Effects) > 0 {
		effects := append([]string(nil), r.Effects...)
		sort.Strings(effects)
		for _, eff := range effects {
			p.sb.WriteString(" " + eff)
		}
	}
	p.sb.WriteString(")\n")

	// On-Err (opcional)
	if r.OnErrVar != "" {
		p.writeIndent()
		p.sb.WriteString("(on-err " + r.OnErrVar)
		p.printBlock(r.OnErrBody, "\n")
	}

	// Body
	p.writeIndent()
	if len(r.Body) == 0 {
		p.sb.WriteString("(body))" + suffix)
	} else {
		p.sb.WriteString("(body\n")
		p.indent++
		for i, stmt := range r.Body {
			p.writeIndent()
			if i == len(r.Body)-1 {
				p.printStmt(stmt, "))"+suffix)
			} else {
				p.printStmt(stmt, "")
				p.sb.WriteString("\n")
			}
		}
		p.indent--
	}

	p.indent--
}

func (p *printer) printStmt(stmt Stmt, suffix string) {
	switch s := stmt.(type) {
	case *LoopControl:
		p.sb.WriteString("(" + s.Kind + ")" + suffix)
	case *LoopStmt:
		p.sb.WriteString("(" + s.Kind)
		switch s.Kind {
		case "while":
			p.sb.WriteString(" " + formatExpr(s.Condition))
		case "for-range":
			p.sb.WriteString(" " + s.Name + " " + formatExpr(s.Start) + " " + formatExpr(s.End))
		case "for-each":
			p.sb.WriteString(" " + s.Name + " " + formatExpr(s.Collection))
		}
		p.printBlock(s.Body, suffix)
	case *DBTransactionStmt:
		p.sb.WriteString("(db-transaction " + s.Conn)
		p.printBlock(s.Body, suffix)
	case *MatchStmt:
		p.sb.WriteString("(match " + formatExpr(s.Value) + "\n")
		p.indent++
		p.writeIndent()
		p.sb.WriteString("(ok " + s.OKName)
		p.printBlock(s.OK, "")
		p.sb.WriteString("\n")
		p.writeIndent()
		p.sb.WriteString("(err " + s.ErrName)
		p.printBlock(s.Err, ")"+suffix)
		p.indent--
	case *LetStmt:
		p.sb.WriteString(fmt.Sprintf("(let %s %s %s)%s", s.Name, s.Type.String(), formatExpr(s.Value), suffix))
	case *SetStmt:
		p.sb.WriteString(fmt.Sprintf("(set %s %s)%s", s.Name, formatExpr(s.Value), suffix))
	case *ReturnStmt:
		if s.Value != nil {
			p.sb.WriteString(fmt.Sprintf("(return %s)%s", formatExpr(s.Value), suffix))
		} else {
			p.sb.WriteString("(return)" + suffix)
		}
	case *IfStmt:
		p.printIf(s, suffix)
	case *SpawnStmt:
		p.sb.WriteString(fmt.Sprintf("(spawn %s)%s", formatExpr(s.Call), suffix))
	case *SendStmt:
		p.sb.WriteString(fmt.Sprintf("(send %s %s)%s", formatExpr(s.Channel), formatExpr(s.Value), suffix))
	case *ExprStmt:
		if call, ok := s.Expr.(*CallExpr); ok && !call.HasCall {
			p.sb.WriteString(formatExpr(s.Expr) + suffix)
		} else if _, ok := s.Expr.(*TryExpr); ok {
			p.sb.WriteString(formatExpr(s.Expr) + suffix)
		} else {
			p.sb.WriteString(fmt.Sprintf("(do %s)%s", formatExpr(s.Expr), suffix))
		}
	}
}

func (p *printer) printBlock(body []Stmt, suffix string) {
	if len(body) == 0 {
		p.sb.WriteString(")" + suffix)
		return
	}
	p.sb.WriteString("\n")
	p.indent++
	for i, s := range body {
		p.writeIndent()
		if i == len(body)-1 {
			p.printStmt(s, ")"+suffix)
		} else {
			p.printStmt(s, "")
			p.sb.WriteString("\n")
		}
	}
	p.indent--
}

func (p *printer) printIf(s *IfStmt, suffix string) {
	p.sb.WriteString(fmt.Sprintf("(if %s\n", formatExpr(s.Condition)))
	p.indent++

	// Then
	p.writeIndent()
	if len(s.Else) == 0 {
		if len(s.Then) == 0 {
			p.sb.WriteString("(then))" + suffix)
		} else {
			p.sb.WriteString("(then\n")
			p.indent++
			for i, stmt := range s.Then {
				p.writeIndent()
				if i == len(s.Then)-1 {
					p.printStmt(stmt, "))"+suffix)
				} else {
					p.printStmt(stmt, "")
					p.sb.WriteString("\n")
				}
			}
			p.indent--
		}
	} else {
		if len(s.Then) == 0 {
			p.sb.WriteString("(then)\n")
		} else {
			p.sb.WriteString("(then\n")
			p.indent++
			for i, stmt := range s.Then {
				p.writeIndent()
				if i == len(s.Then)-1 {
					p.printStmt(stmt, ")\n")
				} else {
					p.printStmt(stmt, "")
					p.sb.WriteString("\n")
				}
			}
			p.indent--
		}

		p.writeIndent()
		if len(s.Else) == 0 {
			p.sb.WriteString("(else))" + suffix)
		} else {
			p.sb.WriteString("(else\n")
			p.indent++
			for i, stmt := range s.Else {
				p.writeIndent()
				if i == len(s.Else)-1 {
					p.printStmt(stmt, "))"+suffix)
				} else {
					p.printStmt(stmt, "")
					p.sb.WriteString("\n")
				}
			}
			p.indent--
		}
	}

	p.indent--
}

func formatExpr(expr Expr) string {
	switch e := expr.(type) {
	case *IntLiteral:
		return e.Raw
	case *FloatLiteral:
		return e.Raw
	case *BoolLiteral:
		if e.Value {
			return "true"
		}
		return "false"
	case *StringLiteral:
		return fmt.Sprintf("%q", e.Value)
	case *IdentExpr:
		return e.Name
	case *CallExpr:
		var res string
		if e.HasCall {
			res = "(call " + e.Func
		} else {
			res = "(" + e.Func
		}
		for _, arg := range e.Args {
			res += " " + formatExpr(arg)
		}
		res += ")"
		return res
	case *TryExpr:
		return fmt.Sprintf("(try %s)", formatExpr(e.Expr))
	case *BinaryOpExpr:
		return fmt.Sprintf("(%s %s %s)", e.Op, formatExpr(e.Left), formatExpr(e.Right))
	case *RecvExpr:
		return fmt.Sprintf("(recv %s)", formatExpr(e.Channel))
	default:
		return ""
	}
}
