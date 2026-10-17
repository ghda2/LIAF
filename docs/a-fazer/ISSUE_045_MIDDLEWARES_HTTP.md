# Issue #045 — Middlewares HTTP Declarativos

**Estado:** Aberta. **Criada em:** 17/10/2026. **Tipo:** web/linguagem.  
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
