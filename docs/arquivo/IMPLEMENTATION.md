# Contrato implementado da LIAF

Data: 2026-09-16. Este arquivo descreve o código atual. A especificação completa está em [docs/SPEC.md](../linguagem/SPEC.md) — a seção 18 cobre a v0.3 e a seção 19 cobre a v0.4. [SPEC.md](SPEC_v0.1.md) neste diretório é a v0.1 histórica.

## Linguagem e validação

- A CLI aceita a sintaxe v0.4, que é a v0.3 mais banco de dados e WebSocket. Programas v0.2 e v0.3 continuam válidos.
- **Chamadas diretas:** `(json-encode x)`. A forma `(call json-encode x)` continua aceita.
- **`(try expr)`** desempacota `(result T E)` e, no erro, sai da função. Exige um bloco `(on-err var ...)` — que vem antes de `(body ...)` e recebe a mensagem — ou uma função que retorne `(result ...)`. Sem isso: `E_UNHANDLED_RESULT`.
- **`(route MÉTODO "/caminho" (params ...) (returns T) (effects ...) [(on-err var ...)] (body ...))`** é declaração de topo e se registra sozinha. Cada `{nome}` no caminho exige parâmetro homônimo `int` ou `str`, extraído da posição correta do padrão; parâmetro struct recebe o corpo JSON desserializado, e corpo inválido vira `400`. Violações: `E_ROUTE_PARAM`.
- **`(ws-route "/caminho" (params ...) (effects ...) [(on-err var ...)] [(on-open ...)] (on-message var ...) [(on-close ...)])`** é declaração de topo e se registra sozinha. Sem `(returns ...)` e sem `(body ...)`: os três blocos são o corpo. Exige exatamente um parâmetro `WSConn`; os demais vêm de `{nome}` no caminho e são `int` ou `str`. Violações: `E_WS_PARAM`. `(effects ...)` tem de incluir `net`, que conta como usado sempre.
- **`(db-transaction conn ...)`** confirma ao fim do bloco e desfaz em qualquer saída antecipada. Dentro do bloco, `conn` designa a transação. Exige `(on-err ...)` ou retorno `(result ...)`, como `try`.
- Efeito declarado que o corpo nunca consome, direta ou transitivamente, é `E_UNUSED_EFFECT`.
- `(module ...)`, `(fn ... (params ...) (returns ...) (effects ...) (body ...))` permanecem a forma das declarações.
- `check`, `emit`, `run` e `build` verificam tipos, aridade, símbolos, retornos, canais e efeitos antes de gerar Go.
- Loops: `(while condição instruções...)`, `(for-range nome início fim instruções...)` (fim exclusivo, limites avaliados uma vez), `(for-each nome lista instruções...)`, `(break)` e `(continue)`.
- Tipos: `int`, `float`, `str`, `bool`, `void` (retorno), structs, `(chan T)`, `(list T)`, `(map K V)`, `(result T E)` e os opacos `Request`, `Response`, `DBConnection` e `WSConn`. Opaco significa sem campos: `(field conn host)` é `E_UNKNOWN_FIELD`.
- Efeitos: `io`, `net`, `fs`, `clock`, `spawn` e `db`. `db` é distinto de `net` porque promete estado externo compartilhado, não só tráfego de rede.
- Listas têm semântica de referência: `list-push` dentro de uma função modifica a lista do chamador. Mapas têm chaves escalares. Compartilhamento mutável entre goroutines exige sincronização; canais não garantem ausência geral de data races.
- `list-get`, `list-set` e `map-get` retornam `Result` para índices/chaves inválidos.
- Resultados são tratados com `(match expressão (ok nome instruções...) (err nome instruções...))`, com `try`, ou retornados. Construtores: `(ok valor)` / `(err erro)`, em contexto com tipo explícito.
- Structs: `(new Nome campos-na-ordem...)`; leitura: `(field objeto campo)`. Não há `set-field`: construa uma nova struct.

## Biblioteca

