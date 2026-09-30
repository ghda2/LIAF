// Package docrt é o runtime de documentos dos programas LIAF (issue #034): liga os builtins
// pdf-* e png-* ao motor Typst embutido (pkg/typst). O código gerado só importa este pacote
// quando usa um desses builtins, para o motor não pesar em quem não gera documentos.
//
// Não há modelos prontos: o documento é código Typst escrito pelo programa, que recebe os dados
// em /dados.json e os arquivos extras (imagens, SVG, CSV) nos caminhos pedidos.
package docrt

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	rt "liaf/pkg/runtime"
	"liaf/pkg/typst"
	"liaf/pkg/web"
)

// Aquecimento: compilar o motor leva de ~0,5 s (cache do wazero) a vários segundos (primeira
// execução numa máquina). O programa começa a carregar o motor assim que sobe, em segundo
// plano, para a primeira requisição de PDF não pagar essa espera. LIAF_DOC_WARMUP=0 desliga.
func init() {
	if os.Getenv("LIAF_DOC_WARMUP") != "0" {
		go func() { _, _ = typst.Default() }()
	}
}

// Timeout de uma renderização: protege o servidor de um documento que não termina.
var Timeout = 30 * time.Second

// MaxExtraBytes limita a soma dos arquivos extras de um documento.
var MaxExtraBytes = 64 << 20

const (
	mainPath = "/main.typ"
	dataPath = "/dados.json"
)

// Source renderiza em PDF um documento Typst. Os dados (JSON) ficam em /dados.json.
func Source(src, dataJSON string) rt.Result[string, string] {
	return render(src, dataJSON, nil, typst.FormatPDF)
}

// SourceFiles é Source com arquivos extras: {"img/logo.png": bytes} fica em "/img/logo.png".
func SourceFiles(src, dataJSON string, files map[string]string) rt.Result[string, string] {
	return render(src, dataJSON, files, typst.FormatPDF)
}

// SourcePNG gera a prévia da primeira página em PNG (144 ppi), para conferir o visual.
func SourcePNG(src, dataJSON string) rt.Result[string, string] {
	return render(src, dataJSON, nil, typst.FormatPNG)
}

// SourcePNGFiles é SourcePNG com arquivos extras.
func SourcePNGFiles(src, dataJSON string, files map[string]string) rt.Result[string, string] {
	return render(src, dataJSON, files, typst.FormatPNG)
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

// extraPath valida o nome de um arquivo extra e devolve o caminho virtual. /main.typ e
// /dados.json são reservados.
func extraPath(name string) (string, error) {
	if name == "" || strings.ContainsAny(name, "\\\x00") || strings.HasPrefix(name, "@") {
		return "", fmt.Errorf("nome de arquivo inválido: %q", name)
	}
	p := "/" + strings.TrimPrefix(name, "/")
	for part := range strings.SplitSeq(p[1:], "/") {
		if part == "" || part == "." || part == ".." {
			return "", fmt.Errorf("nome de arquivo inválido: %q (sem \"..\" nem partes vazias)", name)
		}
	}
	if p == mainPath || p == dataPath {
		return "", fmt.Errorf("nome de arquivo reservado: %q", name)
	}
	return p, nil
}

func render(src, dataJSON string, extra map[string]string, format typst.Format) rt.Result[string, string] {
	eng, err := typst.Default()
	if err != nil {
		return rt.Err[string, string]("motor de documentos indisponível: " + err.Error())
	}
	if dataJSON == "" {
		dataJSON = "{}"
	}
	files := map[string][]byte{mainPath: []byte(src), dataPath: []byte(dataJSON)}
	addPackages(files)

	// O identificador do PDF deriva do conteúdo: a mesma entrada gera os mesmos bytes.
	h := sha256.New()
	h.Write([]byte(src))
	h.Write([]byte{0})
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
