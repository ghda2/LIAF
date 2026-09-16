// Package c emits standalone C99 source. Unsupported operations are errors.
package c

import (
	_ "embed"
	"fmt"
	"liaf/pkg/ast"
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
		if t := g.info.Types[v.Left]; t != nil && t.String() == "float" {
			field, wrap = "f", "vf"
		}
		if v.Op == "gt" || v.Op == "lt" || v.Op == "gte" || v.Op == "lte" {
			wrap = "vb"
		}
		if field == "i" && wrap == "vi" {
			return g.value(fmt.Sprintf("vi((int64_t)((uint64_t)%s.i %s (uint64_t)%s.i))", left, op, right))
		}
		return g.value(fmt.Sprintf("%s(%s.%s %s %s.%s)", wrap, left, field, op, right, field))
	}
	g.err = fmt.Errorf("C backend: unsupported expression %T", e)
	return g.value("vnone()")
}
func (g *generator) call(c *ast.CallExpr) string {
	n := strings.ReplaceAll(c.Func, "_", "-")
	switch n {
	case "make-list":
		return g.value("vlist()")
	case "make-map":
		return g.value("vmap()")
	case "make-chan":
		return g.value("vchan()")
	case "field":
		v := g.expr(c.Args[0])
		return g.value("vfield(" + v + "," + quote(c.Args[1].(*ast.IdentExpr).Name) + ")")
	case "new":
		s := g.info.Structs[c.Args[0].(*ast.IdentExpr).Name]
		names, args := []string{}, []string{}
		for i, a := range c.Args[1:] {
			args = append(args, g.expr(a))
			names = append(names, quote(s.Fields[i].Name))
		}
		nms := "NULL"
		if len(names) > 0 {
			nms = "(const char*[]){" + strings.Join(names, ",") + "}"
		}
		return g.value(fmt.Sprintf("vobject(%d,%s,%s)", len(args), nms, array(args)))
	case "json-decode":
		g.err = fmt.Errorf("C backend: json-decode is not implemented yet")
		return g.value("vnone()")
	}
	args := []string{}
	for _, a := range c.Args {
		args = append(args, g.expr(a))
	}
	join := strings.Join(args, ",")
	call := func(fn string) string { return g.value(fn + "(" + join + ")") }
	switch n {
	case "print", "println":
		nl := 0
		if n == "println" {
			nl = 1
		}
		return g.value(fmt.Sprintf("vprint(%d,%s,%d)", len(args), array(args), nl))
	case "concat":
		return g.value(fmt.Sprintf("vconcat(%d,%s)", len(args), array(args)))
	case "str-from-int":
		return call("vintstr")
	case "float-from-int":
		return g.value("vf((double)" + args[0] + ".i)")
	case "int-from-str":
		return call("vintparse")
	case "sleep-ms":
		return call("vsleep")
	case "list-push":
		return call("vpush")
	case "list-get":
		return call("vget")
	case "list-set":
		return call("vset")
	case "list-len":
		return g.value("vi(((List*)" + args[0] + ".p)->n)")
	case "map-len":
		return g.value("vi(((Map*)" + args[0] + ".p)->n)")
	case "map-set":
		return call("vmset")
	case "map-get":
		return g.value("vmget(" + join + ",0)")
	case "map-has":
		return g.value("vmget(" + join + ",1)")
	case "ok":
		return g.value("vr(1," + join + ")")
	case "err":
		return g.value("vr(0," + join + ")")
	case "fs-read-file":
		return call("vread")
	case "fs-write-file":
		return call("vwrite")
	case "fs-remove":
		return call("vremove")
	case "fs-exists":
		return call("vexists")
	case "str-len":
		return g.value("vi(" + args[0] + ".len)")
	case "str-eq":
		return g.value("vb(eq(" + join + "))")
	case "str-slice":
		return call("vslice")
	case "args":
		return g.value("program_args")
	}
	if g.info.Functions[c.Func] != nil {
		return g.value(ident(c.Func) + "(" + array(args) + ")")
	}
	g.err = fmt.Errorf("C backend: unsupported operation %s", c.Func)
	return g.value("vnone()")
}