| Operação | Retorno | Efeito |
|---|---|---|
| `print valores...` / `println valores...` | `void` | `io` |
| `concat valores...` | `str` | nenhum |
| `str-from-int número` | `str` | nenhum |
| `fs-read-file caminho` | `(result str str)` | `fs` |
| `fs-write-file caminho texto` | `(result void str)` | `fs` |
| `fs-rename origem destino` | `(result void str)` | `fs` |
| `fs-write-atomic caminho texto` | `(result void str)` | `fs` |
| `fs-remove caminho` | `(result void str)` | `fs` |
| `fs-exists caminho` | `bool` | `fs` |
| `json-encode valor` | `(result str str)` | nenhum |
| `json-decode texto Tipo` | `(result Tipo str)` | nenhum |
| `int-from-str texto` | `(result int str)` | nenhum |
| `http-get/http-post/http-put/http-delete rota handler` | `void` | `net` + efeitos do handler |
| `json-response status corpo` | `Response` | nenhum |
| `request-body/request-path/request-method requisição` | `str` | nenhum |
| `serve-hybrid pasta porta` | `(result bool str)` | `fs net` |
| `db-connect "driver" dsn` | `(result DBConnection str)` | `db` |
| `db-close conexão` | `(result void str)` | `db` |
| `db-query conexão "SQL" TipoLinha arg...` | `(result (list TipoLinha) str)` | `db` |
| `db-exec conexão "SQL" arg...` | `(result int str)` | `db` |
| `redis-get conexão chave` | `(result str str)` | `db` |
| `redis-set conexão chave valor ttl` | `(result void str)` | `db` |
| `ws-send conexão mensagem` | `(result void str)` | `net` |
| `ws-send-json conexão valor` | `(result void str)` | `net` |
| `ws-close conexão código motivo` | `(result void str)` | `net` |
| `ws-join conexão tópico` / `ws-leave conexão tópico` | `void` | `net` |
| `ws-broadcast tópico mensagem` | `(result int str)` | `net` |
| `ws-topic-size tópico` | `int` | `net` |

As quatro escritas (`fs-write-file`, `fs-rename`, `fs-write-atomic`, `fs-remove`) retornam `(result void str)`: `ok` já significa que a operação terminou e não carrega valor, então não existe o caminho `ok false`. `json-encode` e `json-decode` são puras — operam sobre strings em memória e não declaram efeito. `json-decode` aceita tipos compostos, como `(json-decode texto (list Task))`. `fs-write-atomic` grava num temporário e renomeia sobre o destino. `json-encode` de `(list T)` produz array JSON nativo `[...]`, não `{"Items":[...]}`.

`http-get` registra uma rota; não é um cliente HTTP. O handler recebe `Request` e retorna `Response`. Corpos HTTP têm limite de 1 MiB. O autor de `json-response` fornece JSON válido; use `json-encode` para conteúdo variável.

Efeitos são transitivos inclusive em funções despachadas por `spawn`. `serve-site` mantém a API legada de retorno `void` e exige `fs net`; `serve-hybrid` permite tratar falhas de inicialização.

## Banco de dados e WebSocket

Os protocolos do PostgreSQL (wire v3), do MySQL (protocolo 41) e do Redis (RESP2) vivem em `pkg/dbdrv`, sobre TCP puro; o enquadramento RFC 6455 vive em `pkg/web/websocket.go`. Não há dependência externa nova: `go.mod` continua só com `golang.org/x/*`, e o binário gerado não carrega driver nem biblioteca de WebSocket.

