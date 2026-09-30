// Package docrt é o runtime de documentos dos programas LIAF (issue #034): liga os builtins
// pdf-* ao motor Typst embutido (pkg/typst). O código gerado só importa este pacote quando
// usa um desses builtins, para o motor não pesar em quem não gera documentos.
package docrt

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"sort"
	"strings"
	"time"

	rt "liaf/pkg/runtime"
	"liaf/pkg/typst"
	"liaf/pkg/typst/lib"
	"liaf/pkg/web"
)

// Timeout de uma renderização: protege o servidor de um documento que não termina.
var Timeout = 30 * time.Second

const dataPath = "/dados.json"

// Template renderiza um modelo embutido (ex.: "cv-moderno") com os dados em JSON e devolve
// os bytes do PDF.
func Template(name, dataJSON string) rt.Result[string, string] {
	if !lib.HasTemplate(name) {
		return rt.Err[string, string](fmt.Sprintf("modelo desconhecido: %q (disponíveis: %s)",
			name, strings.Join(lib.Templates(), ", ")))
	}
	return render(lib.TemplateMain(name, dataPath), dataJSON, nil, typst.FormatPDF, name)
}

// TemplateFiles é Template com arquivos extras (foto, logo, SVG, CSV), entregues ao documento
// nos caminhos das chaves: {"foto.jpg": bytes} fica em "/foto.jpg".
func TemplateFiles(name, dataJSON string, files map[string]string) rt.Result[string, string] {
	if !lib.HasTemplate(name) {
		return rt.Err[string, string](fmt.Sprintf("modelo desconhecido: %q (disponíveis: %s)",
			name, strings.Join(lib.Templates(), ", ")))
	}
	return render(lib.TemplateMain(name, dataPath), dataJSON, files, typst.FormatPDF, name)
}

// SourceFiles é Source com arquivos extras.
func SourceFiles(src, dataJSON string, files map[string]string) rt.Result[string, string] {
	return render([]byte(src), dataJSON, files, typst.FormatPDF, "")
}

// Source renderiza um documento Typst escrito pelo programa. Os dados ficam em /dados.json
// e a biblioteca da LIAF em /liaf/.
func Source(src, dataJSON string) rt.Result[string, string] {
	return render([]byte(src), dataJSON, nil, typst.FormatPDF, "")
}

// TemplatePNG gera a prévia da primeira página de um modelo, em PNG (144 ppi).
func TemplatePNG(name, dataJSON string) rt.Result[string, string] {
	if !lib.HasTemplate(name) {
		return rt.Err[string, string](fmt.Sprintf("modelo desconhecido: %q", name))
	}
	return render(lib.TemplateMain(name, dataPath), dataJSON, nil, typst.FormatPNG, name)
}

// Response devolve o PDF como resposta HTTP, aberto no navegador com o nome sugerido.
func Response(pdf, filename string) web.Response {
	name := strings.NewReplacer(`"`, "", "\r", "", "\n", "").Replace(filename)
	return web.Response{
		Status:      200,
		ContentType: "application/pdf",
		Body:        pdf,
		Headers: []web.ResponseHeader{
			{Name: "Content-Disposition", Value: `inline; filename="` + name + `"`},
			{Name: "X-Content-Type-Options", Value: "nosniff"},
			{Name: "Cache-Control", Value: "private, no-store"},
		},
	}
}

// MaxExtraBytes limita a soma dos arquivos extras de um documento.
var MaxExtraBytes = 64 << 20

// extraPath valida o nome de um arquivo extra e devolve o caminho virtual. Os caminhos
// reservados (/main.typ, /dados.json e a biblioteca em /liaf/) não podem ser sobrescritos.
func extraPath(name string) (string, error) {
	if name == "" || strings.ContainsAny(name, "\\\x00") {
		return "", fmt.Errorf("nome de arquivo inválido: %q", name)
	}
	p := "/" + strings.TrimPrefix(name, "/")
	for _, part := range strings.Split(p[1:], "/") {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("nome de arquivo inválido: %q (sem \"..\" nem partes vazias)", name)
		}
	}
	if p == "/main.typ" || p == dataPath || strings.HasPrefix(p, lib.Mount) {
		return "", fmt.Errorf("nome de arquivo reservado: %q", name)
	}
	return p, nil
}

func render(main []byte, dataJSON string, extra map[string]string, format typst.Format, ident string) rt.Result[string, string] {
	eng, err := typst.Default()
	if err != nil {
		return rt.Err[string, string]("motor de documentos indisponível: " + err.Error())
	}
	if dataJSON == "" {
		dataJSON = "{}"
	}
	files := lib.Files()
	files["/main.typ"] = main
	files[dataPath] = []byte(dataJSON)
	addPackages(files)

	// O identificador do PDF deriva do conteúdo: a mesma entrada gera os mesmos bytes.
	h := sha256.New()
	h.Write([]byte(ident + "\x00"))
	h.Write(main)
	h.Write([]byte(dataJSON))

	names := make([]string, 0, len(extra))
	for name := range extra {
		names = append(names, name)
	}
	sort.Strings(names)
	total := 0
	for _, name := range names {
		p, err := extraPath(name)
		if err != nil {
			return rt.Err[string, string](err.Error())
		}
		data := extra[name]
		total += len(data)
		if total > MaxExtraBytes {
			return rt.Err[string, string](fmt.Sprintf("arquivos extras passam de %d MB", MaxExtraBytes>>20))
		}
		files[p] = []byte(data)
		h.Write([]byte("\x00" + p + "\x00" + data))
	}
	sum := h.Sum(nil)

	ctx, cancel := context.WithTimeout(context.Background(), Timeout)
	defer cancel()
	res, err := eng.Render(ctx, typst.Job{
		Files:  files,
		Format: format,
		Ident:  hex.EncodeToString(sum[:16]),
	})
	if err != nil {
		return rt.Err[string, string](err.Error())
	}
	if err := res.Err(); err != nil {
		return rt.Err[string, string](err.Error())
	}
	if len(res.Outputs) == 0 {
		return rt.Err[string, string]("documento sem páginas")
	}
	return rt.Ok[string, string](string(res.Outputs[0]))
}
