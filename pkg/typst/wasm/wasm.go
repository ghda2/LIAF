// Package wasm guarda o motor de documentos (Typst compilado para wasm32-wasip1) embutido.
//
// O artefato é gerado por scripts/build-typst-wasm.sh a partir de engines/typst. Ele é
// versionado já compilado para que quem usa a LIAF não precise de Rust.
package wasm

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"io"
)

//go:embed liaf_typst.wasm.gz
var compressed []byte

// Module devolve o binário WASM descomprimido.
func Module() ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}