- **O SQL de `db-query` e `db-exec` tem de ser literal de string.** Variável ou `(concat ...)` é `E_SQL_INTERPOLATION`. Os valores viajam fora do texto do comando — mensagem `Bind` no PostgreSQL, `COM_STMT_EXECUTE` no MySQL — e não por escape. Não há escapatória; montar SQL dinamicamente não é expressável.
- O checker confere a contagem de marcadores contra a de argumentos (`E_SQL_PARAM_COUNT`), ignorando o que está dentro de literais e comentários, e abandona a contagem quando aparecem `?|`, `?&` ou `??` (operadores JSONB, não marcadores).
- Driver é literal conferido em compilação: `postgres`, `mysql` ou `redis`. `E_UNKNOWN_DRIVER` caso contrário.
- Autenticação suportada: PostgreSQL cleartext, MD5 e SCRAM-SHA-256 (com verificação da assinatura do servidor); MySQL `mysql_native_password` e `caching_sha2_password`, incluindo o caminho lento com RSA; Redis `AUTH` de um e de dois argumentos.
- `db-query` exige tipo de linha struct e casa coluna com campo pelo nome, tratando `snake_case` e `kebab-case` como equivalentes. O resultado é materializado inteiro; não há cursor.
- Pool com teto de 16 conexões simultâneas, ajustável por `?pool_max=N` no DSN. Conexão cuja falha foi de transporte não volta ao pool; erro do banco (sintaxe, restrição) não invalida a sessão.
- `db-transaction` fixa uma conexão e agenda o rollback com `defer`, que é o que cobre a saída antecipada de um `try` que falhou. Depois do commit o rollback é no-op. Transações aninhadas são recusadas em execução.
- WebSocket: mensagem de até 1 MiB, ping a cada 30 s, ocioso derrubado em 90 s, máscara exigida em todo frame do cliente. Fragmentação na leitura é remontada; na escrita não é usada. O pub/sub é um mapa em memória por processo — não substitui broker com mais de um nó. Fechar a conexão desinscreve de todos os tópicos.
- O backend C não implementa nada disso: `ws-route` é erro explícito e os builtins caem em "unsupported operation".

## Web e deploy

- SSG: Markdown/frontmatter, layouts, includes, coleções, loops, fingerprint, gzip, ETag, URLs limpas e `--embed`.
- Publicação e reload exigem POST e token. Sem `LIAF_DEPLOY_TOKEN`/`DeployToken`, administração fica desabilitada (503).
- Publish limita o corpo (`MaxPublishSize`, padrão 32 MiB), grava arquivo temporário, renomeia, reconstrói cache e troca o ponteiro. Escritas/reloads são serializados. Falha de recarga tenta restaurar o arquivo anterior.
- Sites embutidos são imutáveis: publish retorna 409. Use diretório em disco para publicação dinâmica.
- `MaxRAMAssetSize` é configurável (padrão 512 KiB). Mídias pesadas e arquivos binários grandes usam disco; textos de templates permanecem em RAM. Streaming suporta Range/206/416. Não há LRU adicional: o sistema operacional administra seu cache de páginas.
- `liafc service install --dry-run` produz unidade systemd inspecionável; instalação real exige Linux e permissões adequadas. Token pode ser fornecido em `/etc/liaf/NOME.env`.
- `liafc deploy caddy-bind` usa a Admin API com proteção de versão via ETag, preservando outras rotas.

## Validação e limites

Execute `go test ./...`. Os testes de integração compilam e executam exemplos em diretórios temporários e verificam resultados observáveis.

Métricas antigas de produção nos logs são registros históricos, não garantias de RAM ou latência para qualquer carga. Benchmark entre modelos precisa de execuções reais e dados brutos; metas não são resultados.

O backend Go exige Go e os pacotes do projeto na compilação. Os binários gerados não precisam do compilador Go em execução. Backend autônomo e self-hosting estão nas issues #010 e #011; não estão implicitamente concluídos pela existência de uma AST v0.2.

---

## Guia de Sintaxe Concreta e Modelos LIAF v0.2

Todo programa `.liaf` v0.2 DEVE seguir rigorosamente esta estrutura S-expression:

### 1. Estrutura Básica do Programa
```liaf
(module meu-modulo
  (fn main (params) (returns void) (effects io)
    (body
      (do (call println "Ola Mundo")))))
```
- Efeitos permitidos: `(effects)` (vazio para funções puras — NUNCA use `(effects none)`), ou `(effects io)`, `(effects fs io)`, `(effects net)`.
- Se a função tiver parâmetros: `(params (item int) (nome str))`

