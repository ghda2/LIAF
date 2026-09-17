package codegen

import (
	"fmt"
	"liaf/pkg/ast"
	"strings"
)

func (g *Generator) libraryCall(c *ast.CallExpr) (string, bool) {
	n := strings.ReplaceAll(c.Func, "_", "-")
	args := []string{}
	for _, a := range c.Args {
		args = append(args, g.genExpr(a))
	}
	join := strings.Join(args, ", ")
	call := func(fn string) (string, bool) { return fn + "(" + join + ")", true }
	// Aceita tanto `Task` quanto construtores como `(list Task)`; ast.TypeFromExpr
	// e a mesma traducao que o checker usa, entao os dois nao divergem.
	typeName := func(i int) string {
		if t, ok := ast.TypeFromExpr(c.Args[i]); ok {
			return mapType(t)
		}
		return "interface{}"
	}
	switch n {
	case "make-list":
		return "rt.MakeList[" + typeName(0) + "]()", true
	case "make-map":
		return "make(map[" + typeName(0) + "]" + typeName(1) + ")", true
	case "list-len":
		return "int64(len((" + args[0] + ").Items))", true
	case "map-len", "str-len":
		return "int64(len(" + args[0] + "))", true
	case "list-push":
		return call("rt.ListPush")
	case "list-get":
		return call("rt.ListGet")
	case "list-set":
		return call("rt.ListSet")
	case "map-get":
		return call("rt.MapGet")
	case "map-set":
		return call("rt.MapSet")
	case "map-has":
		return call("rt.MapHas")
	case "ok", "err":
		t := g.info.Types[c].(*ast.AppliedType)
		fn := "Ok"
		if n == "err" {
			fn = "Err"
		}
		return fmt.Sprintf("rt.%s[%s,%s](%s)", fn, mapType(t.Args[0]), mapType(t.Args[1]), join), true
	case "fs-read-file":
		return call("rt.ReadFile")
	case "fs-write-file":
		return call("rt.WriteFile")
	case "fs-rename":
		return call("rt.Rename")
	case "fs-write-atomic":
		return call("rt.WriteAtomic")
	case "fs-remove":
		return call("rt.Remove")
	case "fs-exists":
		return call("rt.Exists")
	case "json-encode":
		return call("rt.JSONEncode")
	case "json-decode":
		return "rt.JSONDecode[" + typeName(1) + "](" + args[0] + ")", true
	case "int-from-str":
		return call("rt.IntFromStr")
	case "new":
		return typeName(0) + "{" + strings.Join(args[1:], ", ") + "}", true
	case "field":
		return "(" + args[0] + ")." + fieldIdent(c.Args[1].(*ast.IdentExpr).Name), true
	case "json-response":
		return call("web.JSONResponse")
	case "http-get", "http-post", "http-put", "http-delete":
		return fmt.Sprintf("web.Register(%q, %s)", strings.ToUpper(strings.TrimPrefix(n, "http-")), join), true
	case "serve-hybrid":
		return "rt.Failure(web.ServeHybrid(" + join + "))", true
	case "request-body":
		return "(" + args[0] + ").Body", true
	case "request-path":
		return "(" + args[0] + ").Path", true
	case "request-method":
		return "(" + args[0] + ").Method", true
	case "str-slice":
		return call("rt.StrSlice")
	case "str-eq":
		return "(" + args[0] + " == " + args[1] + ")", true
	case "args":
		return call("rt.Args")

	// --- Banco de dados (issue #015) ---
	// Em db-query e db-exec o SQL e os parametros seguem separados ate o
	// driver; nada aqui costura valor dentro do texto do comando.
	case "db-connect":
		return call("rt.DBConnect")
	case "db-close":
		return call("rt.DBClose")
	case "db-query":
		// (db-query conn SQL Tipo p...) -> rt.DBQuery[Tipo](conn, SQL, p...)
		params := append([]string{args[0], args[1]}, args[3:]...)
		return "rt.DBQuery[" + typeName(2) + "](" + strings.Join(params, ", ") + ")", true
	case "db-exec":
		return call("rt.DBExec")
	case "redis-get":
		return call("rt.RedisGet")
	case "redis-set":
		return call("rt.RedisSet")

	// --- WebSocket (issue #016) ---
	case "ws-send":
		return call("rt.WSSend")
	case "ws-send-json":
		return call("rt.WSSendJSON")
	case "ws-close":
		return call("rt.WSCloseConn")
	case "ws-broadcast":
		return call("rt.WSBroadcast")
	case "ws-join":
		return call("rt.WSJoin")
	case "ws-leave":
		return call("rt.WSLeave")
	case "ws-topic-size":
		return call("rt.WSTopicSize")
	}
	return "", false
}
