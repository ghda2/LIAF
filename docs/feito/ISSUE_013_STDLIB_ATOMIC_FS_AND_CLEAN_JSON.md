# Issue #013: I/O Atômico no Filesystem, Arrays JSON Limpos e Validação Estrita de Efeitos

**Status:** Implementada em 16/09/2026 — ver "Estado da implementação" no fim  
**Componente:** `std/fs`, `std/json`, `pkg/checker`, `runtime/`  
**Data:** 16 de setembro de 2026  

---

## 1. Contexto e Motivação
A auditoria prática da API de referência (`task_api.liaf`) revelou gargalos críticos na biblioteca padrão e no sistema de efeitos que afetam a robustez de aplicações escritas por IAs:

1. **Gravação Não-Atômica no FS:** `fs-write-file` direto sobreescreve o arquivo in-place. Se o processo morrer durante a escrita, o arquivo corrompe e a aplicação entra em falha irrecuperável. Falta suporte a gravação atômica (`fs-rename` ou `fs-write-atomic`).
2. **Duplo Caminho de Falha em `fs-write-file`:** Atualmente retorna `(result bool str)`. Isso gera dois caminhos de erro: `err message` e `ok false`. Para uma IA, isso causa bugs frequentes por esquecimento de um dos caminhos. O correto é `(result void str)`.
3. **Envelope de Coleções JSON (`{"Items":[...]}`):** O `json-encode` de listas embute o wrapper interno da struct Go/C, obrigando programas a montarem arrays JSON serializados manualmente com concatenação de strings.
4. **Efeitos Declarados e Não Usados ("Efeitos Mortos"):** O checker aceita que uma função declare `effects io` mesmo sendo puramente matemática, enfraquecendo o valor documental e a auditoria dos contratos para agentes de IA.

---

## 2. Proposta Técnica

### 2.1. Primitivas de I/O Atômico
- Adicionar builtin `(fs-rename old-path new-path)` retornando `(result void str)`.
- Adicionar builtin conveniente de alto nível `(fs-write-atomic path content)` que grava em `.tmp` e renomeia atomicamente.
- Simplificar retorno de `fs-write-file` para `(result void str)`.

### 2.2. Arrays JSON Nativos
- Corrigir codificador JSON para serializar tipos `(list T)` diretamente como `[...]` e desserializar diretamente sem exigir struct intermediária.

### 2.3. Verificação Estrita de Efeitos
- O `pkg/checker` deve reportar warning ou erro se um efeito declarado em `(effects ...)` não for exercido por nenhuma expressão no corpo da função.

---

## 3. Tarefas de Implementação
- [ ] Alterar assinatura de `fs-write-file` para `(result void str)`.
- [ ] Implementar builtin `fs-rename` no backend Go e no backend C.
- [ ] Corrigir `json-encode` e `json-decode` para suportar listas e coleções como raiz JSON.
- [ ] Adicionar checagem de "efeitos não utilizados" no `pkg/checker`.
- [ ] Refatorar `task_api.liaf` para usar gravação atômica e serialização limpa de listas.

---

Contrato atual e especificações: [docs/SPEC.md](../linguagem/SPEC.md). Evidências consolidadas: [STATUS.md](../STATUS.md).

---

## Estado da implementação (16/09/2026)

Os quatro pontos da motivação foram resolvidos:

1. **Gravação atômica:** `fs-write-atomic` grava em `<arquivo>.tmp.<pid>` e renomeia sobre o
   destino, removendo o temporário se o rename falhar. `fs-rename` expõe o rename direto.
   Implementação em `pkg/runtime/runtime.go`.
2. **Duplo caminho de falha:** `fs-write-file`, `fs-rename` e `fs-write-atomic` passaram a
   retornar `(result void str)`. Tentar usar o valor de `ok` agora é erro de tipo, o que fecha
   o caminho `ok false`. **`fs-remove` também foi convertido** numa segunda passada (o RFC da v0.3,
   seção 3.1, o especifica como `void`); `examples/fs_json.liaf` e a saída esperada em
   `integration_test.go` foram ajustados.
3. **Envelope de coleções:** `List[T].MarshalJSON` devolve `[...]` — e `[]` para lista vazia,
   não `null`. Coberto por `TestExecuteV03TaskAPI/listas_sao_arrays_JSON_nativos`, que falha se
   a resposta contiver `"Items"`.
4. **Efeitos mortos:** `E_UNUSED_EFFECT` acusa efeito declarado que o corpo nunca consome,
   em funções e em rotas. Efeito transitivo de função chamada conta como uso.
5. **`json-encode` e `json-decode` passaram a ser puras** (RFC seção 3.3). Remover o efeito `io`
   cascateia com o item 4: toda função que declarava `io` apenas para serializar passa a acusar
   efeito morto. A convergência é um ponto fixo — remover `io` de uma função mata o `io` de quem a
   chama. Nos exemplos foram duas rodadas, mais as cinco rotas de `task_api_v03.liaf`. O resultado é
   uma assinatura honesta: as rotas da API declaram `fs`, não `fs io`.
6. **`json-decode` aceita tipos compostos** (RFC seção 3.2): `(json-decode texto (list Task))`.
   A conversão de expressão para tipo virou `ast.TypeFromExpr`, compartilhada por checker e codegen,
   que antes tinham cada um a sua versão limitada a `IdentExpr`.

**Quebra de compatibilidade registrada:** a mudança para `(result void str)` invalidou
`(if written ...)` em `task_api.liaf`, que testava um valor que era sempre `true` — código morto
que a issue existia para eliminar. `benchmarks/http-context.md` foi corrigido: ele ainda instruía
modelos a montar arrays JSON com `concat`, o que agora produz saída errada.

## Bug encontrado ao fechar as lacunas

`(try ...)` dentro de uma função `void` com bloco `(on-err ...)` gerava
`return _liaf_on_err(...)`. O handler de uma função `void` não devolve valor, então o Go emitido não
compilava (`used as value`). A chamada e o `return` precisam ficar separados.

Não tinha sido detectado porque todas as funções de `task_api_v03.liaf` retornam `Response`.
Regressão coberta por `TestExecuteV03OnErrInVoidFunction`.
