// Package c emits standalone C99 source. Unsupported operations are errors.
package c

import (
	_ "embed"
	"fmt"
	"liaf/pkg/ast"
	"liaf/pkg/builtins"
	"liaf/pkg/checker"
	"strconv"
	"strings"
)

//go:embed runtime.h
var runtimeSource string

type generator struct {
	info   *checker.Info
	out    strings.Builder
	serial int
	err    error
}

func ident(name string) string { return fmt.Sprintf("u_%x", []byte(name)) }
func quote(s string) string {
	var b strings.Builder
	b.WriteByte('"')
	for _, c := range []byte(s) {
		switch c {
		case '\\', '"':
			b.WriteByte('\\')
			b.WriteByte(c)
		default:
			if c < 32 || c >= 127 {
				fmt.Fprintf(&b, "\\%03o", c)
			} else {
				b.WriteByte(c)
			}
		}
	}
	b.WriteByte('"')
	return b.String()
}
func (g *generator) emit(format string, args ...any) { fmt.Fprintf(&g.out, format+"\n", args...) }
func (g *generator) temp() string                    { g.serial++; return fmt.Sprintf("tmp_%d", g.serial) }
func (g *generator) value(code string) string {
	id := g.temp()
	g.emit("V %s = %s;", id, code)
	return id
}

