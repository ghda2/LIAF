package runtime

import (
	"io"
	"mime"
	"mime/multipart"
	"path/filepath"
	"strings"

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

func ResponseAddHeader(res web.Response, name, value string) web.Response {
	return res.AddHeader(name, value)
}

// RequestCookie le o valor do cookie especificado pelo nome a partir do header Cookie.
// Se ausente, devolve erro explicito.
func RequestCookie(req web.Request, name string) Result[string, string] {
	rawCookies := req.Header.Values("Cookie")
	for _, line := range rawCookies {
		for _, part := range strings.Split(line, ";") {
			part = strings.TrimSpace(part)
			if k, v, ok := strings.Cut(part, "="); ok {
				if k == name {
					return Ok[string, string](v)
				}
			}
		}
	}
	return Err[string, string]("cookie not found: " + name)
}

// RequestFileData extrai os dados binários do arquivo enviado na requisição HTTP.
// Suporta requisições multipart/form-data (buscando pelo campo especificado, ou pelo primeiro arquivo)
// bem como requisições com corpo binário direto (ex: uploads diretos tipo PUT/POST estilo S3).
func RequestFileData(req web.Request, fieldName string) Result[string, string] {
	ct := req.Header.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(ct)
	if err == nil && strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		if boundary == "" {
			return Err[string, string]("multipart request missing boundary")
		}
		mr := multipart.NewReader(strings.NewReader(req.Body), boundary)
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return Err[string, string]("failed parsing multipart body: " + err.Error())
			}
			if fieldName == "" || part.FormName() == fieldName {
				data, readErr := io.ReadAll(part)
				if readErr != nil {
					return Err[string, string]("failed reading multipart part: " + readErr.Error())
				}
				return Ok[string, string](string(data))
			}
		}
		return Err[string, string]("file field not found in multipart body: " + fieldName)
	}

	// Caso o corpo direto contenha dados da imagem/arquivo
	if len(req.Body) > 0 {
		return Ok[string, string](req.Body)
	}
	return Err[string, string]("empty request body and no multipart file found")
}

// RequestFileName extrai o nome do arquivo enviado na requisição (multipart, cabeçalho X-Filename ou query param).
func RequestFileName(req web.Request, fieldName string) Result[string, string] {
	ct := req.Header.Get("Content-Type")
	mediaType, params, err := mime.ParseMediaType(ct)
	if err == nil && strings.HasPrefix(mediaType, "multipart/") {
		boundary := params["boundary"]
		if boundary == "" {
			return Err[string, string]("multipart request missing boundary")
		}
		mr := multipart.NewReader(strings.NewReader(req.Body), boundary)
		for {
			part, err := mr.NextPart()
			if err == io.EOF {
				break
			}
			if err != nil {
				return Err[string, string]("failed parsing multipart body: " + err.Error())
			}
			if fieldName == "" || part.FormName() == fieldName {
				fn := part.FileName()
				if fn != "" {
					return Ok[string, string](filepath.Base(fn))
				}
				return Ok[string, string](part.FormName())
			}
		}
		return Err[string, string]("file field not found in multipart: " + fieldName)
	}

	if xfn := req.Header.Get("X-Filename"); xfn != "" {
		return Ok[string, string](filepath.Base(xfn))
	}
	if nameQuery := req.Query.Get("name"); nameQuery != "" {
		return Ok[string, string](filepath.Base(nameQuery))
	}
	return Ok[string, string]("upload")
}
