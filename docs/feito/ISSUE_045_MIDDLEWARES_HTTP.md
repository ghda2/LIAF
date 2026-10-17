# Issue #045 — Middlewares HTTP Declarativos

**Estado:** Implementada em 17/10/2026. **Criada em:** 17/10/2026. **Tipo:** web/linguagem.  
**Componentes:** `pkg/parser`, `pkg/checker`, `pkg/codegen`, `pkg/web`.  

---

## 1. Contexto e Objetivo
Atualmente, qualquer validação de cabeçalhos, CORS ou autenticação precisa ser repetida explicitamente em cada rota (`GET`, `POST`, etc.). Middlewares permitem interceptar requisições antes de chegarem ao handler final, interrompendo a cadeia ou adicionando contexto.

---

## 2. Proposta Técnica

1. **Declaração de Middleware:**
   ```liaf
   middleware auth_guard(req Request) (result Request Response) effects(net)
     let token = try request-header(req, "Authorization")
     if str-len(token) == 0
       return err(response(401, "Token obrigatorio"))
     end
     ok(req)
   end
   ```
2. **Aplicação em Rotas ou Grupos:**
   - Sintaxe em rotas: `route GET "/painel" with(auth_guard) ...` ou escopo global no módulo.
3. **Casos de Aceitação:**
   - Interceptação com sucesso e com rejeição 401 em testes de integração de codegen.

---

## 3. Implementação (17/10/2026)

O que foi entregue difere da proposta em dois pontos, ambos por causa do checker:

- `try request-header(...)` não serve num middleware: o erro de `request-header` é `str` e o do
  middleware é `Response`. Use `unwrap-or` ou `match` e devolva `err(json-error(401, ...))`.
- Não existe builtin `response(...)`; a resposta de rejeição sai de `json-error`,
  `json-response` ou `html-response`.

**Linguagem.** `middleware` e `with` são palavras contextuais (continuam válidas como nomes). Um
middleware é uma `fn` de assinatura fixa `(req Request) (result Request Response)`, aceita `pub`
e efeitos próprios. Na rota, `with(a, b)` vem logo depois do caminho; em S-expression,
`(with a b)`. O `fmt` imprime as duas formas e é idempotente.

**Semântica.** Os middlewares rodam na ordem do `with`, antes da extração dos parâmetros.
`ok(req)` segue adiante com a `Request` devolvida; `err(resposta)` encerra a cadeia. Para
entregar contexto à rota, o novo builtin `request-set-header(req, nome, valor)` devolve uma cópia
da requisição com o header substituído, o que também descarta um header homônimo forjado pelo
cliente.

**Diagnósticos.** `E_MIDDLEWARE_SIGNATURE`, `E_UNKNOWN_MIDDLEWARE` (nome inexistente ou declarado
com `fn`), `E_DUPLICATE_MIDDLEWARE`, `E_EXPECTED_MIDDLEWARE_NAME`.

**Testes.** `pkg/parser/middleware_test.go` (as duas sintaxes e ida e volta pelo `fmt`),
`pkg/checker/checker_middleware_test.go`, `pkg/codegen/codegen_middleware_test.go` com
`testdata/middleware_api.liaf`: sem token e com token errado dá 401; com token dá 200, com o
contexto dos dois middlewares e o parâmetro de path.

**Fora do escopo.** Escopo global (aplicar a todas as rotas do módulo) e `with` em `ws-route`.
