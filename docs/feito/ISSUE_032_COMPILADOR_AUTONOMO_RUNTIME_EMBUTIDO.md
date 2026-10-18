# Issue #032: Compilador Autônomo e Runtime Embutido (Self-Contained Toolchain)

**Status:** Concluída em 27/09/2026 — ver "Conclusão" no fim  
**Componente:** `cmd/liafc`, `pkg/loader`, `pkg/codegen`, `pkg/runtime`  
**Data:** 26 de setembro de 2026  

---

## 1. Contexto

Atualmente, quando um usuário ou agente de IA cria um projeto LIAF em um diretório fora da raiz do compilador (ex: `liaf_ecomerce/cardapio.liaf`), o comando `liafc build` falha caso não existam os arquivos `go.mod` e `go.sum` com a diretiva:
```go
replace liaf => ../liaf
```

Isso ocorre porque o gerador Go emite chamadas para `liaf/pkg/runtime` e `liaf/pkg/web`, exigindo a presença dos fontes locais do compilador na máquina hospedeira. A linguagem hoje não pode ser distribuída como um único executável CLI autônomo.

---

## 2. Proposta

Transformar o `liafc` em uma toolchain **100% autossuficiente (Self-Contained Single Binary)**:

1. **Embutir Fontes do Runtime via `//go:embed`:**
   - Similar ao que já foi feito com `std/`, embutir recursivamente os fontes de `pkg/runtime`, `pkg/web`, etc., no executável `liafc`.
2. **Materialização em Cache VFS Temporário:**
   - Durante a compilação (`liafc build` ou `liafc run`), o compilador gera um ambiente temporário em `%TEMP%/liaf_cache/` ou `.liaf_cache/` contendo o runtime vendorizado.
3. **Zero Configuração de `go.mod` para o Usuário:**
   - O desenvolvedor ou IA precisa apenas do arquivo `.liaf`. Rodar `liafc build app.liaf -o server` deve funcionar em qualquer diretório sem exigir `git clone`, `go.mod` ou paths relativos.

---

## 3. Critérios de Aceitação

1. O comando `liafc build` compila qualquer arquivo `.liaf` fora da árvore do repositório sem exigir `go.mod` local.
2. O binário `liafc` pode ser baixado/instalado isoladamente e funcionar como compilador independente.
3. Suíte de testes `conformance_test.go` continua passando 100%.

---

## Conclusão (27/09/2026)

**Como ficou.** [runtime_embed.go](../../runtime_embed.go), na raiz do módulo, embute `go.mod`, `go.sum`
e os fontes de `pkg/runtime`, `pkg/web`, `pkg/dbdrv`, `pkg/markdown` e `pkg/minifier`: exatamente os
pacotes `liaf/...` alcançados pelos dois imports do código gerado. [pkg/toolchain](../../pkg/toolchain/toolchain.go)
grava esses arquivos (sem `_test.go`) em `<UserCacheDir>/liaf/runtime-<hash>/`, uma vez por versão do
runtime. O diretório pode ser trocado com `LIAF_CACHE`. A escrita acontece numa pasta temporária
renomeada no fim, então dois `liafc` em paralelo nunca veem o módulo pela metade. Cada build cria um pacote
`app-*` dentro desse módulo, compila e apaga o pacote. Como o módulo se chama `liaf`, os imports
`liaf/pkg/...` resolvem sem `replace`.

- `liafc build` funciona em qualquer pasta. `--embed` continua funcionando: a pasta `public/` vai para
  dentro do pacote `app-*`.
- `liafc run` compila e executa o binário **na pasta atual do usuário**. Antes, com `go run`, caminhos
  relativos de `fs-*` resolviam no diretório do `go.mod`. O código de saída do programa é repassado
  (`(exit 3)` sai com 3) e argumentos depois do `.liaf` chegam ao programa.
- O `go build` roda com `GOWORK=off` (um `go.work` do usuário não interfere) e `CGO_ENABLED=0` quando
  a variável não está definida. O runtime é Go puro (sqlite do modernc), então o binário sai estático e
  a cross-compilação `GOOS=linux` funciona mesmo com `CGO_ENABLED=1` no `go env`. Nesta máquina esse
  era o caso, e a receita de cross-compilação da SKILL falhava.
- No Windows, `-o` sem extensão (ou a saída padrão) ganha `.exe`, como no `go build`. A regra olha o
  `GOOS` de destino.

**Testes.**

- `cmd/liafc/standalone_test.go` compila o `liafc`, cria um `.liaf` numa pasta temporária fora do
  repositório e roda `liafc run` e `liafc build` com `GOPROXY=off` e `LIAF_CACHE` vazio. Verifica a saída e a
  leitura de arquivo relativo.
- `pkg/toolchain/toolchain_test.go` compara o conjunto embutido com `go list -deps` do runtime: se um
  pacote interno novo for importado e não entrar no `//go:embed`, o teste falha com a instrução. Também
  cobre a reutilização do cache e a exclusão de testes.
- Conformidade e integração continuam passando.

**O que o critério 2 ainda não cobre.** O `liafc` isolado dispensa o repositório, mas não dispensa o
**comando `go`** na máquina: o backend continua sendo o compilador Go (independência é a #010). As
dependências de terceiros (`golang.org/x/crypto`, `modernc.org/sqlite` e transitivas) saem do cache de
módulos do Go. Numa máquina que nunca as baixou, a primeira compilação precisa de acesso ao proxy de
módulos. Embutir essas dependências (`vendor/`) foi descartado porque o `modernc.org/libc` sozinho
multiplicaria o tamanho do `liafc`. Versões antigas do runtime em cache não são apagadas
automaticamente; ocupam cerca de 250 KB cada.
