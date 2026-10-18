#!/usr/bin/env sh
# Recompila o motor de documentos (engines/typst) para WASM e atualiza os artefatos embutidos em
# pkg/typst/wasm: liaf_typst.wasm.gz e THIRD_PARTY.txt (licenças das dependências Rust).
# Só é preciso rodar ao mudar engines/typst ou a versão do Typst: quem usa a LIAF recebe o
# .wasm.gz pronto e não precisa de Rust.
#
#   scripts/build-typst-wasm.sh            # Rust local (versão fixada em rust-toolchain.toml)
#   scripts/build-typst-wasm.sh --docker   # dentro do contêiner oficial do Rust, igual ao CI
#   scripts/build-typst-wasm.sh --check    # recompila e falha se o artefato versionado difere
#
# Reprodutibilidade: os caminhos da máquina são removidos do binário (--remap-path-prefix), a
# versão do Rust é fixa e o gzip não grava data nem nome. O CI (Linux) é a referência.
set -eu

root=$(cd "$(dirname "$0")/.." && pwd)
mode=${1:-}
rust_image="rust:1.98.1-bookworm"

if [ "$mode" = "--docker" ]; then
	exec docker run --rm -v "$root":/liaf -w /liaf "$rust_image" \
		sh -c "rustup target add wasm32-wasip1 >/dev/null && sh scripts/build-typst-wasm.sh"
fi

cd "$root/engines/typst"
cargo_home=${CARGO_HOME:-$HOME/.cargo}
export RUSTFLAGS="--remap-path-prefix=$cargo_home=/cargo --remap-path-prefix=$root=/liaf"
cargo build --release --locked --target wasm32-wasip1
wasm=target/wasm32-wasip1/release/liaf_typst.wasm

out="$root/pkg/typst/wasm"
new=$(mktemp)
gzip -9 -n -c "$wasm" > "$new"

# Licenças das dependências Rust que vão dentro do motor.
notices=$(mktemp)
{
	echo "Motor de documentos da LIAF (engines/typst), compilado para WebAssembly."
	echo "Inclui o Typst (https://typst.app, Apache-2.0) e as dependências abaixo,"
	echo "cada uma sob a licença indicada (texto completo no crate de cada uma)."
	echo
	cargo tree --locked --target wasm32-wasip1 -e normal --prefix none --format "{p} | {l}" \
		| sed 's/ (\*)$//; s/ (proc-macro)$//' | sort -u
} > "$notices"

if [ "$mode" = "--check" ]; then
	if cmp -s "$new" "$out/liaf_typst.wasm.gz" && cmp -s "$notices" "$out/THIRD_PARTY.txt"; then
		echo "motor WASM confere com o artefato versionado"
		rm -f "$new" "$notices"
		exit 0
	fi
	cp "$new" "${LIAF_WASM_OUT:-/tmp}/liaf_typst.wasm.gz" 2>/dev/null || true
	echo "motor WASM difere do versionado: rode scripts/build-typst-wasm.sh --docker e versione o resultado" >&2
	sha256sum "$new" "$out/liaf_typst.wasm.gz" >&2 || true
	exit 1
fi

mv "$new" "$out/liaf_typst.wasm.gz"
mv "$notices" "$out/THIRD_PARTY.txt"
ls -l "$wasm" "$out/liaf_typst.wasm.gz" "$out/THIRD_PARTY.txt"
