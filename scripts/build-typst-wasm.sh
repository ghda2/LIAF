#!/usr/bin/env sh
# Recompila o motor de documentos (engines/typst) para WASM e atualiza o artefato embutido em
# pkg/typst/wasm. Só é preciso rodar ao mudar engines/typst ou a versão do Typst: quem usa a LIAF
# recebe o .wasm.gz pronto e não precisa de Rust.
#
# Requisitos: Rust estável (ver rust-version do crate typst) e `rustup target add wasm32-wasip1`.
set -eu

root=$(cd "$(dirname "$0")/.." && pwd)
cd "$root/engines/typst"

cargo build --release --target wasm32-wasip1
wasm=target/wasm32-wasip1/release/liaf_typst.wasm

if command -v wasm-opt >/dev/null 2>&1; then
	wasm-opt -Oz --enable-bulk-memory --enable-nontrapping-float-to-int --enable-sign-ext \
		"$wasm" -o "$wasm.opt"
	wasm="$wasm.opt"
fi

gzip -9 -n -c "$wasm" > "$root/pkg/typst/wasm/liaf_typst.wasm.gz"
ls -l "$wasm" "$root/pkg/typst/wasm/liaf_typst.wasm.gz"
