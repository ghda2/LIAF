package typst

import (
	"fmt"
	"math"
	"strings"
)

// PageLayout é o relatório de uma página (Job.Layout): onde o conteúdo caiu, em pontos.
type PageLayout struct {
	WidthPt  float64     `json:"width_pt"`
	HeightPt float64     `json:"height_pt"`
	Content  *[4]float64 `json:"content"` // [x0, y0, x1, y1]; nil = página vazia
	Texts    int         `json:"texts"`
	Images   int         `json:"images"`
	Shapes   int         `json:"shapes"`
	Overflow []Overflow  `json:"overflow"`
	// Textos com contraste abaixo do WCAG AA contra o fundo desenhado atrás deles.
	LowContrast []LowContrast `json:"low_contrast"`
}

// LowContrast é um texto difícil de ler: cor e fundo em hexadecimal, razão de contraste medida
// e o mínimo exigido (4,5 para texto normal, 3 para texto grande).
type LowContrast struct {
	Text       string  `json:"text"`
	Ratio      float64 `json:"ratio"`
	Minimum    float64 `json:"minimum"`
	Color      string  `json:"color"`
	Background string  `json:"background"`
	SizePt     float64 `json:"size_pt"`
}

// Overflow é um elemento que passa da borda da página.
type Overflow struct {
	Kind string     `json:"kind"` // texto, imagem, forma, grupo
	Text string     `json:"text,omitempty"`
	BBox [4]float64 `json:"bbox"`
	ByPt float64    `json:"by_pt"`
}

// Metrics resume a qualidade do layout em números comparáveis entre rodadas.
type Metrics struct {
	Pages         int     `json:"pages"`
	Overflow      int     `json:"overflow"`        // elementos passando da borda
	LowContrast   int     `json:"low_contrast"`    // textos abaixo do contraste WCAG AA
	BlankPages    int     `json:"blank_pages"`     // páginas sem nenhum conteúdo
	LastPageUsage float64 `json:"last_page_usage"` // fração da altura usada na última página
	Warnings      int     `json:"warnings"`
}

// Limite abaixo do qual a última página é considerada quase vazia.
const emptyTailUsage = 0.15