### 2. Declaração de Variáveis e Modificação
- Declaração com tipo obrigatório: `(let <nome> <tipo> <expressão>)`
  Exemplo: `(let sum int 0)`
- Modificação de variável existente: `(set <nome> <expressão>)`
  Exemplo: `(set sum (add sum i))`

### 3. Chamadas de Função (Instruções vs Expressões)
- Toda chamada executada por efeito colateral no corpo da função DEVE ser envolvida em `(do ...)`:
  `(do (call println item))`
  `(do (call list-push itens 42))`
- Chamadas retornando valores usados em expressões não usam `do`:
  `(let texto str (call str-from-int numero))`
- `json-encode` retorna `(result str str)`, não `str`: trate o resultado com `match`, como no exemplo de JSON abaixo.

#### Impressão, conversão e concatenação

`print` e `println` aceitam diretamente valores escalares (`int`, `float`, `str`, `bool`). `println` acrescenta uma quebra de linha; não é necessário converter números para imprimi-los. Valores compostos exigem serialização explícita.

Para converter um inteiro em texto, a função é `str-from-int`. Os nomes `int-to-str`, `to-str` e `string-from-int` não existem na biblioteca implementada.

`concat` é uma função e exige `call`: `(call concat "valor=" numero)`. A forma `(concat "valor=" numero)` é inválida. Operadores como `add` e `eq` são expressões primitivas e usam a sintaxe da próxima seção.

```liaf
(module impressao
  (fn main (params) (returns void) (effects io)
    (body
      (let numero int 7)
      (do (call println numero))
      (let texto str (call str-from-int numero))
      (do (call println texto))
      (do (call println (call concat "valor=" numero))))))
```

Saída: `7`, `7` e `valor=7`, cada um em sua própria linha.

### 4. Operações Binárias (Expressões Prefixadas)
- Aritmética: `(add a b)`, `(sub a b)`, `(mul a b)`, `(div a b)`
- Comparação: `(eq a b)`, `(neq a b)`, `(lt a b)`, `(gt a b)`, `(lte a b)`, `(gte a b)`
- Lógica: `(and a b)`, `(or a b)`

### 5. Controle de Fluxo
- **Condicional If**:
  `(if (eq i 3) (then (continue)))`
  `(if (lt x 10) (then (set x (add x 1))) (else (set x 0)))`
- **Loop For-Range** (início inclusivo, fim exclusivo):
  `(for-range i 0 10 (if (eq i 7) (then (break))) (set sum (add sum i)))`
- **Loop For-Each**:
  `(for-each item itens (do (call println item)))`
- **Loop While**:
  `(while (lt count 10) (set count (add count 1)))`

### 6. Structs
- Definição no topo do módulo:
  `(struct User (fields (name str) (score int)))`
- Construtor: `(call new User "alice" 42)`
- Leitura de campo: `(call field u name)`

### 7. Listas e Mapas
- Criar lista: `(let items (list int) (call make-list int))`
- Adicionar à lista: `(do (call list-push items 42))`
- Ler lista (retorna Result):
  ```liaf
  (match (call list-get items 0)
    (ok val (do (call println val)))
    (err msg (do (call println "missing"))))
  ```
- Criar mapa: `(let scores (map str int) (call make-map str int))`
- Inserir no mapa: `(do (call map-set scores "alice" 100))`
- Ler do mapa:
  ```liaf
  (match (call map-get scores "alice")
    (ok score (do (call println score)))
    (err msg (do (call println "missing"))))
  ```

### 8. JSON e Sistema de Arquivos
```liaf
(match (call json-encode user)
  (ok json-str
    (match (call fs-write-file "data.json" json-str)
      (ok written
        (match (call fs-read-file "data.json")
          (ok text
            (match (call json-decode text User)
              (ok restored (do (call println (call field restored name))))
              (err err3 (do (call println err3)))))
          (err err2 (do (call println err2)))))
      (err err1 (do (call println err1)))))
  (err err0 (do (call println err0))))
```
