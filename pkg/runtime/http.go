package runtime

import (
	"liaf/pkg/web"
)

// Adaptadores dos builtins request-header, request-query e
// response-set-header. Seguem a convencao das outras buscas da linguagem
// (env-get, map-get): ausencia e (err ...), nao string vazia, porque um
// header presente e vazio e um header ausente sao casos diferentes para
// quem autentica.

// RequestHeader le o primeiro valor do header, sem diferenciar maiusculas
// (Authorization e authorization sao o mesmo header em HTTP).
func RequestHeader(req web.Request, name string) Result[string, string] {
	values := req.Header.Values(name)
	if len(values) == 0 {
		return Err[string, string]("request header not found: " + name)
	}
	return Ok[string, string](values[0])
}

// RequestQuery le o primeiro valor do parametro da query string. O nome
// diferencia maiusculas, como a propria URL.
func RequestQuery(req web.Request, name string) Result[string, string] {
	values, ok := req.Query[name]
	if !ok || len(values) == 0 {
		return Err[string, string]("query parameter not found: " + name)
	}
	return Ok[string, string](values[0])
}

func ResponseSetHeader(res web.Response, name, value string) web.Response {
	return res.WithHeader(name, value)
}
