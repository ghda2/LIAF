package typst

import (
	"context"
	"strings"
	"testing"
)

func analyze(t *testing.T, src string) ([]Diagnostic, Metrics, Result) {
	t.Helper()
	res, err := engine(t).Render(context.Background(), Job{Files: src2(src), Format: FormatCheck, Layout: true})
	if err != nil {
		t.Fatal(err)
	}
	if err := res.Err(); err != nil {
		t.Fatal(err)
	}
	warns, m := Analyze(res)
	return warns, m, res
}

func src2(s string) map[string][]byte {
	return map[string][]byte{"/main.typ": []byte("#set text(font: \"Inter\")\n" + s)}
}

func codes(ws []Diagnostic) map[string]int {
	out := map[string]int{}
	for _, w := range ws {
		out[w.Code]++
	}
	return out
}

func TestAnalyzeCleanDocument(t *testing.T) {
	warns, m, res := analyze(t, "= Título\nUm parágrafo que cabe na página.")
	if len(warns) != 0 {
		t.Fatalf("documento limpo gerou avisos: %+v", warns)
	}
	if m.Pages != 1 || res.Layout[0].Content == nil || res.Layout[0].Texts == 0 {
		t.Fatalf("relatório incompleto: %+v / %+v", m, res.Layout)
	}
}

func TestAnalyzeOutOfPage(t *testing.T) {
	warns, m, _ := analyze(t, `#place(dx: 500pt, rect(width: 200pt, height: 20pt, fill: red))
#box(width: 900pt)[texto largo demais para a página]`)
	if codes(warns)["W_LAYOUT_OUT_OF_PAGE"] == 0 || m.Overflow == 0 {
		t.Fatalf("estouro não detectado: %+v", warns)
	}
	for _, w := range warns {
		if w.Fix == "" || w.Page != 1 {
			t.Fatalf("aviso sem correção ou página: %+v", w)
		}
	}
}

func TestAnalyzeBlankPageAndEmptyTail(t *testing.T) {
	src := "#lorem(700)\n#pagebreak()\n#pagebreak()\nFim."
	warns, m, _ := analyze(t, src)
	c := codes(warns)
	if c["W_LAYOUT_BLANK_PAGE"] != 1 || m.BlankPages != 1 {
		t.Fatalf("página em branco não detectada: %+v", warns)
	}
	if c["W_LAYOUT_EMPTY_TAIL"] != 1 || m.LastPageUsage >= emptyTailUsage {
		t.Fatalf("última página quase vazia não detectada: %+v %+v", warns, m)
	}
}

func TestAnalyzeIgnoresInvisibleOverflow(t *testing.T) {
	// Uma caixa larga sem fundo não imprime nada fora da página: só espaços caem lá.
	warns, _, _ := analyze(t, "#box(width: 700pt)[curto]\nmais texto")
	if codes(warns)["W_LAYOUT_OUT_OF_PAGE"] != 0 {
		t.Fatalf("espaço em branco contado como estouro: %+v", warns)
	}
}

func TestAnalyzeClippedContentIsNotOverflow(t *testing.T) {
	// Recorte intencional (ex.: foto num círculo) não é estouro.
	warns, _, _ := analyze(t, `#box(width: 40pt, height: 40pt, clip: true, rect(width: 400pt, height: 400pt, fill: blue))`)
	if codes(warns)["W_LAYOUT_OUT_OF_PAGE"] != 0 {
		t.Fatalf("conteúdo recortado contado como estouro: %+v", warns)
	}
}

func TestAnalyzeLowContrast(t *testing.T) {
	src := "#text(fill: rgb(\"#b0b0b0\"))[cinza claro no branco]\n\n" +
		"#box(fill: rgb(\"#1e3a8a\"), inset: 4pt, text(fill: white)[branco no azul escuro])\n\n" +
		"#box(fill: rgb(\"#2563eb\"), inset: 4pt, text(fill: rgb(\"#1e40af\"))[azul no azul])"
	warns, m, _ := analyze(t, src)
	if m.LowContrast != 2 {
		t.Fatalf("esperava 2 textos de baixo contraste (cinza no branco, azul no azul), veio %d: %+v", m.LowContrast, warns)
	}
	for _, w := range warns {
		if w.Code == "W_LAYOUT_LOW_CONTRAST" && !strings.Contains(w.Fix, "troque a cor") {
			t.Fatalf("aviso sem cor sugerida: %+v", w)
		}
	}
}

func TestFixContrastReachesMinimum(t *testing.T) {
	better, ok := fixContrast("#b0b0b0", "#ffffff", 4.5)
	if !ok {
		t.Fatal("sem sugestão")
	}
	f, _ := parseHex(better)
	if c := contrast(f, [3]float64{1, 1, 1}); c < 4.5 {
		t.Fatalf("sugestão %s tem contraste %.2f", better, c)
	}
	// Fundo escuro: clareia em vez de escurecer.
	light, _ := fixContrast("#334155", "#0f172a", 4.5)
	if l, _ := parseHex(light); luminance(l) <= luminance([3]float64{0x33 / 255.0, 0x41 / 255.0, 0x55 / 255.0}) {
		t.Fatalf("em fundo escuro deveria clarear, veio %s", light)
	}
}
