//go:build !liaf_doc_sem_mono

package fonts

import "embed"

// DejaVu Sans Mono — Monoespaçada: blocos de código e textos para copiar.
// Excluir do build: liafc build --doc-fonts sem "mono" (tag liaf_doc_sem_mono).
//
//go:embed mono
var monoFS embed.FS

func init() { register("mono", "DejaVu Sans Mono", monoFS, "mono") }
