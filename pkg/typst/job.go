package typst

import (
	"fmt"
	"strings"
)

// Format é o tipo de saída de uma renderização.
type Format string

const (
	FormatPDF   Format = "pdf"
	FormatPNG   Format = "png"   // um PNG por página
	FormatCheck Format = "check" // só compila: diagnósticos e tamanho das páginas
)

// Job descreve um documento a renderizar. Todos os arquivos que o documento lê (fonte Typst,
// dados, imagens) vão em Files: o motor não enxerga o disco nem a rede.
type Job struct {
	Main      string            // arquivo principal; padrão "/main.typ"
	Files     map[string][]byte // caminho virtual ("/dados.json") -> conteúdo
	Format    Format            // padrão FormatPDF
	PPI       float64           // resolução do PNG; padrão 144
	Standards []string          // padrões PDF exigidos, ex.: "a-2b", "ua-1"
	Ident     string            // identificador estável: deixa o /ID do PDF determinístico
	Today     string            // data de datetime.today(), "AAAA-MM-DD"; vazio = indisponível
	Untagged  bool              // desliga o PDF marcado (acessível), que é o padrão
	Layout    bool              // inclui o relatório de layout (Result.Layout)
}

// Diagnostic é um erro ou aviso do Typst, com a posição no arquivo virtual.
type Diagnostic struct {
	Code     string   `json:"code,omitempty"` // W_LAYOUT_*, preenchido pela análise de qualidade
	Severity string   `json:"severity"`       // "error" ou "warning"
	Message  string   `json:"message"`
	File     string   `json:"file,omitempty"`
	Line     int      `json:"line,omitempty"`
	Column   int      `json:"column,omitempty"`
	Hints    []string `json:"hints,omitempty"`
	Page     int      `json:"page,omitempty"` // página (1..n) dos avisos de layout
	Fix      string   `json:"fix,omitempty"`  // correção sugerida, pronta para aplicar
}

func (d Diagnostic) String() string {
	var b strings.Builder
	if d.File != "" {
		fmt.Fprintf(&b, "%s:%d:%d: ", d.File, d.Line, d.Column)
	}
	b.WriteString(d.Severity + ": " + d.Message)
	for _, h := range d.Hints {
		b.WriteString("\n  dica: " + h)
	}
	return b.String()
}

// Page traz o tamanho de uma página em pontos tipográficos (1 pt = 1/72 pol).
type Page struct {
	WidthPt  float64 `json:"width_pt"`
	HeightPt float64 `json:"height_pt"`
}

// Result é o que uma renderização produziu.
type Result struct {
	OK          bool         `json:"ok"`
	Diagnostics []Diagnostic `json:"diagnostics"`
	Pages       []Page       `json:"pages"`
	Layout      []PageLayout `json:"layout,omitempty"` // só com Job.Layout
	Outputs     [][]byte     `json:"-"`                // PDF: um item; PNG: um por página; check: nenhum
}

// Err devolve os erros do documento como um único error, ou nil se compilou.
func (r Result) Err() error {
	if r.OK {
		return nil
	}
	var msgs []string
	for _, d := range r.Diagnostics {
		if d.Severity == "error" {
			msgs = append(msgs, d.String())
		}
	}
	if len(msgs) == 0 {
		return fmt.Errorf("typst: falha sem diagnóstico")
	}
	return fmt.Errorf("typst: %s", strings.Join(msgs, "\n"))
}

// Warnings devolve só os avisos.
func (r Result) Warnings() []Diagnostic {
	var out []Diagnostic
	for _, d := range r.Diagnostics {
		if d.Severity == "warning" {
			out = append(out, d)
		}
	}
	return out
}

type request struct {
	Main      string   `json:"main"`
	Format    Format   `json:"format,omitempty"`
	PPI       float64  `json:"ppi,omitempty"`
	Standards []string `json:"standards,omitempty"`
	Ident     string   `json:"ident,omitempty"`
	Today     string   `json:"today,omitempty"`
	Untagged  bool     `json:"untagged,omitempty"`
	Layout    bool     `json:"layout,omitempty"`
}

func (j Job) request() request {
	main := j.Main
	if main == "" {
		main = "/main.typ"
	}
	return request{
		Main: main, Format: j.Format, PPI: j.PPI, Standards: j.Standards,
		Ident: j.Ident, Today: j.Today, Untagged: j.Untagged, Layout: j.Layout,
	}
}
