package builtins

// Documentos e PDF (issue #034): motor Typst embutido, em pkg/docrt. O import é condicional
// (GoImport) para o motor só entrar no binário de quem gera documentos. Não há modelos
// prontos: o layout é código Typst do próprio programa.
const docImport = `liafdoc "liaf/pkg/docrt"`

func docBuiltin(name, goCall string, params []string, ret string) *Builtin {
	return &Builtin{
		Name:               name,
		Category:           "doc",
		Arity:              ExactArity(len(params)),
		Params:             params,
		Return:             ret,
		GoCall:             goCall,
		GoImport:           docImport,
		UnsupportedC:       true,
		UnsupportedCReason: "documentos exigem o motor Typst do backend Go",
	}
}

func init() {
	// (fonte Typst, dados em JSON): os dados ficam em /dados.json.
	Register(docBuiltin("pdf-typst", "liafdoc.Source", []string{"str", "str"}, "(result str str)"))
	Register(docBuiltin("png-typst", "liafdoc.SourcePNG", []string{"str", "str"}, "(result str str)"))

	// Variantes com arquivos extras (foto, logo, SVG, CSV): o mapa vai do nome do arquivo
	// ("foto.jpg", "img/logo.png") aos bytes, e o documento lê em "/foto.jpg".
	files := []string{"str", "str", "(map str str)"}
	Register(docBuiltin("pdf-typst-files", "liafdoc.SourceFiles", files, "(result str str)"))
	Register(docBuiltin("png-typst-files", "liafdoc.SourcePNGFiles", files, "(result str str)"))

	Register(docBuiltin("pdf-response", "liafdoc.Response", []string{"str", "str"}, "Response"))
}
