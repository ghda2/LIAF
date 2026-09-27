package runtime

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode/utf8"

	"liaf/pkg/web"
)

// Cliente HTTP do builtin http-fetch. Segue o fetch do JavaScript: (err ...)
// so quando nao houve resposta (DNS, conexao, TLS, timeout, URL invalida).
// Um 404 ou 500 e uma resposta, e chega como (ok ...) com o status; muita
// API manda o motivo do erro no corpo, e o programa precisa le-lo.

// HttpReply e o valor por tras do tipo opaco HttpReply da LIAF.
type HttpReply struct {
	status int
	body   string
	header http.Header
}

const (
	// Limite do corpo lido. Uma resposta maior e erro, nao truncamento: um
	// JSON cortado no meio falharia mais adiante, longe da causa.
	fetchMaxBody = 10 << 20
	userAgent    = "liaf-http-fetch"
)

// fetchTimeout cobre a requisicao inteira, da conexao ao fim do corpo. O
// http.Client padrao nao tem timeout: um servidor que nunca responde
// prenderia a rota que o chamou para sempre. E variavel so para os testes
// nao esperarem 30 segundos.
var fetchTimeout = 30 * time.Second

// FetchMethods e a lista que o checker confere quando o metodo e literal.
var FetchMethods = []string{"GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"}

func HTTPFetch(method, rawURL string, headers *List[string], body string) Result[*HttpReply, string] {
	fail := func(format string, args ...any) Result[*HttpReply, string] {
		return Err[*HttpReply, string]("http-fetch: " + fmt.Sprintf(format, args...))
	}

	method = strings.ToUpper(method)
	known := false
	for _, m := range FetchMethods {
		known = known || m == method
	}
	if !known {
		return fail("unknown method %q", method)
	}

	u, err := url.Parse(rawURL)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return fail("invalid URL %q: expected http:// or https:// with a host", rawURL)
	}

	var reader io.Reader
	if body != "" {
		reader = strings.NewReader(body)
	}
	req, err := http.NewRequest(method, u.String(), reader)
	if err != nil {
		return fail("%v", err)
	}
	req.Header.Set("User-Agent", userAgent)

	if headers != nil {
		for _, line := range headers.Items {
			name, value, ok := strings.Cut(line, ":")
			name, value = strings.TrimSpace(name), strings.TrimSpace(value)
			if !ok || !web.ValidHeaderName(name) {
				return fail("invalid header %q: expected \"Name: value\"", line)
			}
			if strings.ContainsAny(value, "\r\n\x00") {
				return fail("header %s has a line break or NUL in its value", name)
			}
			req.Header.Add(name, value)
		}
	}
	// Corpo sem Content-Type declarado e quase sempre JSON numa API; quem
	// manda outra coisa declara o header.
	if body != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", "application/json")
	}

	// Um Client por chamada e barato: o Transport padrao, com o pool de
	// conexoes, continua compartilhado.
	res, err := (&http.Client{Timeout: fetchTimeout}).Do(req)
	if err != nil {
		// A mensagem do net/http traz a URL inteira, e muita API leva o
		// token na query string; o erro vai para log. Fica so metodo e host.
		cause := err
		var urlErr *url.Error
		if errors.As(err, &urlErr) {
			if urlErr.Timeout() {
				return fail("%s %s: timeout after %s", method, u.Host, fetchTimeout)
			}
			cause = urlErr.Err
		}
		return fail("%s %s: %v", method, u.Host, cause)
	}
	defer res.Body.Close()

	data, err := io.ReadAll(io.LimitReader(res.Body, fetchMaxBody+1))
	if err != nil {
		return fail("reading the response body: %v", err)
	}
	if len(data) > fetchMaxBody {
		return fail("response body larger than %d MiB", fetchMaxBody>>20)
	}
	// str na LIAF e texto; um corpo binario (imagem, PDF) nao cabe nele.
	if !utf8.Valid(data) {
		return fail("response body is not UTF-8 text (Content-Type %q)", res.Header.Get("Content-Type"))
	}
	return Ok[*HttpReply, string](&HttpReply{status: res.StatusCode, body: string(data), header: res.Header})
}

func ReplyStatus(r *HttpReply) int64 { return int64(r.status) }

func ReplyBody(r *HttpReply) string { return r.body }

func ReplyHeader(r *HttpReply, name string) Result[string, string] {
	values := r.header.Values(name)
	if len(values) == 0 {
		return Err[string, string]("response header not found: " + name)
	}
	return Ok[string, string](values[0])
}
