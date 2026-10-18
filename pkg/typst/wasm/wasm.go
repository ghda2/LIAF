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

//go:embed THIRD_PARTY.txt
var notices string

// Notices devolve as licenças do Typst e das dependências Rust embutidas no motor. O arquivo é
// gerado por scripts/build-typst-wasm.sh junto com o .wasm.gz.
func Notices() string { return notices }

// Module devolve o binário WASM descomprimido.
func Module() ([]byte, error) {
	r, err := gzip.NewReader(bytes.NewReader(compressed))
	if err != nil {
		return nil, err
	}
	defer r.Close()
	return io.ReadAll(r)
}
