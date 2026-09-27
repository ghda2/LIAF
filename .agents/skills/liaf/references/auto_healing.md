# Catálogo de Auto-Cura Mecânica (Auto-Healing)

Quando o comando `liafc check <arquivo> --json` falha, analise o código do erro e aplique a correção pontual:

| Código | Descrição | Como Corrigir |
|---|---|---|
| `E_UNDEFINED_SYMBOL` | Variável, tipo ou função não declarada | Verifique imports, ortografia ou declare com `let`. |
| `E_TYPE_MISMATCH` | Tipo passado não corresponde ao esperado | Converta tipos com `int-from-str`, `str-from-int`, ou ajuste a assinatura. |
| `E_TYPE_INFERENCE_FAILED` | `let` sem tipo explícito onde não foi possível inferir | Especifique o tipo: `let x int = expr`. |
| `E_UNHANDLED_RESULT` | Expressão que retorna `(result T E)` sem desempacotar | Use `try expr`, `unwrap-or(expr, fallback)` ou `match`. |
| `E_EFFECT_MISMATCH` | Função realiza operação impura sem declarar efeito | Adicione o efeito faltante em `effects(fs, io, net, db, etc.)`. |
| `E_MISSING_MAIN` | Módulo compilado como executável sem função `main` | Adicione `fn main() void ... end`. |
| `E_REDUNDANT_BOOL_COMPARE` | Comparação explícita com `true` ou `false` (`x == true`) | Remova a comparação: use `x` diretamente ou `not(x)`. |
| `E_DUPLICATE_ROUTE` | Mesma rota registrada mais de uma vez | Renomeie o path ou o método HTTP da rota duplicada. |