func Generate(mod *ast.Module) (string, error) {
	info, diags := checker.Check(mod, "")
	if len(diags) > 0 {
		return "", fmt.Errorf("semantic check: %s", diags[0].String())
	}
	g := &generator{info: info}
	for _, d := range mod.Decls {
		if _, ok := d.(*ast.WSRouteDecl); ok {
			// Falhar aqui em vez de ignorar: um binario C sem as rotas de
			// WebSocket compilaria e subiria calado, sem o socket que o
			// programa declara.
			return "", fmt.Errorf("C backend: ws-route is not implemented yet")
		}
	}
	g.out.WriteString(runtimeSource)
	for _, d := range mod.Decls {
		if f, ok := d.(*ast.FuncDecl); ok {
			g.emit("static V %s(V *argv);", ident(f.Name))
		}
	}
	for _, d := range mod.Decls {
		if f, ok := d.(*ast.FuncDecl); ok {
			g.emit("static V %s(V *argv) {", ident(f.Name))
			for i, p := range f.Params {
				g.emit("V %s=argv[%d];", ident(p.Name), i)
			}
			g.body(f.Body)
			g.emit("return vnone();\n}")
		}
	}
	g.emit("int main(int argc,char **argv){int i;program_args=vlist();for(i=1;i<argc;i++)vpush(program_args,vs(argv[i]));%s(NULL);return 0;}", ident("main"))
	if g.err != nil {
		return "", g.err
	}
	return g.out.String(), nil
}
func array(args []string) string {
	if len(args) == 0 {
		return "NULL"
	}
	return "(V[]){" + strings.Join(args, ",") + "}"
}
func (g *generator) body(body []ast.Stmt) {
	for _, s := range body {
		g.stmt(s)
	}
}
func (g *generator) stmt(st ast.Stmt) {
	switch s := st.(type) {
	case *ast.LetStmt:
		v := g.expr(s.Value)
		g.emit("V %s=%s;", ident(s.Name), v)
	case *ast.SetStmt:
		v := g.expr(s.Value)
		g.emit("%s=%s;", ident(s.Name), v)
	case *ast.ReturnStmt:
		if s.Value == nil {
			g.emit("return vnone();")
		} else {
			v := g.expr(s.Value)
			g.emit("return %s;", v)
		}
	case *ast.ExprStmt:
		g.expr(s.Expr)
	case *ast.IfStmt:
		v := g.expr(s.Condition)
		g.emit("if(%s.i){", v)
		g.body(s.Then)
		g.emit("}else{")
		g.body(s.Else)
		g.emit("}")
	case *ast.LoopControl:
		g.emit("%s;", s.Kind)
	case *ast.LoopStmt:
		g.emit("{")
		switch s.Kind {
		case "while":
			g.emit("for(;;){")
			v := g.expr(s.Condition)
			g.emit("if(!%s.i)break;", v)
		case "for-range":
			a, b := g.expr(s.Start), g.expr(s.End)
			g.emit("V %s=%s;for(;%s.i<%s.i;%s.i++){", ident(s.Name), a, ident(s.Name), b, ident(s.Name))
		case "for-each":
			v := g.expr(s.Collection)
			i, n := g.temp(), g.temp()
			g.emit("size_t %s=0,%s=((List*)%s.p)->n;for(;%s<%s;%s++){V %s=((List*)%s.p)->a[%s];", i, n, v, i, n, i, ident(s.Name), v, i)
		}
		g.body(s.Body)
		g.emit("}\n}")
	case *ast.MatchStmt:
		v := g.expr(s.Value)
		if s.IsOption {
			g.emit("if(((Option*)%s.p)->some){V %s=((Option*)%s.p)->value;", v, ident(s.OKName))
			g.body(s.OK)
			g.emit("}else{")
			g.body(s.Err)
			g.emit("}")
			return
		}
		g.emit("if(((Result*)%s.p)->ok){V %s=((Result*)%s.p)->value;", v, ident(s.OKName), v)
		g.body(s.OK)
		g.emit("}else{V %s=((Result*)%s.p)->value;", ident(s.ErrName), v)
		g.body(s.Err)
		g.emit("}")
	case *ast.SendStmt:
		a, b := g.expr(s.Channel), g.expr(s.Value)
		g.emit("vsend(%s,%s);", a, b)
	case *ast.SpawnStmt:
		if g.info.Functions[s.Call.Func] == nil {
			g.err = fmt.Errorf("C backend: spawn requires user-defined function")
			return
		}
		args := []string{}
		for _, a := range s.Call.Args {
			args = append(args, g.expr(a))
		}
		g.emit("vspawn(%s,%d,%s);", ident(s.Call.Func), len(args), array(args))
	default:
		g.err = fmt.Errorf("C backend: unsupported statement %T", st)
	}
}
func (g *generator) expr(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.IntLiteral:
		if v.Value == -9223372036854775808 {
			return g.value("vi(INT64_MIN)")
		}
		return g.value("vi(" + strconv.FormatInt(v.Value, 10) + "LL)")
	case *ast.FloatLiteral:
		return g.value("vf(" + strconv.FormatFloat(v.Value, 'g', -1, 64) + ")")
	case *ast.BoolLiteral:
		if v.Value {
			return g.value("vb(1)")
		}
		return g.value("vb(0)")
	case *ast.StringLiteral:
		return g.value(fmt.Sprintf("vsn(%s,%d)", quote(v.Value), len(v.Value)))
	case *ast.IdentExpr:
		return g.value(ident(v.Name))
	case *ast.RecvExpr:
		ch := g.expr(v.Channel)
		return g.value("vrecv(" + ch + ")")
	case *ast.CallExpr:
		return g.call(v)
	case *ast.BinaryOpExpr:
		left := g.expr(v.Left)
		if v.Op == "and" || v.Op == "or" {
			out := g.value(left)
			condition := out + ".i"
			if v.Op == "or" {
				condition = "!" + condition
			}
			g.emit("if(%s){", condition)
			r := g.expr(v.Right)
			g.emit("%s=%s;}", out, r)
			return out
		}
		right := g.expr(v.Right)
		if v.Op == "eq" {
			return g.value("vb(eq(" + left + "," + right + "))")
		}
		if v.Op == "neq" {
			return g.value("vb(!eq(" + left + "," + right + "))")
		}
		if v.Op == "div" {
			return g.value("vdiv(" + left + "," + right + ")")
		}
		op := map[string]string{"add": "+", "sub": "-", "mul": "*", "gt": ">", "lt": "<", "gte": ">=", "lte": "<="}[v.Op]
		field, wrap := "i", "vi"
		if t := g.info.Types[v.Left]; t != nil {
			if t.String() == "float" {
				field, wrap = "f", "vf"
			} else if t.String() == "str" {
				field, wrap = "s", "vb"
			}
		}
		if v.Op == "gt" || v.Op == "lt" || v.Op == "gte" || v.Op == "lte" {
			wrap = "vb"
			if field == "s" {
				return g.value(fmt.Sprintf("vb(strcmp(%s.s, %s.s) %s 0)", left, right, op))
			}
		}
		if field == "i" && wrap == "vi" {
			switch v.Op {
			case "add":
				return g.value(fmt.Sprintf("vadd(%s, %s)", left, right))
			case "sub":
				return g.value(fmt.Sprintf("vsub(%s, %s)", left, right))
			case "mul":
				return g.value(fmt.Sprintf("vmul(%s, %s)", left, right))
			}
			return g.value(fmt.Sprintf("vi((int64_t)((uint64_t)%s.i %s (uint64_t)%s.i))", left, op, right))
		}
		return g.value(fmt.Sprintf("%s(%s.%s %s %s.%s)", wrap, left, field, op, right, field))
	case *ast.IfExpr:
		cond := g.expr(v.Condition)
		out := g.value("vnone()")
		g.emit("if(%s.i){", cond)
		thenVal := g.expr(v.Then)
		g.emit("%s=%s;}else{", out, thenVal)
		if v.Else != nil {
			elseVal := g.expr(v.Else)
			g.emit("%s=%s;}", out, elseVal)
		} else {
			g.emit("}")
		}
		return out
	case *ast.MatchExpr:
		val := g.expr(v.Value)
		out := g.value("vnone()")
		if v.IsOption {
			g.emit("if(%s.tag == LIAF_OPTION && ((Option*)%s.p)->some){", val, val)
			if v.OKName != "" && v.OKName != "_" {
				g.emit("V %s = ((Option*)%s.p)->value;", ident(v.OKName), val)
			}
			okVal := g.expr(v.OK)
			g.emit("%s = %s;}else{", out, okVal)
			if v.ErrName != "" && v.ErrName != "_" {
				g.emit("V %s = vnone();", ident(v.ErrName))
			}
			errVal := g.expr(v.Err)
			g.emit("%s = %s;}", out, errVal)
			return out
		}
		g.emit("if(%s.tag == LIAF_RESULT && ((Result*)%s.p)->ok){", val, val)
		if v.OKName != "" && v.OKName != "_" {
			g.emit("V %s = ((Result*)%s.p)->value;", ident(v.OKName), val)
		}
		okVal := g.expr(v.OK)
		g.emit("%s = %s;}else{", out, okVal)
		if v.ErrName != "" && v.ErrName != "_" {
			g.emit("V %s = vs(\"\");", ident(v.ErrName))
		}
		errVal := g.expr(v.Err)
		g.emit("%s = %s;}", out, errVal)
		return out
	}
	g.err = fmt.Errorf("C backend: unsupported expression %T", e)
	return g.value("vnone()")
}
func (g *generator) Expr(e ast.Expr) string              { return g.expr(e) }
func (g *generator) Value(code string) string             { return g.value(code) }
func (g *generator) Quote(s string) string                { return quote(s) }
func (g *generator) Array(items []string) string          { return array(items) }
func (g *generator) FindStruct(name string) *ast.StructDecl { return g.info.Structs[name] }
func (g *generator) SetError(err error)                   { g.err = err }

