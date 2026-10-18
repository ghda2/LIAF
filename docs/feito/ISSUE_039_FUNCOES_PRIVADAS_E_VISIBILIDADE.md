# Issue #039: Funções Privadas e Visibilidade (`pub`) em Módulos

**Status:** Concluída  
**Componente:** `pkg/token`, `pkg/ast`, `pkg/parser`, `pkg/loader`, `pkg/checker`, `pkg/codegen`, `std/`, `docs/linguagem/SPEC.md`  
**Data:** 03 de outubro de 2026  

---

## 1. Contexto e Motivação

Atualmente na LIAF, qualquer declaração feita em um arquivo ou módulo (`fn`, `struct`, `enum`) é automaticamente pública e visível globalmente quando importada via `(import "...")`.

Isso acarreta graves problemas identificados na Issue #031:
1. **Poluição de namespace:** Módulos da biblioteca padrão (`std/jwt`, `std/auth`) exportam helpers internos (como `JwtHeader`, `JwtExp`, `jwt-segredo-valido`).
2. **Colisões falsas (`E_DUPLICATE_DECL`):** Se uma aplicação definir uma função utilitária ou struct interna com o mesmo nome de um helper de um módulo importado, o compilador barra com `E_DUPLICATE_DECL`.
3. **Falta de encapsulamento:** Agentes de IA podem alucinar e chamar funções internas de módulos que não deveriam fazer parte da API pública estável.

Com visibilidade explícita (`pub`) e privado por padrão, os módulos expõem apenas o que é intencional.

---

## 2. Proposta Técnica

### 2.1 Sintaxe

#### Sintaxe Linear (v0.6)
```liaf
module auth

struct TokenInterno
  id str

pub struct Sessao
  usuario str

fn helper_validacao(t str) bool
  return true

pub fn autenticar(credenciais str) Sessao
  return Sessao{usuario: "admin"}
```

#### Sintaxe S-Expressions
```liaf
(module auth
  (struct TokenInterno (fields (id str)))
  (pub struct Sessao (fields (usuario str)))

  (fn helper-validacao (params (t str)) (returns bool) (effects)
    (body (return true)))

  (pub fn autenticar (params (cred str)) (returns Sessao) (effects)
    (body (return (new Sessao "admin")))))
```

### 2.2 Regras Semânticas

1. **Privado por Padrão:** Qualquer declaração (`fn`, `struct`, `enum`) sem o modificador `pub` é privada ao módulo em que foi declarada.
2. **Acesso no Mesmo Módulo:** Declarações dentro do mesmo arquivo ou diretório de módulo podem acessar símbolos privados livremente.
3. **Importação (`import`):**
   - O `import` disponibiliza no escopo do importador apenas símbolos marcados com `pub`.
   - Se o importador tentar acessar um símbolo privado: erro diagnóstico `E_PRIVATE_SYMBOL` ("o símbolo '%s' é privado no módulo '%s'").
4. **Resolução de Colisões (`checkCollisions`):**
   - `E_DUPLICATE_DECL` ocorre apenas quando dois símbolos públicos ou do mesmo arquivo compartilham o mesmo nome.
   - Símbolos privados de módulos distintos não colidem.
5. **Codegen Go:**
   - Funções, structs e enums privados de submódulos importados são prefixados com `<modulo>__<nome>` no Go gerado para evitar colisões no pacote final do Go.

---

## 3. Critérios de Aceite e Progresso

- [x] Token `pub` reconhecido pelo lexer ([`pkg/token/token.go`](file:///c:/Users/gabri/Documents/Teste/liaf/pkg/token/token.go)).
- [x] AST com campo `IsPub` e `Module` em `FuncDecl`, `StructDecl` e `EnumDecl` ([`pkg/ast/ast.go`](file:///c:/Users/gabri/Documents/Teste/liaf/pkg/ast/ast.go)).
- [x] Suporte a `pub` no AST Printer ([`pkg/ast/printer.go`](file:///c:/Users/gabri/Documents/Teste/liaf/pkg/ast/printer.go)).
- [x] Parser linear e S-expressions aceitando `pub fn`, `pub struct`, `pub enum` com testes validados ([`pkg/parser/linear.go`](file:///c:/Users/gabri/Documents/Teste/liaf/pkg/parser/linear.go), [`pkg/parser/decl.go`](file:///c:/Users/gabri/Documents/Teste/liaf/pkg/parser/decl.go)).
- [x] Loader isolando colisões entre declarações privadas de arquivos distintos ([`pkg/loader/loader.go`](file:///c:/Users/gabri/Documents/Teste/liaf/pkg/loader/loader.go)).
- [x] Módulos da biblioteca padrão atualizados com `pub` nas APIs exportadas e helpers privados protegidos ([`std/`](file:///c:/Users/gabri/Documents/Teste/liaf/std/)).
- [x] Codegen Go com mangling de símbolos privados de módulos importados ([`pkg/codegen/codegen.go`](file:///c:/Users/gabri/Documents/Teste/liaf/pkg/codegen/codegen.go), [`pkg/codegen/expr.go`](file:///c:/Users/gabri/Documents/Teste/liaf/pkg/codegen/expr.go)).
- [x] Testes de compilação e execução ponta a ponta ([`pkg/codegen/codegen_visibility_test.go`](file:///c:/Users/gabri/Documents/Teste/liaf/pkg/codegen/codegen_visibility_test.go)).

