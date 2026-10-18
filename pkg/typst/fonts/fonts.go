// Package fonts reúne as famílias de fonte que o motor de documentos carrega.
//
// Cada família fica num arquivo próprio, com uma build tag para excluí-la
// (liaf_doc_sem_<familia>); o liafc build --doc-fonts escolhe as tags. As fontes vão
// comprimidas (gzip) e o texto da licença de cada família vai junto no binário.
package fonts

import (
	"bytes"
	"compress/gzip"
	"embed"
	"io"
	"io/fs"
	"sort"
	"strings"
	"sync"
)

// Pack é uma família de fontes: nome (como o documento a chama), os arquivos de cada face e
// o texto da licença.
type Pack struct {
	ID      string // inter, serif, math, mono
	Name    string
	Files   [][]byte
	License string
}

type source struct {
	id, name string
	fsys     embed.FS
	dir      string
}

var (
	mu        sync.Mutex
	available []source
	cache     = map[string]Pack{}
)

// register é chamado pelo init() de cada família incluída no build.
func register(id, name string, fsys embed.FS, dir string) {
	mu.Lock()
	defer mu.Unlock()
	available = append(available, source{id, name, fsys, dir})
	sort.Slice(available, func(i, j int) bool { return available[i].id < available[j].id })
}

// IDs lista as famílias incluídas neste binário.
func IDs() []string {
	mu.Lock()
	defer mu.Unlock()
	ids := make([]string, len(available))
	for i, s := range available {
		ids[i] = s.id
	}
	return ids
}

// Default devolve todas as famílias incluídas neste binário, já descomprimidas.
func Default() []Pack {
	mu.Lock()
	srcs := append([]source(nil), available...)
	mu.Unlock()
	packs := make([]Pack, 0, len(srcs))
	for _, s := range srcs {
		packs = append(packs, load(s))
	}
	return packs
}

// Notices devolve as licenças de todas as famílias incluídas, para exibir ou distribuir.
func Notices() string {
	// Famílias que vêm do mesmo projeto compartilham o arquivo de licença: imprime uma vez só.
	var b strings.Builder
	var order []string
	names := map[string][]string{}
	for _, p := range Default() {
		if _, seen := names[p.License]; !seen {
			order = append(order, p.License)
		}
		names[p.License] = append(names[p.License], p.Name)
	}
	for _, lic := range order {
		b.WriteString("==== Fontes: " + strings.Join(names[lic], ", ") + " ====\n" + lic + "\n")
	}
	return b.String()
}

func load(s source) Pack {
	mu.Lock()
	if p, ok := cache[s.id]; ok {
		mu.Unlock()
		return p
	}
	mu.Unlock()

	entries, err := fs.ReadDir(s.fsys, s.dir)
	if err != nil {
		panic("fonts: família embutida ausente: " + s.dir)
	}
	p := Pack{ID: s.id, Name: s.name}
	for _, e := range entries {
		data, err := s.fsys.ReadFile(s.dir + "/" + e.Name())
		if err != nil {
			panic("fonts: " + err.Error())
		}
		switch {
		case strings.HasSuffix(e.Name(), ".gz"):
			p.Files = append(p.Files, gunzip(data))
		case strings.HasSuffix(e.Name(), ".txt"):
			p.License = string(data)
		}
	}
	mu.Lock()
	cache[s.id] = p
	mu.Unlock()
	return p
}

func gunzip(data []byte) []byte {
	r, err := gzip.NewReader(bytes.NewReader(data))
	if err != nil {
		panic("fonts: arquivo embutido corrompido: " + err.Error())
	}
	defer r.Close()
	out, err := io.ReadAll(r)
	if err != nil {
		panic("fonts: arquivo embutido corrompido: " + err.Error())
	}
	return out
}