// Analyze transforma o relatório de layout em avisos W_LAYOUT_* com correção sugerida e em
// métricas. Precisa de um Result renderizado com Job.Layout.
func Analyze(res Result) ([]Diagnostic, Metrics) {
	var out []Diagnostic
	m := Metrics{Pages: len(res.Layout)}
	for i, p := range res.Layout {
		page := i + 1
		for _, o := range p.Overflow {
			if o.Kind == "texto" && strings.TrimSpace(o.Text) == "" {
				continue // espaço não imprime nada: não há o que cortar
			}
			m.Overflow++
			what := o.Kind
			if o.Text != "" {
				what += fmt.Sprintf(" %q", o.Text)
			}
			out = append(out, Diagnostic{
				Code:     "W_LAYOUT_OUT_OF_PAGE",
				Severity: "warning",
				Page:     page,
				Message:  fmt.Sprintf("%s passa %.1f pt da borda da página %d (vai ser cortado na impressão)", what, o.ByPt, page),
				Fix:      "reduza a largura fixa (width) ou o tamanho do elemento, ou use largura relativa (ex.: 100%); para texto longo sem espaços, quebre o texto ou diminua a fonte",
			})
		}
		// Um aviso por par de cores: a mesma correção serve para todos os textos do grupo.
		type pair struct {
			color, bg string
			minimum   float64
		}
		groups := map[pair][]LowContrast{}
		var order []pair
		for _, c := range p.LowContrast {
			m.LowContrast++
			k := pair{c.Color, c.Background, c.Minimum}
			if _, seen := groups[k]; !seen {
				order = append(order, k)
			}
			groups[k] = append(groups[k], c)
		}
		for _, k := range order {
			g := groups[k]
			texts := make([]string, 0, 3)
			for i, c := range g {
				if i == 3 {
					texts = append(texts, fmt.Sprintf("e mais %d", len(g)-3))
					break
				}
				texts = append(texts, fmt.Sprintf("%q", c.Text))
			}
			fix := fmt.Sprintf("aumente o contraste para pelo menos %.1f:1", k.minimum)
			if better, ok := fixContrast(k.color, k.bg, k.minimum); ok {
				fix = fmt.Sprintf("troque a cor do texto %s por %s (contraste %.1f:1 sobre %s)", k.color, better, k.minimum, k.bg)
			}
			out = append(out, Diagnostic{
				Code:     "W_LAYOUT_LOW_CONTRAST",
				Severity: "warning",
				Page:     page,
				Message: fmt.Sprintf("%d texto(s) %s com contraste %.2f:1 sobre %s (mínimo %.1f:1): %s",
					len(g), k.color, g[0].Ratio, k.bg, k.minimum, strings.Join(texts, ", ")),
				Fix: fix,
			})
		}
		if p.Content == nil {
			m.BlankPages++
			out = append(out, Diagnostic{
				Code:     "W_LAYOUT_BLANK_PAGE",
				Severity: "warning",
				Page:     page,
				Message:  fmt.Sprintf("a página %d está em branco", page),
				Fix:      "remova o pagebreak() sobrando ou o elemento que força página nova",
			})
		}
	}
	if n := len(res.Layout); n > 0 {
		last := res.Layout[n-1]
		if last.Content != nil && last.HeightPt > 0 {
			m.LastPageUsage = round2(last.Content[3] / last.HeightPt)
		}
		if n > 1 && last.Content != nil && m.LastPageUsage < emptyTailUsage {
			out = append(out, Diagnostic{
				Code:     "W_LAYOUT_EMPTY_TAIL",
				Severity: "warning",
				Page:     n,
				Message:  fmt.Sprintf("a última página usa só %.0f%% da altura: o documento quase cabe em %d página(s)", m.LastPageUsage*100, n-1),
				Fix:      fmt.Sprintf("reduza espaçamentos ou o tamanho da fonte em ~%.0f%% para caber em %d página(s)", math.Ceil(m.LastPageUsage/float64(n-1)*100)+1, n-1),
			})
		}
	}
	m.Warnings = len(out)
	return out, m
}

func round2(v float64) float64 { return math.Round(v*100) / 100 }

// fixContrast escurece (ou clareia, em fundo escuro) a cor do texto, mantendo o tom, até atingir
// o contraste mínimo. Devolve a cor em hexadecimal.
func fixContrast(fg, bg string, minimum float64) (string, bool) {
	f, ok1 := parseHex(fg)
	b, ok2 := parseHex(bg)
	if !ok1 || !ok2 {
		return "", false
	}
	target := [3]float64{0, 0, 0}
	if luminance(b) < 0.18 {
		target = [3]float64{1, 1, 1}
	}
	for step := 1; step <= 100; step++ {
		t := float64(step) / 100
		c := [3]float64{}
		for i := range c {
			c[i] = f[i] + (target[i]-f[i])*t
		}
		if contrast(c, b) >= minimum+0.15 {
			return toHex(c), true
		}
	}
	return "", false
}

func parseHex(h string) ([3]float64, bool) {
	var r, g, b int
	if len(h) != 7 || h[0] != '#' {
		return [3]float64{}, false
	}
	if _, err := fmt.Sscanf(h[1:], "%02x%02x%02x", &r, &g, &b); err != nil {
		return [3]float64{}, false
	}
	return [3]float64{float64(r) / 255, float64(g) / 255, float64(b) / 255}, true
}

func toHex(c [3]float64) string {
	v := func(x float64) int { return int(math.Round(math.Max(0, math.Min(1, x)) * 255)) }
	return fmt.Sprintf("#%02x%02x%02x", v(c[0]), v(c[1]), v(c[2]))
}

func luminance(c [3]float64) float64 {
	ch := func(v float64) float64 {
		if v <= 0.03928 {
			return v / 12.92
		}
		return math.Pow((v+0.055)/1.055, 2.4)
	}
	return 0.2126*ch(c[0]) + 0.7152*ch(c[1]) + 0.0722*ch(c[2])
}

func contrast(a, b [3]float64) float64 {
	la, lb := luminance(a), luminance(b)
	return (math.Max(la, lb) + 0.05) / (math.Min(la, lb) + 0.05)
}
