package docrt

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/jpeg"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"liaf/pkg/typst"
)

// Com LIAF_DOC_OUT=<pasta>, os testes gravam o PDF e a prévia PNG para inspeção visual.
func save(t *testing.T, name, data string) {
	dir := os.Getenv("LIAF_DOC_OUT")
	if dir == "" {
		return
	}
	if err := os.WriteFile(filepath.Join(dir, name), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
}

// O currículo da raiz do repositório (curriculo.typ) é o exemplo de referência: este teste
// garante que ele continua compilando, é determinístico e gera prévia.
func curriculo(t *testing.T) (string, string) {
	t.Helper()
	src, err := os.ReadFile("../../curriculo.typ")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("testdata/cv.json")
	if err != nil {
		t.Fatal(err)
	}
	return string(src), string(data)
}

func TestCurriculoExample(t *testing.T) {
	src, data := curriculo(t)
	pdf := Source(src, data)
	if !pdf.OK {
		t.Fatal(pdf.Error)
	}
	if !bytes.HasPrefix([]byte(pdf.Value), []byte("%PDF-")) {
		t.Fatal("não gerou PDF")
	}
	if again := Source(src, data); again.Value != pdf.Value {
		t.Fatal("a mesma entrada gerou PDFs diferentes")
	}
	png := SourcePNG(src, data)
	if !png.OK {
		t.Fatal(png.Error)
	}
	save(t, "curriculo.pdf", pdf.Value)
	save(t, "curriculo.png", png.Value)
}

func TestCurriculoMinimalData(t *testing.T) {
	src, _ := curriculo(t)
	if res := Source(src, `{"nome": "Ana"}`); !res.OK {
		t.Fatalf("campos opcionais deveriam ser opcionais: %s", res.Error)
	}
}

func TestSourceErrorHasPosition(t *testing.T) {
	res := Source("#let x = \n", "{}")
	if res.OK || !strings.Contains(res.Error, "/main.typ:1:") {
		t.Fatalf("esperava erro com posição, veio %q", res.Error)
	}
}

func TestResponseHeaders(t *testing.T) {
	r := Response("%PDF-1.7", "cv \"x\".pdf")
	if r.ContentType != "application/pdf" {
		t.Fatal(r.ContentType)
	}
	if r.Headers[0].Value != `inline; filename="cv x.pdf"` {
		t.Fatalf("nome não sanitizado: %s", r.Headers[0].Value)
	}
}

// Imagem sintética (gradiente com um círculo) para os testes de arquivos extras.
func fakePhoto(t *testing.T) string {
	t.Helper()
	img := image.NewRGBA(image.Rect(0, 0, 160, 160))
	for y := range 160 {
		for x := range 160 {
			c := color.RGBA{uint8(60 + x/2), uint8(90 + y/3), 170, 255}
			if (x-80)*(x-80)+(y-70)*(y-70) < 38*38 {
				c = color.RGBA{240, 200, 170, 255}
			}
			img.Set(x, y, c)
		}
	}
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85}); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

// Todo recurso do Typst que depende de fonte precisa compilar sem aviso: é isso que garante
// "qualquer documento Typst" dentro do binário.
func TestSourceFontsCoverTypst(t *testing.T) {
	cases := map[string]string{
		"matematica": `$ sum_(i=1)^n i = (n(n+1))/2 $ e $integral_0^1 x^2 dif x$`,
		"serifada":   `#set text(font: "Libertinus Serif")` + "\nTexto com serifa: ação, café.",
		"mono":       "```go\nfunc main() { println(\"oi\") }\n```",
		"padrao":     `Sem escolher fonte nenhuma.`,
	}
	for name, src := range cases {
		t.Run(name, func(t *testing.T) {
			eng := mustEngine(t)
			res, err := eng.Render(context.Background(), typst.Job{Files: map[string][]byte{"/main.typ": []byte(src)}})
			if err != nil {
				t.Fatal(err)
			}
			if err := res.Err(); err != nil {
				t.Fatal(err)
			}
			for _, w := range res.Warnings() {
				t.Errorf("aviso: %s", w)
			}
		})
	}
}

func mustEngine(t *testing.T) *typst.Engine {
	t.Helper()
	eng, err := typst.Default()
	if err != nil {
		t.Fatal(err)
	}
	return eng
}

func TestSourceFilesImage(t *testing.T) {
	src := `#image("/img/foto.jpg", width: 3cm)` + "\n" +
		`#image(bytes("<svg xmlns='http://www.w3.org/2000/svg' width='10' height='10'><circle cx='5' cy='5' r='5'/></svg>"))`
	res := SourceFiles(src, "{}", map[string]string{"img/foto.jpg": fakePhoto(t)})
	if !res.OK {
		t.Fatal(res.Error)
	}
}

func TestCurriculoWithPhoto(t *testing.T) {
	src, data := curriculo(t)
	withPhoto := strings.Replace(data, `"cor":`, `"foto": "foto.jpg", "cor":`, 1)
	files := map[string]string{"foto.jpg": fakePhoto(t)}
	res := SourceFiles(src, withPhoto, files)
	if !res.OK {
		t.Fatal(res.Error)
	}
	png := SourcePNGFiles(src, withPhoto, files)
	if !png.OK {
		t.Fatal(png.Error)
	}
	save(t, "curriculo-foto.png", png.Value)
	missing := SourceFiles(src, withPhoto, nil)
	if missing.OK || !strings.Contains(missing.Error, "foto.jpg") {
		t.Fatalf("foto ausente deveria falhar citando o arquivo, veio %q", missing.Error)
	}
}

func TestExtraPathRejectsReservedAndTraversal(t *testing.T) {
	for _, name := range []string{"", "../x.png", "a/../../x", "main.typ", "/dados.json",
		"@preview/x:1.0.0/lib.typ", `a\b.png`, "a//b.png", "./x.png"} {
		if res := SourceFiles("oi", "{}", map[string]string{name: "x"}); res.OK {
			t.Errorf("aceitou o nome %q", name)
		}
	}
	if res := SourceFiles(`#read("/sub/ok.txt")`, "{}", map[string]string{"sub/ok.txt": "ok"}); !res.OK {
		t.Fatalf("recusou nome válido: %s", res.Error)
	}
}

func TestRegisteredPackageImports(t *testing.T) {
	pkg := fstest.MapFS{
		"pk/preview/saudacao/1.0.0/typst.toml": {Data: []byte("[package]\nname = \"saudacao\"\nversion = \"1.0.0\"\nentrypoint = \"lib.typ\"\n")},
		"pk/preview/saudacao/1.0.0/lib.typ":    {Data: []byte(`#let ola(n) = [Olá, #n!]`)},
	}
	if err := RegisterPackages(pkg, "pk"); err != nil {
		t.Fatal(err)
	}
	ok := Source(`#import "@preview/saudacao:1.0.0": ola`+"\n#ola(\"Ana\")", "{}")
	if !ok.OK {
		t.Fatal(ok.Error)
	}
	missing := Source(`#import "@preview/nao-embutido:9.9.9": x`, "{}")
	if missing.OK || !strings.Contains(missing.Error, "liafc build") {
		t.Fatalf("pacote ausente deveria explicar como embutir, veio %q", missing.Error)
	}
}
