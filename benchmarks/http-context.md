# Recursos HTTP e persistência implementados na LIAF v0.3

Use S-expressions, como no contrato principal. Esta referência descreve APIs disponíveis, não uma implementação da tarefa.

> **Atenção ao comparar medições:** este contexto descreve a v0.3. Resultados
> gravados em `results/` antes de 16/09/2026 usaram o contexto v0.2, em que
> `call` era obrigatório, não havia `try`/`on-err` nem `route`, e `json-encode`
> de `(list T)` produzia `{"Items":[...]}`. Rodadas feitas com contextos
> diferentes não são comparáveis entre si.

## Ergonomia da v0.3

- **Chamadas diretas:** escreva `(json-encode x)`, não `(call json-encode x)`. A forma com `call` ainda é aceita, mas não é a canônica.
- **`(try expr)`** desempacota um `(result T E)`: devolve `T` em caso de sucesso e, em caso de erro, sai da função imediatamente. Só pode ser usado em função/rota que tenha um bloco `(on-err var ...)` ou que retorne `(result ...)`.
- **`(on-err var (instruções...))`** vem antes de `(body ...)` e recebe a mensagem de erro em `var`. É o destino de qualquer `try` que falhar, o que elimina a pirâmide de `match` aninhado.
- **Rotas declarativas:** `(route MÉTODO "/caminho" (params ...) (returns Response) (effects ...) [(on-err var ...)] (body ...))`. Cada `{nome}` no caminho precisa de um parâmetro de mesmo nome, do tipo `int` ou `str`, e chega já convertido. Um parâmetro de struct recebe o corpo JSON já desserializado; corpo inválido vira `400` automaticamente, sem código do usuário. Rotas se registram sozinhas — não chame `http-get` para elas.

## APIs

- `(args)` retorna `(list str)` com efeito `io`, **sem o nome do executável**. O primeiro argumento está no índice 0 e `(list-get argumentos 0)` retorna `(result str str)`.
- Um handler escrito como função tem `(params (req Request)) (returns Response)` e declara seus efeitos. `(request-body req)` e `(request-path req)` retornam `str` sem efeitos. Não existe `request-param`. Com `route`, nada disso é necessário.
- Registro manual de handlers por nome continua disponível: `(do (http-get "/health" health))`, com `http-get`, `http-post`, `http-put`, `http-delete`, todos com efeito `net` mais os efeitos do handler.
- `(str-len texto)` retorna `int`; `(str-slice texto início fim)` retorna `(result str str)`, com início inclusivo e fim exclusivo em bytes. `(int-from-str texto)` retorna `(result int str)`.
- `(json-response status texto-json)` retorna `Response`, sem efeito. O segundo argumento é JSON já serializado. `(json-encode valor)` retorna `(result str str)`, com efeito `io`. Campos de structs mantêm seus nomes no JSON.
- `(json-decode texto NomeDaStruct)` retorna `(result NomeDaStruct str)`, com efeito `io`. Campos ausentes recebem valores zero; tipos incompatíveis ou JSON malformado retornam erro. Valide os campos depois de decodificar.
- `(serve-hybrid "./public" porta)` inicia e bloqueia o servidor, retorna `(result bool str)` e exige `fs net`. A pasta deve existir (o benchmark a cria). **Não existe `http-serve`**.
- `fs-exists` retorna `bool`. `fs-read-file` retorna `(result str str)`. `fs-write-file`, `fs-rename` e `fs-write-atomic` retornam `(result void str)` — o sucesso não carrega valor, então não teste o conteúdo de `ok`. `fs-remove` retorna `(result bool str)`. Todos exigem `fs`.
- **`(fs-write-atomic caminho texto)`** grava num arquivo temporário e renomeia. Use para estado que não pode ficar pela metade se o processo morrer no meio da escrita.
- Não existem variáveis globais nem closures de handlers: o estado pode ser carregado de arquivo por funções auxiliares a cada requisição.
- Construa structs com `(new Tipo campos-na-ordem...)` e leia campos com `(field objeto campo)`. Para atualizar uma struct, construa uma nova; não há `set-field`.
- Listas: `(make-list Tipo)`, `(list-push lista item)` retorna `void`, `(list-len lista)` retorna `int`, `(list-get lista índice)` retorna Result. `(for-each item lista instruções...)` itera a lista. Não há `list-remove`; uma lista filtrada pode ser construída com `make-list` e `list-push`.
- **Serialização de listas:** `json-encode` de `(list T)` produz um array JSON nativo `[...]`. Devolva a lista diretamente; não componha colchetes e vírgulas à mão.
- Conversão explícita: `(str-from-int numero)` retorna `str`. `concat` aceita escalares. `println` aceita números diretamente. Declare `(effects)` para funções puras, nunca `(effects none)`.
- **Efeitos declarados e não usados são erro** (`E_UNUSED_EFFECT`). Declare exatamente os efeitos que o corpo consome, direta ou transitivamente.
- `(match resultado (ok valor instruções...) (err mensagem instruções...))` é uma instrução; prefira `try`/`on-err` quando só quiser propagar o erro.

Exemplo mínimo de inicialização (porta ilustrativa; leia o argumento no programa solicitado):

```liaf
(module http-example
  (struct Item (fields (id int) (nome str)))

  (route GET "/health" (params) (returns Response) (effects)
    (body (return (json-response 200 "{\"status\":\"ok\"}"))))

  (route GET "/itens/{id}" (params (id int)) (returns Response) (effects fs io)
    (on-err message (return (json-response 500 "{\"error\":\"falha\"}")))
    (body
      (let texto str (try (fs-read-file "itens.json")))
      (return (json-response 200 texto))))

  (fn main (params) (returns void) (effects fs io net)
    (body
      (match (serve-hybrid "./public" "8080")
        (ok stopped (do (println "stopped")))
        (err message (do (println message)))))))
```
