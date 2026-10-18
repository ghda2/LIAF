# Issue #026: Interação Básica com o Sistema

**Status:** Concluída
**Componente:** `pkg/builtins`, `pkg/checker`, `pkg/runtime`
**Data:** 24 de setembro de 2026 (concluída em 26 de setembro de 2026)
**Depende de:** #019
**Casos de aceitação:** `conformance/basics/026_*.liaf`

---

## 1. Contexto
Não há relógio, número aleatório, variável de ambiente, leitura do stdin nem forma de encerrar com
código de saída. Sem isso não dá para escrever nem uma CLI simples.

## 2. Proposta
| Forma | Tipo | Efeito |
|---|---|---|
| `(now-ms)` | `-> int`, epoch em milissegundos | `clock` (já existe) |
| `(rand-int n)` | `-> int` em `[0, n)`; `n <= 0` é erro (ver #022) | `rand` (novo) |
| `(env-get nome)` | `-> (result str str)` | `env` (novo) |
| `(read-line)` | `-> (result str str)`, sem o `\n`; EOF é `err` | `io` |
| `(exit código)` | `-> void`, não retorna | `io` |

### Decisão: efeitos novos
Adotada a **Opção A (efeitos próprios `rand` e `env`)**. Mantém a auditoria fina de determinismo e efeitos colaterais.
Decisão de `exit` em rotas: `exit` encerra o processo imediatamente (com o código fornecido), já que declara o efeito `io`.

## 3. Critérios de conclusão
- [x] Decisão sobre os efeitos registrada; `E_UNUSED_EFFECT` e o erro de efeito ausente funcionando para os novos efeitos.
- [x] Todos os casos `026_*` com `status: done`.
- [x] `exit` dentro de uma rota HTTP: definido que encerra o processo/servidor sob efeito `io`.
