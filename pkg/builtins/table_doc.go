package builtins

// Documentos e PDF (issue #034): motor Typst embutido, em pkg/docrt. O import é condicional
// (GoImport) para o motor só entrar no binário de quem gera documentos.
const docImport = `liafdoc "liaf/pkg/docrt"`

func init() {
	Register(&Builtin{
		Name:               "pdf-template",
		Category:           "doc",
		Arity:              ExactArity(2),
		Params:             []string{"str", "str"},
		Return:             "(result str str)",
		GoCall:             "liafdoc.Template",
		GoImport:           docImport,
		UnsupportedC:       true,
		UnsupportedCReason: "documentos exigem o motor Typst do backend Go",
	})

	Register(&Builtin{
		Name:               "pdf-typst",
		Category:           "doc",
		Arity:              ExactArity(2),
		Params:             []string{"str", "str"},
		Return:             "(result str str)",
		GoCall:             "liafdoc.Source",
		GoImport:           docImport,
		UnsupportedC:       true,
		UnsupportedCReason: "documentos exigem o motor Typst do backend Go",
	})

	// Variantes com arquivos extras (foto, logo, SVG, CSV): o mapa vai do nome do arquivo
	// ("foto.jpg", "img/logo.png") aos bytes, e o documento lê em "/foto.jpg".
	Register(&Builtin{
		Name:               "pdf-template-files",
		Category:           "doc",
		Arity:              ExactArity(3),
		Params:             []string{"str", "str", "(map str str)"},
		Return:             "(result str str)",
		GoCall:             "liafdoc.TemplateFiles",
		GoImport:           docImport,
		UnsupportedC:       true,
		UnsupportedCReason: "documentos exigem o motor Typst do backend Go",
	})

	Register(&Builtin{
		Name:               "pdf-typst-files",
		Category:           "doc",
		Arity:              ExactArity(3),
		Params:             []string{"str", "str", "(map str str)"},
		Return:             "(result str str)",
		GoCall:             "liafdoc.SourceFiles",
		GoImport:           docImport,
		UnsupportedC:       true,
		UnsupportedCReason: "documentos exigem o motor Typst do backend Go",
	})

	Register(&Builtin{
		Name:               "png-template",
		Category:           "doc",
		Arity:              ExactArity(2),
		Params:             []string{"str", "str"},
		Return:             "(result str str)",
		GoCall:             "liafdoc.TemplatePNG",
		GoImport:           docImport,
		UnsupportedC:       true,
		UnsupportedCReason: "documentos exigem o motor Typst do backend Go",
	})

	Register(&Builtin{
		Name:               "pdf-response",
		Category:           "doc",
		Arity:              ExactArity(2),
		Params:             []string{"str", "str"},
		Return:             "Response",
		GoCall:             "liafdoc.Response",
		GoImport:           docImport,
		UnsupportedC:       true,
		UnsupportedCReason: "documentos exigem o motor Typst do backend Go",
	})
}