func (g *generator) call(c *ast.CallExpr) string {
	n := strings.ReplaceAll(c.Func, "_", "-")
	b := builtins.Lookup(n)
	if b != nil {
		if b.UnsupportedC {
			reason := b.UnsupportedCReason
			if reason == "" {
				reason = fmt.Sprintf("unsupported operation %s", c.Func)
			}
			g.err = fmt.Errorf("C backend: %s", reason)
			return g.value("vnone()")
		}
		var args []string
		if !b.UnevaluatedArgs {
			args = make([]string, len(c.Args))
			for i, a := range c.Args {
				args[i] = g.expr(a)
			}
		}
		if b.CEmit != nil {
			if res, ok := b.CEmit(g, c, args); ok {
				return res
			}
		}
		if b.CTemplate != "" {
			anyArgs := make([]any, len(args))
			for i, v := range args {
				anyArgs[i] = v
			}
			return g.value(fmt.Sprintf(b.CTemplate, anyArgs...))
		}
		if b.CCall != "" {
			return g.value(b.CCall + "(" + strings.Join(args, ",") + ")")
		}
	}
	if g.info.Functions[c.Func] != nil {
		args := make([]string, len(c.Args))
		for i, a := range c.Args {
			args[i] = g.expr(a)
		}
		return g.value(ident(c.Func) + "(" + array(args) + ")")
	}
	g.err = fmt.Errorf("C backend: unsupported operation %s", c.Func)
	return g.value("vnone()")
}
