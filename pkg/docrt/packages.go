package docrt

import (
	"io/fs"
	"strings"
	"sync"
)

var (
	pkgMu    sync.RWMutex
	pkgFiles = map[string][]byte{}
)

// RegisterPackages disponibiliza pacotes Typst para todos os documentos do programa. fsys, a
// partir de root, segue o layout <namespace>/<nome>/<versão>/..., como o liafc build grava ao
// embutir os pacotes citados. Chamado pelo código gerado num init().
func RegisterPackages(fsys fs.FS, root string) error {
	files := map[string][]byte{}
	err := fs.WalkDir(fsys, root, func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel := strings.TrimPrefix(p, root+"/")
		parts := strings.SplitN(rel, "/", 4)
		if len(parts) < 4 {
			return nil
		}
		data, err := fs.ReadFile(fsys, p)
		if err != nil {
			return err
		}
		files["@"+parts[0]+"/"+parts[1]+":"+parts[2]+"/"+parts[3]] = data
		return nil
	})
	if err != nil {
		return err
	}
	pkgMu.Lock()
	for k, v := range files {
		pkgFiles[k] = v
	}
	pkgMu.Unlock()
	return nil
}

func addPackages(files map[string][]byte) {
	pkgMu.RLock()
	defer pkgMu.RUnlock()
	for k, v := range pkgFiles {
		files[k] = v
	}
}
