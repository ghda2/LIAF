// Package liaf embute os fontes Go de que todo programa LIAF compilado
// precisa (issue #032).
//
// O codigo gerado importa liaf/pkg/web e liaf/pkg/runtime. Para que liafc
// build funcione em qualquer pasta, sem clone do repositorio nem go.mod com
// replace, esses pacotes (e os pacotes internos de que dependem) viajam dentro
// do liafc e sao materializados num cache por pkg/toolchain. go.mod e go.sum
// vao junto para fixar as mesmas versoes de terceiros com que o compilador
// foi testado.
//
// Se o runtime passar a importar outro pacote liaf/pkg/..., ele tem de entrar
// aqui; TestRuntimeFSCoversRuntimeDeps em pkg/toolchain falha ate isso
// acontecer.
package liaf

import "embed"

//go:embed go.mod go.sum
//go:embed pkg/runtime/*.go pkg/web/*.go pkg/dbdrv/*.go pkg/markdown/*.go pkg/minifier/*.go
var RuntimeFS embed.FS
