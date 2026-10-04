# Catálogo de Auto-Cura Mecânica (Auto-Healing)

Quando o comando `liafc check <arquivo> --json` falha, analise o código do erro e aplique a correção mecânica correspondente:

| Código | Descrição | Como Corrigir |
|---|---|---|
| `E_UNDEFINED_SYMBOL` | Variável, tipo ou função não declarada | Verifique se importou o módulo correto, corrija a digitação ou declare a variável com `let`. |
| `E_PRIVATE_SYMBOL` | Símbolo é privado no módulo importado | Adicione `pub` na frente da declaração (`pub fn`, `pub struct`, `pub enum`) no módulo de origem ou utilize a API pública exportada. |
| `E_TYPE_MISMATCH` | Tipo passado não corresponde ao tipo esperado | Converta explicitamente usando `str-from-int`, `int-from-str`, `float-from-int` ou ajuste a assinatura da função. |
| `E_TYPE_INFERENCE_FAILED` | O tipo do `let` não pôde ser inferido automaticamente | Especifique o tipo explicitamente: `let x int = expr` ou `let itens (list str) = ...`. |
| `E_UNHANDLED_RESULT` | Expressão que retorna `(result T E)` sem tratamento | Use `try expr`, capture com `match expr` ou adicione bloco `on-err`. |
| `E_NONEXHAUSTIVE_MATCH` | Pattern matching em `enum` não cobre todas as variantes | Adicione todos os `case Enum.Variante` faltantes para cobrir todas as variantes do enum. |
| `E_DUPLICATE_DECL` | Declaração de função/struct com mesmo nome no mesmo escopo | Renomeie uma das declarações para evitar colisão. |
| `E_UNDECLARED_EFFECT` | Função chama builtin/operação impura sem declarar efeito | Adicione o efeito necessário ao cabeçalho da função: `effects(io, fs, net, db, clock, rand)`. |
| `E_UNUSED_EFFECT` | Função declara efeito que nunca utiliza no seu corpo | Remova o efeito desnecessário da lista em `effects(...)`. |
| `E_IMPORT_NOT_FOUND` | Arquivo ou módulo importado não existe | Verifique se é um módulo da `std/` (`std/auth`, `std/cli`, `std/fs`, `std/jwt`, `std/csv`, `std/storage`, `std/cookie`, `std/cors`, `std/seo`) ou corrija o caminho relativo (`./arquivo.liaf`). |
| `E_MISSING_MAIN` | Módulo raiz não possui ponto de entrada `main` | Adicione a função `fn main() void ... end` ou `(fn main () void ...)` no módulo executável. |
| `E_REDUNDANT_BOOL_COMPARE` | Comparação redundante com booleano (`x == true` ou `x == false`) | Remova a comparação: use `x` diretamente ou `not(x)`. |
| `E_DUPLICATE_ROUTE` | Mesma rota (método + caminho) registrada mais de uma vez | Altere o path ou método de uma das rotas duplicadas. |
| `W_MANUAL_JSON_FMT` | Aviso: concatenação manual de JSON de erro | Substitua concatenações manuais de string JSON pelo builtin `json-error(status, msg)`. |
