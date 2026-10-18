//go:build !liaf_doc_sem_serif

package fonts

import "embed"

// Libertinus Serif — Com serifa; é a fonte de texto padrão do Typst: sem ela, documentos que não escolhem fonte caem em outra família.
// Excluir do build: liafc build --doc-fonts sem "serif" (tag liaf_doc_sem_serif).
//
//go:embed serif
var serifFS embed.FS

func init() { register("serif", "Libertinus Serif", serifFS, "serif") }
