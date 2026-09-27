# Instruções para o Modelo de IA

Você está gerando código na linguagem **LIAF (Language for AI First)** v0.3/v0.4.
O código é estritamente formatado em S-expressions (estilo Lisp/Clojure). Não use tags antigas como `[fn ...]`.

---

## Regras Obrigatórias

1. **Sintaxe Única:**
   Todo bloco começa e termina com parênteses `( ... )`.
   O primeiro elemento é a operação ou construto: `(fn ...)`, `(let ...)`, `(if ...)`, `(add a b)`.

2. **Assinatura de Funções:**
   Toda função DEVE declarar obrigatoriamente:
   - `(params ...)`: parâmetros tipados, ex: `(params (x int) (nome str))`
   - `(returns ...)`: tipo de retorno, ex: `(returns int)` ou `(returns void)`
   - `(effects ...)`: efeitos colaterais utilizados. Se a função for pura, declare `(effects)`. Se usar print/console use `io`, arquivos `fs`, rede `net`, banco `db`. **Declarar efeito não usado gera erro no compilador!**
   - `(body ...)`: corpo da função.

3. **Tipos Básicos:**
   - Primitivos: `int`, `float`, `str`, `bool`, `void`
   - Compostos: `(list T)`, `(map K V)`, `(result T E)`

4. **Operações Comuns:**
   - Aritmética: `(add a b)`, `(sub a b)`, `(mul a b)`, `(div a b)`
   - Comparação: `(eq a b)`, `(neq a b)`, `(lt a b)`, `(lte a b)`, `(gt a b)`, `(gte a b)`
   - Impressão: `(println "mensagem")` (requer efeito `io`)
   - Atribuição/Variáveis:
     - Criação: `(let nome tipo valor)`
     - Mutação de variável local: `(set nome novo_valor)`

5. **Tratamento de Erros:**
   - Operações falíveis retornam `(result T E)`.
   - Use `(try expr)` para propagar erros ou `(match expr (ok v ...) (err m ...))`.
   - Se a função usar `try`, inclua antes de `(body ...)` a cláusula `(on-err msg ...)`.

6. **Validação:**
   Para validar sua solução, execute no terminal desta pasta:
   ```bash
   ..\..\liafc.exe check solucao.liaf --json
   ```
   Se o compilador retornar erros no JSON, leia o campo `code` e `suggested_patch` e corrija o arquivo `solucao.liaf`.
