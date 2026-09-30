// Package fonts reúne os pacotes de fontes que o motor de documentos pode carregar.
//
// Cada pacote é embutido à parte, para o binário levar só as famílias usadas. Todas as fontes
// aqui têm licença livre (OFL) e o arquivo de licença acompanha cada pacote.
package fonts

import (
	"embed"
	"io/fs"
	"sort"
	"strings"
)

// Pack é uma família de fontes: nome (como o documento a chama) e os arquivos de cada face.
type Pack struct {
	Name  string
	Files [][]byte
}

var (
	//go:embed inter/*.ttf
	interFS embed.FS
	//go:embed serif/*.otf
	serifFS embed.FS
	//go:embed math/*.otf
	mathFS embed.FS
	//go:embed mono/*.ttf
	monoFS embed.FS
)

// Inter é a família sem serifa padrão dos temas modernos (6 pesos: 300 a 700 e itálico).
func Inter() Pack { return load("Inter", interFS, "inter") }

// Serif é a Libertinus Serif (regular, itálico, seminegrito, negrito): textos longos, temas
// clássicos. É a fonte de texto padrão do Typst.
func Serif() Pack { return load("Libertinus Serif", serifFS, "serif") }

// Math é a New Computer Modern Math: sem ela, fórmulas ($ ... $) não compilam.
func Math() Pack { return load("New Computer Modern Math", mathFS, "math") }

// Mono é a DejaVu Sans Mono: blocos de código (raw) e dados tabulares.
func Mono() Pack { return load("DejaVu Sans Mono", monoFS, "mono") }

// Default é o conjunto carregado quando o chamador não escolhe as fontes: cobre texto sem
// serifa, com serifa, matemática e código, para que qualquer documento Typst compile.
func Default() []Pack { return []Pack{Inter(), Serif(), Math(), Mono()} }

func load(name string, fsys embed.FS, dir string) Pack {
	entries, err := fs.ReadDir(fsys, dir)
	if err != nil {
		panic("fonts: pacote embutido ausente: " + dir)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".ttf") || strings.HasSuffix(e.Name(), ".otf") {
			names = append(names, e.Name())
		}
	}
	sort.Strings(names)
	p := Pack{Name: name}
	for _, n := range names {
		data, err := fsys.ReadFile(dir + "/" + n)
		if err != nil {
			panic("fonts: " + err.Error())
		}
		p.Files = append(p.Files, data)
	}
	return p
}
