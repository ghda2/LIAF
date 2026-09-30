// Package lib é a biblioteca de documentos da LIAF escrita em Typst: tokens de design
// (tema.typ), componentes (componentes.typ) e modelos prontos (modelos/*.typ).
//
// Os arquivos são montados em /liaf/ em toda renderização, então um documento Typst próprio
// também pode usá-los: #import "/liaf/componentes.typ": etiqueta
package lib

import (
	"embed"
	"io/fs"
	"path"
	"sort"
	"strings"
)

//go:embed *.typ modelos/*.typ
var files embed.FS

// Mount é o diretório virtual onde a biblioteca aparece para o documento.
const Mount = "/liaf/"

// Files devolve todos os arquivos da biblioteca já com o caminho virtual (/liaf/...).
func Files() map[string][]byte {
	out := map[string][]byte{}
	_ = fs.WalkDir(files, ".", func(p string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := files.ReadFile(p)
		if err != nil {
			return err
		}
		out[Mount+p] = data
		return nil
	})
	return out
}

// Templates lista os modelos disponíveis (nome sem extensão), em ordem.
func Templates() []string {
	entries, _ := fs.ReadDir(files, "modelos")
	var names []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".typ") {
			names = append(names, strings.TrimSuffix(e.Name(), ".typ"))
		}
	}
	sort.Strings(names)
	return names
}

// HasTemplate diz se o modelo existe.
func HasTemplate(name string) bool {
	if name == "" || strings.ContainsAny(name, "/\\.") {
		return false
	}
	_, err := fs.Stat(files, path.Join("modelos", name+".typ"))
	return err == nil
}

// TemplateMain é o arquivo principal que aplica o modelo aos dados em dataPath.
func TemplateMain(name, dataPath string) []byte {
	return []byte("#import \"" + Mount + "modelos/" + name + ".typ\": modelo\n" +
		"#modelo(json(\"" + dataPath + "\"))\n")
}
