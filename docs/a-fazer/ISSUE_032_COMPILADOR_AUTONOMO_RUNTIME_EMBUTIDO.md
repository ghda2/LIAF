# Issue #032: Compilador Autônomo e Runtime Embutido (Self-Contained Toolchain)

**Status:** Aberta  
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
