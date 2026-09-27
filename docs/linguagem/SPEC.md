# LIAF — Especificação para agentes de IA (núcleo v0.2, sintaxe corrente v0.4)

**Status:** especificação de design parcialmente implementada; consulte o [guia de agentes](AI_GUIDE.md) para o contrato executável atual
**Compatibilidade:** incompatível com a sintaxe LIAF v0.1  
**Público-alvo:** modelos de linguagem e agentes autônomos  
**Objetivo:** permitir que modelos menores e de menor custo produzam software correto, minimizando o esforço de inferência e o custo total até uma solução validada

> O compilador atual aceita a sintaxe v0.3, que é a v0.2 mais três adições retrocompatíveis
> descritas na [seção 18](#18-adições-da-v03): chamadas diretas sem `call`, `try`/`on-err` e rotas
> declarativas. Todo programa v0.2 continua válido. As seções 1 a 17 descrevem o núcleo v0.2 e
> permanecem corretas; onde a seção 18 diverge, ela é a autoridade.
>
> Recursos propostos neste documento, como o protocolo completo de patches estruturais, não devem
> ser considerados implementados sem confirmação no guia de agentes e em testes. A sintaxe v0.1 é
> histórica.

---

## 1. Objetivos de projeto

A LIAF v0.2 é uma representação textual canônica de uma AST. Ela não tenta imitar uma linguagem escrita por humanos. Seu design prioriza:

1. **Uniformidade estrutural:** toda construção é uma lista delimitada por `(` e `)`.
2. **Semântica explícita:** o primeiro elemento de cada lista identifica a operação ou espécie do nó.
3. **Interpretação única:** um programa válido possui uma única forma canônica.
4. **Validação total:** nenhum token inválido, argumento excedente ou construção desconhecida pode ser ignorado.
5. **Semântica local:** o significado de um nó depende do mínimo possível de contexto distante.
6. **Tipos e efeitos explícitos:** assinaturas revelam valores, erros e capacidades externas.
7. **Correção mecânica:** diagnósticos identificam intervalos e caminhos da AST e oferecem patches aplicáveis.
8. **Economia global:** otimiza-se o custo até o programa correto, e não apenas o tamanho da primeira resposta.

Não são objetivos:

- legibilidade ou conveniência para humanos;
- compatibilidade sintática com a v0.1;
- aceitar múltiplos estilos equivalentes;
- inferir silenciosamente a intenção do autor;
- recuperar entrada inválida descartando tokens.

### 1.1 Problema que a linguagem pretende resolver

Modelos avançados conseguem operar linguagens convencionais apesar de suas exceções, ambiguidades e múltiplas formas equivalentes. Modelos menores tendem a perder mais desempenho quando precisam manter regras distantes, interpretar mensagens voltadas a humanos ou reconstruir grandes trechos para corrigir um erro local.

A LIAF v0.2 pretende transferir parte desse trabalho cognitivo do modelo para ferramentas determinísticas:

```text
objetivo
  -> modelo gera AST textual
  -> parser valida a estrutura
  -> checker valida tipos, efeitos e contratos
  -> compilador emite diagnósticos estruturados
  -> modelo aplica um patch localizado
  -> testes verificam o comportamento
```

O modelo não precisa acertar tudo em uma única geração. Ele precisa conseguir convergir de forma econômica por meio de decisões pequenas, explícitas e verificáveis.

### 1.2 Hipótese central

> Uma linguagem uniforme, restrita na forma e rigorosamente verificável reduz a inteligência mínima necessária para produzir software correto.

Essa afirmação é uma hipótese mensurável, não uma propriedade presumida da sintaxe. A v0.2 somente será considerada bem-sucedida se modelos menores alcançarem resultados próximos aos de modelos avançados, com custo monetário total inferior.

### 1.3 Restrição sintática não é restrição computacional

A LIAF diferencia:

- **poder de expressão:** quais programas podem ser representados;
- **complexidade de expressão:** quanto raciocínio é necessário para representá-los;
- **liberdade sintática:** quantas formas diferentes existem para expressar o mesmo programa.

A v0.2 reduz deliberadamente a liberdade sintática. Isso não deve reduzir o poder de expressão. Deve existir uma forma canônica para cada operação, mas as operações precisam ser combináveis o suficiente para construir sistemas gerais.

Uma linguagem com poucas regras pode continuar sendo de propósito geral. A simplicidade das S-expressions afeta a representação da árvore, não o conjunto de algoritmos que a árvore pode expressar.

### 1.4 Limites aceitáveis e limites indevidos

São restrições desejáveis:

- uma única notação para chamadas;
- uma única forma de tratar cada categoria de erro;
- ausência de coerções implícitas;
- ausência de sobrecargas ambíguas;
- efeitos externos declarados;
- APIs determinísticas e contratos verificáveis.

São limitações indevidas para uma linguagem de propósito geral:

- ausência de iteração ou recursão;
- ausência de coleções e estruturas de dados extensíveis;
- impossibilidade de compor funções;
- inexistência de módulos;
- impossibilidade de acessar rede, arquivos, processos e memória de maneira controlada;
- dependência de um builtin de alto nível para cada aplicação possível;
- impossibilidade de interoperar com bibliotecas ou runtimes externos.

O núcleo inicial pode implementar esses recursos progressivamente, mas a arquitetura não deve impedir sua inclusão.

---

## 2. Forma léxica

### 2.1 Codificação

- Todo arquivo usa UTF-8 sem BOM.
- Identificadores e palavras reservadas usam ASCII na versão inicial.
- Quebras de linha aceitas na entrada são normalizadas para `LF` pela representação canônica.
- Espaços e quebras de linha separam tokens, mas não carregam semântica.

### 2.2 Tokens

Existem apenas estas categorias léxicas:

```ebnf
LParen      ::= "("
RParen      ::= ")"
Identifier  ::= Letter { Letter | Digit | "-" | "_" }
Integer     ::= [ "-" ] Digit { Digit }
Float       ::= [ "-" ] Digit { Digit } "." Digit { Digit }
Boolean     ::= "true" | "false"
String      ::= '"' { Character | Escape } '"'
Comment     ::= ";" { CharacterExceptNewline }
```

Escapes válidos em strings:

```text
\"  \\  \n  \r  \t  \uXXXX
```

Qualquer escape desconhecido, string não terminada ou número malformado é erro léxico. Comentários são removidos na forma canônica.

### 2.3 Identificadores

Identificadores são sensíveis a maiúsculas e minúsculas. A forma canônica recomenda `kebab-case`:

```liaf
worker
request-handler
result-1
```

Palavras reservadas não podem ser usadas como identificadores.

---

## 3. Regra estrutural universal

Todo nó composto possui a forma:

```liaf
(kind argument-1 argument-2 ... argument-n)
```

Não existem:

- colchetes sintáticos;
- chaves;
- tags de fechamento;
- operadores infixos;
- precedência de operadores;
- ponto e vírgula terminador;
- indentação semântica;
- argumentos opcionais posicionais.

O fechamento `)` encerra sempre o nó composto aberto mais próximo. O nome e a espécie do nó não são repetidos no fechamento.

---

## 4. Gramática do núcleo

```ebnf
Program       ::= Module
Module        ::= "(" "module" ModuleName { TopLevel } ")"
ModuleName    ::= Identifier

TopLevel      ::= Import | Struct | Function
Import        ::= "(" "import" String ")"

Struct        ::= "(" "struct" Identifier Fields ")"
Fields        ::= "(" "fields" { Field } ")"
Field         ::= "(" Identifier Type ")"

Function      ::= "(" "fn" Identifier Params Returns Effects Body ")"
Params        ::= "(" "params" { Param } ")"
Param         ::= "(" Identifier Type ")"
Returns       ::= "(" "returns" Type ")"
Effects       ::= "(" "effects" { Effect } ")"
Body          ::= "(" "body" { Statement } ")"

Statement     ::= Let | Set | Return | If | Spawn | Send | ExprStatement
Let           ::= "(" "let" Identifier Type Expression ")"
Set           ::= "(" "set" Identifier Expression ")"
Return        ::= "(" "return" [ Expression ] ")"
If            ::= "(" "if" Expression Then [ Else ] ")"
Then          ::= "(" "then" { Statement } ")"
Else          ::= "(" "else" { Statement } ")"
Spawn         ::= "(" "spawn" Call ")"
Send          ::= "(" "send" Expression Expression ")"
ExprStatement ::= "(" "do" Expression ")"

Expression    ::= Literal | Identifier | Call | Operator | Receive
Call          ::= "(" "call" Identifier { Expression } ")"
Operator      ::= "(" OperatorName Expression Expression ")"
Receive       ::= "(" "recv" Expression ")"
Literal       ::= Integer | Float | Boolean | String

Type          ::= PrimitiveType | NamedType | AppliedType
PrimitiveType ::= "int" | "float" | "str" | "bool" | "void"
NamedType     ::= Identifier
AppliedType   ::= "(" TypeConstructor Type { Type } ")"
TypeConstructor ::= "chan" | "list" | "map" | "option" | "result"

Effect        ::= "io" | "net" | "fs" | "clock" | "spawn"
OperatorName  ::= "add" | "sub" | "mul" | "div"
                | "eq" | "neq" | "gt" | "lt" | "gte" | "lte"
                | "and" | "or"
```

Aridades são parte da gramática. Por exemplo, `add` recebe exatamente dois operandos. Operações variádicas devem possuir nomes próprios ou receber uma coleção explícita.

---

## 5. Forma canônica

O formatador oficial deve produzir exatamente uma representação para cada AST:

- dois espaços por nível;
- um espaço entre tokens na mesma linha;
- nenhuma linha em branco dentro de um nó;
- strings com escapes mínimos e determinísticos;
- números sem zeros redundantes;
- nenhum comentário;
- ordem original preservada onde a ordem possui semântica;
- ordem lexicográfica onde a ordem não possui semântica;
- arquivo terminado por uma única quebra de linha.

Exemplo canônico:

```liaf
(module example
  (fn sum
    (params
      (a int)
      (b int))
    (returns int)
    (effects)
    (body
      (return (add a b))))
  (fn main
    (params)
    (returns void)
    (effects io)
    (body
      (do (call println (call sum 20 22))))))
```

Operadores do núcleo usam sintaxe prefixada. `add`, `mul`, `and` e `or` são variádicos (mínimo 2 argumentos, associatividade à esquerda): `(add 1 2 3)`, `(mul 2 3 4)`, `(and true true false)`. `sub` e `div` são estritamente binários.

Funções matemáticas do núcleo: `(mod a b)`, `(neg x)`, `(abs x)`, `(min a b)`, `(max a b)`, `(pow a b)`, `(sqrt x)`, `(floor x)`, `(ceil x)`, `(round x)`.

Literais numéricos suportam base 10, hexadecimal (`0xff`) e notação científica (`1e3`, `2.5e-3` como float).

---

## 6. Tipos

### 6.1 Tipos primitivos

```liaf
int
float
str
bool
void
```

### 6.2 Tipos compostos

Tipos compostos usam a mesma estrutura da linguagem:

```liaf
(chan str)
(list int)
(map str int)
(option User)
(result User AppError)
```

Não existe coerção implícita. Conversões devem ser expressas por funções do núcleo:

```liaf
(str-from-int count)
(float-from-int count)
(int-from-float f)
(str-from-float f)
(float-from-str "2.5")    ;; retorna (result float str)
(str-from-bool flag)
(bool-from-str "true")     ;; retorna (result bool str)
(int-from-str "123")       ;; retorna (result int str)
```

### 6.3 Funções

Parâmetros, retorno e efeitos são sempre declarados, mesmo quando vazios:

```liaf
(fn worker
  (params
    (channel (chan str))
    (id int))
  (returns void)
  (effects clock spawn)
  (body
    ...))
```

O verificador deve rejeitar:

- argumento ausente ou excedente;
- tipo de argumento incompatível;
- retorno incompatível;
- caminho sem retorno em função não `void`;
- chamada de símbolo inexistente;
- variável usada antes da declaração;
- redefinição no mesmo escopo;
- efeito utilizado sem declaração.

---

## 7. Controle de fluxo

Condicionais possuem ramos explicitamente nomeados:

```liaf
(if (gt total 0)
  (then
    (do (call println "positive")))
  (else
    (do (call println "not-positive"))))
```

O ramo `else` é opcional. `then` é obrigatório, inclusive quando vazio:

```liaf
(if condition
  (then))
```

Não há forma curta alternativa. Laços futuros deverão ser introduzidos como nós próprios, sem reutilizar sintaxe implícita.

---

## 8. Concorrência

Concorrência é explícita e tipada:

```liaf
(module concurrency-example
  (fn worker
    (params
      (channel (chan str))
      (id int))
    (returns void)
    (effects clock)
    (body
      (do (call sleep-ms 50))
      (send channel (call concat "worker-" (call str-from-int id)))))
  (fn main
    (params)
    (returns void)
    (effects io spawn)
    (body
      (let channel (chan str) (call make-chan str))
      (spawn (call worker channel 1))
      (spawn (call worker channel 2))
      (let first str (recv channel))
      (let second str (recv channel))
      (do (call println first))
      (do (call println second)))))
```

Regras:

- `spawn` aceita exatamente uma chamada que retorne `void`;
- a função que executa `spawn` declara o efeito `spawn`;
- `send` valida o tipo interno do canal;
- `recv` produz o tipo interno do canal;
- criação de canais é uma chamada de runtime explicitamente tipada;
- deadlocks não são inicialmente erros estáticos, mas podem gerar avisos de análise.

---

## 9. Efeitos e capacidades

Operações externas devem ser visíveis na assinatura. Efeitos iniciais:

| Efeito | Capacidade |
|---|---|
| `io` | entrada e saída de processo |
| `net` | acesso à rede |
| `fs` | leitura ou escrita no sistema de arquivos |
| `clock` | tempo, espera e temporizadores |
| `spawn` | criação de execução concorrente |

Exemplo:

```liaf
(fn fetch
  (params
    (url str))
  (returns (result str NetError))
  (effects net)
  (body
    (return (call http-get url))))
```

Uma função pode chamar outra somente se declarar todos os efeitos transitivos exigidos. A ordem canônica dos efeitos é lexicográfica.

---

## 10. Erros como valores

Falhas recuperáveis usam `result`, não strings especiais nem `panic` implícito:

```liaf
(result Value Error)
```

Construtores do núcleo:

```liaf
(call ok value)
(call err error-value)
```

Acesso a um `result` deve ser exaustivo. A futura construção de correspondência deverá exigir tratamento de ambos os casos. Operações de rede, arquivo e parsing não podem descartar erros.

### 10.1 Núcleo completo e bibliotecas de alto nível

A linguagem deve ser organizada em duas camadas:

```text
LIAF Core
├── valores, funções e controle de fluxo
├── tipos algébricos, coleções e pattern matching
├── módulos, genéricos e contratos
├── erros, efeitos, memória e concorrência
└── interoperabilidade controlada

LIAF Libraries
├── arquivos e processos
├── HTTP, rede e serviços web
├── serialização e bancos de dados
├── observabilidade e infraestrutura
└── integrações especializadas
```

O **Core** deve ser pequeno, formal e suficiente para expressar computação geral. As **Libraries** devem oferecer operações de domínio que reduzam o número de passos exigidos do modelo.

Modelos menores trabalharão preferencialmente com funções de alto nível:

```liaf
(call fs/read-text path)
(call http/client-get request)
(call db/query connection statement parameters)
(call process/start command arguments)
```

Quando uma operação de alto nível não existir, o programa ainda deve poder ser construído a partir de primitivas do Core ou de uma extensão tipada. A linguagem não pode depender de funções monolíticas como `(call create-complete-store ...)` para aparentar capacidade. Isso transformaria a LIAF em uma DSL ou linguagem de configuração limitada aos casos previamente antecipados.

### 10.2 Recursos necessários para propósito geral

Antes de ser classificada como linguagem geral, a evolução da v0.2 deve contemplar:

- recursão e/ou iteração estruturada;
- listas, mapas, conjuntos e acesso indexado seguro;
- structs e tipos algébricos;
- pattern matching exaustivo;
- funções como valores e closures, quando justificadas por benchmark;
- módulos e namespaces;
- genéricos com restrições explícitas;
- gerenciamento verificável de recursos;
- concorrência estruturada;
- FFI tipada ou alvo interoperável, como WebAssembly;
- capacidades para rede, arquivos, processos e sistema operacional;
- testes e contratos executáveis.

Cada recurso deve manter a regra central: poucas formas canônicas, sem reduzir as classes de programa que podem ser construídas.

### 10.3 Complexidade progressiva

A documentação fornecida ao modelo pode ser carregada em camadas. Uma tarefa simples recebe apenas o Core e os módulos necessários. Recursos avançados não precisam ocupar o contexto quando não participam do programa.

Essa organização reduz custo de contexto sem criar dialetos diferentes. Todos os módulos compartilham a mesma gramática e o mesmo protocolo de diagnóstico.

---

## 11. Diagnósticos para autocorreção

O compilador deve escrever somente JSON válido em `stdout` quando executado com saída estruturada. Mensagens humanas opcionais devem ir para `stderr`.

### 11.1 Envelope

```json
{
  "schema": "liaf.diagnostics/2",
  "status": "error",
  "source_hash": "sha256:...",
  "diagnostics": []
}
```

Em caso de sucesso, `diagnostics` deve ser `[]`, nunca `null`.

### 11.2 Diagnóstico

```json
{
  "code": "E_UNDEFINED_SYMBOL",
  "severity": "error",
  "phase": "type-check",
  "span": {
    "start_byte": 214,
    "end_byte": 227,
    "start_line": 9,
    "start_col": 27,
    "end_line": 9,
    "end_col": 40
  },
  "node_path": ["module", "main", "body", 1, "value", "right"],
  "expected": {
    "kind": "declared-symbol"
  },
  "actual": {
    "kind": "identifier",
    "value": "y-inexistente"
  },
  "fixes": [
    {
      "id": "fix-1",
      "confidence": "candidate",
      "op": "replace",
      "range": [214, 227],
      "text": "y"
    }
  ]
}
```

Requisitos:

- `code`, `severity` e `phase` são enums estáveis;
- offsets usam bytes UTF-8 e intervalos semiabertos `[start, end)`;
- `source_hash` impede aplicar um patch sobre outra versão;
- `node_path` localiza semanticamente o nó;
- uma correção incerta usa `confidence: "candidate"`;
- o compilador nunca afirma que um patch é seguro quando há múltiplas intenções possíveis;
- aplicar todos os patches marcados como seguros não pode gerar sobreposição de intervalos.

### 11.3 Códigos mínimos

```text
E_INVALID_TOKEN
E_UNCLOSED_STRING
E_UNCLOSED_NODE
E_UNKNOWN_NODE
E_WRONG_ARITY
E_DUPLICATE_SYMBOL
E_UNDEFINED_SYMBOL
E_TYPE_MISMATCH
E_RETURN_MISMATCH
E_MISSING_RETURN
E_UNDECLARED_EFFECT
E_CHANNEL_TYPE_MISMATCH
```

---

## 12. Edição estrutural

Além de patches textuais, ferramentas podem expor operações sobre a AST:

```json
{
  "schema": "liaf.patch/2",
  "source_hash": "sha256:...",
  "operations": [
    {
      "op": "replace-node",
      "path": ["module", "main", "body", 1, "value", "right"],
      "value": "y"
    }
  ]
}
```

Operações iniciais:

```text
replace-node
insert-before
insert-after
delete-node
rename-symbol
```

`rename-symbol` opera sobre identidade semântica, não sobre busca textual. Depois de qualquer operação, a ferramenta deve validar a AST completa e emitir novamente a forma canônica.

IDs persistentes de nós podem existir em uma representação lateral do compilador, mas não fazem parte obrigatória do código-fonte. Isso evita poluir o programa e permite que ferramentas mantenham identidade entre revisões.

---

## 13. Política de parsing

O parser da v0.2 é estrito:

1. Nunca descarta um token inesperado silenciosamente.
2. Nunca cria um nó parcial como se fosse válido.
3. Nunca interpreta um nome desconhecido como chamada, operador ou tipo por aproximação.
4. Nunca aceita argumentos excedentes.
5. Pode coletar múltiplos diagnósticos somente quando a recuperação mantém limites estruturais inequívocos.
6. Uma AST contendo nós de erro não pode chegar ao gerador de código.
7. O checker percorre recursivamente todos os filhos de toda expressão, mesmo quando o tipo do nó pai parece conhecido.

---

## 14. Versionamento

Ferramentas devem receber a versão de linguagem por argumento, manifesto do projeto ou extensão versionada. Enquanto a v0.1 e a v0.2 coexistirem, a seleção nunca deve depender de heurística.

Exemplo de manifesto futuro:

```liaf
(project liaf-example
  (language "0.2")
  (entry "src/main.liaf"))
```

Uma ferramenta configurada para v0.1 deve rejeitar sintaxe v0.2 com um diagnóstico de versão, e vice-versa.

---

## 15. Estratégia de avaliação

O design deve ser validado empiricamente com vários modelos, especialmente modelos pequenos e baratos, usando temperaturas e tamanhos de contexto controlados. Cada tarefa deve ser executada em v0.1 e v0.2 com documentação equivalente. Quando possível, também deve existir uma implementação de referência em uma linguagem convencional e em uma representação estruturada concorrente, como JSON ou Lisp simples.

O benchmark deve separar pelo menos três classes de modelo:

1. modelo pequeno e econômico;
2. modelo intermediário;
3. modelo avançado usado como referência de qualidade.

A meta não é demonstrar que o modelo avançado consegue escrever LIAF. A meta é reduzir a diferença entre o modelo pequeno e o avançado.

Métricas obrigatórias:

1. compilação correta na primeira tentativa;
2. correção comportamental em testes ocultos;
3. número de ciclos de autocorreção;
4. tokens de entrada e saída até o sucesso;
5. quantidade de diagnósticos repetidos;
6. erros semânticos não detectados pelo compilador;
7. preservação de comportamento após edição localizada;
8. taxa de patches aplicados sem conflito;
9. tempo total até a solução válida;
10. custo monetário de inferência;
11. tamanho da diferença de sucesso entre modelos pequenos e avançados;
12. porcentagem da tarefa resolvida por validação determinística em vez de nova geração;
13. cobertura de domínios não atendidos por builtins especializados.

A métrica principal é:

```text
custo_total_ate_sucesso = tokens_documentacao
                        + tokens_codigo
                        + tokens_diagnosticos
                        + tokens_correcoes
                        + tokens_testes
```

Quando o provedor disponibilizar preços por token, também deve ser calculado:

```text
custo_monetario_total = soma(custo_de_cada_chamada_do_modelo)
                      + custo_de_execucao_das_ferramentas
```

Um modelo barato usando vários ciclos pode ser preferível a um modelo avançado acertando na primeira tentativa, desde que o custo total, o tempo e a taxa de sucesso satisfaçam os limites do produto.

Uma sintaxe menor não é considerada melhor se exigir mais tentativas ou deixar mais erros semânticos. Uma biblioteca com muitos atalhos também não é considerada melhor se falhar em tarefas fora dos casos previamente implementados.

### 15.1 Critério de sucesso

A hipótese principal recebe suporte quando:

```text
sucesso(modelo-pequeno, LIAF-v0.2)
  ~= sucesso(modelo-avancado, linguagem-convencional)

e

custo(modelo-pequeno, LIAF-v0.2)
  < custo(modelo-avancado, linguagem-convencional)
```

Os limites de proximidade, custo, latência e correção devem ser definidos antes de executar o benchmark para evitar conclusões ajustadas aos resultados.

---

## 16. Programa completo de referência

```liaf
(module web-example
  (fn handle-request
    (params
      (path str))
    (returns str)
    (effects)
    (body
      (return
        (call concat
          "{\"status\":200,\"path\":\""
          path
          "\"}"))))
  (fn main
    (params)
    (returns void)
    (effects io net)
    (body
      (do (call println "server-starting"))
      (do (call http-serve ":8080" handle-request)))))
```

Árvore conceitual:

```text
module web-example
├── fn handle-request
│   ├── params: path str
│   ├── returns: str
│   ├── effects: none
│   └── body: return concat(...)
└── fn main
    ├── params: none
    ├── returns: void
    ├── effects: io, net
    └── body: println; http-serve
```

---

## 17. Decisões definitivas da v0.2

- A fonte é uma AST textual baseada exclusivamente em S-expressions.
- Tags de fechamento nomeadas da v0.1 são removidas.
- Chamadas genéricas usam `call` na v0.2. A v0.3 torna `call` opcional e mantém a forma antiga válida (seção 18.1).
- Operadores do núcleo permanecem prefixados.
- Tipos compostos são nós, não fragmentos com sintaxe especial.
- Assinaturas sempre incluem `params`, `returns` e `effects`.
- A sintaxe é estrita e possui forma canônica única.
- Diagnósticos usam códigos estáveis, hashes, intervalos e caminhos de AST.
- Patches estruturais são parte do protocolo de ferramentas.
- O compilador não realiza correção silenciosa nem inferência aproximada de intenção.
- O sucesso da versão será decidido por benchmark multi-modelo, não por preferência estética.
- O público prioritário de avaliação são modelos menores e de menor custo.
- A linguagem restringe formas equivalentes, não as classes de programa possíveis.
- O Core deve permanecer pequeno, combinável e capaz de sustentar propósito geral.
- Bibliotecas de alto nível reduzem esforço sem substituir as primitivas fundamentais.
- O custo monetário total por tarefa correta é uma métrica primária.

---

## 18. Adições da v0.3

A v0.3 não muda o núcleo semântico da v0.2. Ela reduz o número de decisões que um modelo precisa acertar para produzir um programa válido na primeira tentativa. Todo programa v0.2 continua compilando sem alteração.

### 18.1 Chamadas diretas

`(nome arg1 arg2 ...)` é uma chamada. A forma `(call nome arg1 arg2 ...)` continua aceita e produz a mesma AST.

```liaf
(json-encode tarefa)          ;; canônico na v0.3
(call json-encode tarefa)     ;; v0.2, ainda aceito
```

O parser resolve a ambiguidade pela posição: dentro de `(...)`, um identificador em posição de cabeça é o alvo de uma chamada. Operadores do núcleo permanecem prefixados e não mudam.

### 18.2 `try` e `on-err`

`(try expr)` exige que `expr` tenha tipo `(result T E)`. O valor da expressão `try` é `T`. Se o resultado for `err`, a função retorna imediatamente, e o destino desse retorno é:

- o bloco `(on-err var (instruções...))` da função ou rota, se existir; `var` recebe a mensagem de erro;
- caso contrário, o próprio `err`, se a função retorna `(result ...)`.

Sem nenhum dos dois, o checker emite `E_UNHANDLED_RESULT`. `try` sobre um valor que não é `result` emite `E_TYPE_MISMATCH`.

O bloco `(on-err ...)` é opcional, aparece no máximo uma vez e vem imediatamente antes de `(body ...)`.

```liaf
(fn salvar (params (estado Estado)) (returns Response) (effects fs io)
  (on-err message (return (json-response 500 message)))
  (body
    (let texto str (try (json-encode estado)))
    (try (fs-write-atomic "estado.json" texto))
    (return (json-response 200 "{\"ok\":true}"))))
```

`match` continua sendo a forma de tratar sucesso e erro de maneiras diferentes. `try` cobre o caso em que o erro só precisa ser propagado — que é a maioria, e era onde a v0.2 produzia pirâmides de `match` aninhado.

### 18.3 Rotas declarativas

`route` é uma declaração de topo, irmã de `fn`:

```
(route MÉTODO "/caminho" (params ...) (returns T) (effects ...) [(on-err var ...)] (body ...))
```

`MÉTODO` é `GET`, `POST`, `PUT`, `DELETE` ou `OPTIONS` (este último para o preflight de CORS; ver §21). Rotas se registram sozinhas na inicialização do binário; não é preciso chamar `http-get` e afins.

Regras de parâmetro:

- Cada `{nome}` no caminho precisa de uma entrada de mesmo nome em `(params ...)`, do tipo `int` ou `str`. Violações emitem `E_ROUTE_PARAM`.
- Um parâmetro de path chega já convertido, extraído da posição que ocupa no padrão — `/users/{id}/tasks` lê o segundo segmento, não o último.
- Um parâmetro de tipo struct recebe o corpo da requisição já desserializado. Corpo JSON inválido produz `400` antes de o corpo da rota executar.
- Um parâmetro de tipo `Request` recebe a requisição bruta, como na v0.2. Headers e query string são lidos dela com `request-header` e `request-query` (§21).

Se `(returns Response)`, a resposta da rota é usada como está. Se `(returns void)`, a resposta é `200`. Para qualquer outro tipo, o valor é serializado em JSON com status `201` em `POST` e `200` nos demais métodos.

### 18.4 I/O atômico e JSON

- `fs-write-atomic caminho texto` grava num arquivo temporário e renomeia sobre o destino. Serve para estado que não pode ficar pela metade se o processo morrer durante a escrita.
- `fs-rename origem destino` expõe o rename diretamente.
- `fs-write-file`, `fs-rename`, `fs-write-atomic` e `fs-remove` retornam `(result void str)`. O sucesso não carrega valor: `ok` já significa que a operação terminou. Vincular e inspecionar esse valor é erro de tipo. Isso elimina o caminho `ok false`, que duplicava o tratamento de erro e era esquecido com frequência.
- `json-encode` de `(list T)` produz um array JSON nativo `[...]`. A v0.2 produzia `{"Items":[...]}`, o que obrigava a compor arrays manualmente para respostas HTTP.
- `json-encode` e `json-decode` **não têm efeito**. Operam sobre strings em memória; exigir `io` delas fazia qualquer função que apenas serializa parecer efetuosa, e era um erro comum de declaração.
- `json-decode` aceita tipos compostos além de nomes de struct: `(json-decode texto (list Task))`. Construtores de tipo em posição de argumento — `list`, `map`, `chan`, `result` — são traduzidos pela mesma rotina que o checker usa para `make-list` e afins.

### 18.5 Efeitos declarados e não usados

Um efeito listado em `(effects ...)` que nenhuma operação do corpo consome — diretamente ou através de uma função chamada — emite `E_UNUSED_EFFECT`. Uma assinatura que promete `fs` sem tocar o disco descreve mal o que a função faz, para um leitor humano e para um modelo que gera código a partir dela.

### 18.6 Forma canônica e `liafc fmt`

A forma canônica da v0.3 usa chamadas diretas, `try`/`on-err` onde o erro só é propagado, e `route` para endpoints HTTP. `liafc fmt` imprime essa forma; `-w` grava no arquivo e `-l` lista o que está fora do formato.

**Limitação conhecida:** o formatador reimprime a AST, e a AST não guarda comentários. Formatar um arquivo comentado apagaria todos eles, então `-w` se recusa a gravar nesse caso a menos que `--drop-comments` seja passado explicitamente. Preservar comentários exige anexá-los aos nós no parser, o que não está implementado.

### 18.7 Códigos de diagnóstico introduzidos

| Código | Significado |
|---|---|
| `E_UNHANDLED_RESULT` | `try` sem `(on-err ...)` e sem retorno `(result ...)` |
| `E_ROUTE_PARAM` | `{nome}` no caminho sem parâmetro correspondente, ou com tipo que não é `int` nem `str` |
| `E_UNUSED_EFFECT` | efeito declarado que o corpo nunca consome |

## 19. Adições da v0.4

A v0.4 acrescenta acesso a bancos de dados externos (issue #015) e conexões WebSocket bidirecionais (issue #016). Todas as formas da v0.3 continuam válidas.

Os dois protocolos são implementados sobre TCP dentro do próprio compilador: o binário gerado não carrega driver externo nem biblioteca de WebSocket. É a mesma razão da issue #010 — um programa LIAF é um executável único.

### 19.1 O efeito `db`

`db` é o sexto efeito, ao lado de `io`, `net`, `fs`, `clock` e `spawn`. Toda operação de banco o exige. É distinto de `net` de propósito: uma assinatura que diz `db` promete mais do que tráfego de rede — promete estado externo compartilhado, que sobrevive ao processo e é visível a outros.

### 19.2 Tipos opacos

`DBConnection` e `WSConn` são tipos reconhecidos sem `(struct ...)` correspondente, como `Request` e `Response`. Não têm campos: `(field conn host)` é `E_UNKNOWN_FIELD`. O programa recebe o valor de um builtin, passa adiante e nunca o inspeciona.

### 19.3 Conexão e consulta

```
(db-connect "postgres" dsn)              -> (result DBConnection str)
(db-close conn)                          -> (result void str)
(db-query conn "SQL" TipoLinha arg...)   -> (result (list TipoLinha) str)
(db-exec conn "SQL" arg...)              -> (result int str)
(redis-get conn chave)                   -> (result str str)
(redis-set conn chave valor ttl)         -> (result void str)
```

Drivers: `"postgres"`, `"mysql"`, `"redis"` e `"sqlite"`. O nome do driver é um literal, conferido em tempo de compilação — `"postgresql"` e `"pg"` são `E_UNKNOWN_DRIVER`.

DSN na forma URL (`postgres://usuario:senha@host:5432/banco?sslmode=require`). O PostgreSQL também aceita a forma de palavras-chave (`host=... user=... dbname=...`). O parâmetro `pool_max=N` limita as conexões simultâneas do pool; o padrão é 16. No SQLite, o DSN é o caminho do arquivo; o runtime liga `busy_timeout=5000`, `journal_mode=WAL` e `synchronous=NORMAL` em toda conexão, a menos que o DSN defina a mesma PRAGMA (`app.db?_pragma=journal_mode(DELETE)`).

**Pool compartilhado.** Chamadas de `db-connect` com o mesmo driver e o mesmo DSN dividem um único pool, e cada uma devolve um handle próprio. Como a LIAF não tem variável global, o jeito natural de escrever uma rota é chamar `db-connect` dentro dela; o compartilhamento faz isso custar uma busca em mapa, e não uma conexão nova por requisição. `db-close` fecha só o handle recebido e é idempotente: outros handlers que usam o mesmo pool continuam funcionando, e o pool só fecha suas conexões quando o último handle é fechado. Não chamar `db-close` mantém o pool aberto até o fim do processo, o que é o esperado num servidor.

`db-exec` devolve o número de linhas afetadas. `redis-get` trata chave ausente como erro: `(result str str)` não tem um terceiro caso para representar ausência.

Os parâmetros são posicionais e seguem o tipo de linha (`db-query`) ou o SQL (`db-exec`). Cada um tem de ser `int`, `float`, `str` ou `bool`. A issue #015 propunha agrupá-los numa lista — `(db-query conn sql (list id) User)` — mas listas na LIAF são homogêneas, e uma consulta com `id` inteiro e `email` texto não caberia em `(list T)` nenhum. Argumentos variádicos preservam a intenção sem inventar um tipo de tupla.

### 19.4 A regra contra injeção de SQL

**O primeiro argumento de texto de `db-query` e `db-exec` tem de ser um literal de string escrito no fonte.** Qualquer outra expressão — uma variável, um `(concat ...)` — é `E_SQL_INTERPOLATION`.

Consequência: nenhum valor de tempo de execução consegue virar sintaxe SQL, porque o único caminho até o banco é a lista de parâmetros. Os drivers enviam os argumentos fora do texto do comando (mensagem `Bind` no PostgreSQL, `COM_STMT_EXECUTE` no MySQL), e não por escape.

A regra é deliberadamente rígida. A forma insegura é a mais curta, a mais natural de escrever e a que mais aparece em exemplos na internet; um modelo gerando código a escolhe por default. Não existe escapatória: montar SQL dinamicamente — um `ORDER BY` variável, por exemplo — não é expressável.

O checker também confere a contagem de marcadores contra a de argumentos e emite `E_SQL_PARAM_COUNT` quando divergem, ou quando `$1` e `?` aparecem misturados. Marcadores dentro de literais, de identificadores entre aspas e de comentários não contam. A contagem é abandonada quando o SQL contém `?|`, `?&` ou `??`, que são operadores de JSONB do PostgreSQL e não marcadores.

### 19.5 Conversão de linha em struct

`db-query` exige um tipo de linha que seja struct declarada — um escalar não tem nome de campo com que casar a coluna. Cada coluna selecionada alimenta o campo de mesmo nome; uma coluna `snake_case` também alimenta o campo `kebab-case` correspondente, que é como a LIAF escreve nomes compostos. Colunas `NULL` viram o zero do campo. Uma coluna que não cabe no tipo do campo produz erro em tempo de execução, com a indicação do que não coube.

### 19.6 Transações

```
(db-transaction conn
  instrução...)
```

`conn` é o nome de uma variável `DBConnection`. **Dentro do bloco esse mesmo nome passa a designar a transação**, então as consultas do corpo não mudam de forma — e não existe a forma errada de usar o pool original por engano no meio de uma transação.

O bloco confirma ao chegar ao fim e desfaz em qualquer saída antecipada, inclusive num `(try ...)` que falhou no meio. Como abrir e confirmar podem falhar, `db-transaction` exige um destino para o erro: `(on-err ...)` ou uma função que retorne `(result ...)`. Sem isso, `E_UNHANDLED_RESULT`.

Um `return` dentro do bloco conta como retorno da função: ao contrário de um laço, o corpo da transação sempre executa. Transações aninhadas não são suportadas.

### 19.7 Rotas WebSocket

```
(ws-route "/caminho/{param}" (params ...) (effects ...) [(on-err var ...)]
  [(on-open instrução...)]
  (on-message var instrução...)
  [(on-close instrução...)])
```

`ws-route` é uma declaração de topo, irmã de `fn` e `route`. Não tem `(returns ...)` nem `(body ...)`: uma conexão bidirecional não termina numa resposta, e os três blocos são o corpo — cada um disparado por um momento diferente da vida da conexão. A ordem dos blocos é fixa; `on-open` e `on-close` são opcionais, `on-message` não é.

Regras de parâmetro (`E_WS_PARAM` quando violadas):

- Exatamente um parâmetro do tipo `WSConn`, que recebe a conexão aberta.
- No máximo um parâmetro do tipo `Request`, que recebe a requisição do handshake e chega aos três blocos. É por ele que se autentica: a API de WebSocket do navegador não permite definir `Authorization`, então o token costuma vir na query string (`/ws/painel?token=...`), lido com `request-query` (§21).
- Todos os demais parâmetros correspondem a um `{nome}` no caminho e são `int` ou `str`. Não há corpo JSON num handshake de WebSocket, então um parâmetro que não venha do caminho não teria de onde ser preenchido.

`(effects ...)` tem de incluir `net`: manter a conexão aberta já é tráfego de rede. Por isso `net` conta como usado numa `ws-route` mesmo que nenhum builtin `ws-*` seja chamado.

Os três blocos devolvem `void`, então `try` dentro deles exige `(on-err ...)` — a mesma regra de uma função `void`. Cada bloco tem escopo próprio: a variável de `on-message` não existe em `on-close`.

`on-close` roda tanto no fechamento limpo quanto na queda da conexão. Um handshake que não é WebSocket válido recebe `400` e a rota não executa.

O handshake é sempre `GET`, então rotas WebSocket convivem com rotas HTTP no mesmo `serve-hybrid` e na mesma porta.

### 19.8 Primitivas de conexão e tópicos

```
(ws-send conn mensagem)        -> (result void str)
(ws-send-json conn valor)      -> (result void str)
(ws-close conn código motivo)  -> (result void str)
(ws-join conn tópico)          -> void
(ws-leave conn tópico)         -> void
(ws-broadcast tópico mensagem) -> (result int str)
(ws-topic-size tópico)         -> int
```

`ws-join` e `ws-leave` não devolvem `result`: são operações locais sobre um mapa em memória e não têm caminho de falha. Envolvê-las em `result` obrigaria a um `try` que nunca desvia.

A inscrição num tópico é explícita — uma sala nunca é inferida do caminho. Sem `ws-join`, a conexão não recebe broadcast. Fechar a conexão desinscreve de todos os tópicos automaticamente, então `on-close` não precisa chamar `ws-leave`.

`ws-broadcast` devolve quantos clientes receberam. O pub/sub é local ao processo: atende o caso descrito na issue #016, mas não substitui um broker quando há mais de um nó, porque cada processo tem o seu mapa.

Limites do transporte: mensagem de até 1 MiB (o mesmo teto do corpo HTTP), ping a cada 30 s, conexão ociosa derrubada em 90 s. O servidor exige máscara em todo frame do cliente, como manda a RFC 6455. Extensões negociadas (`permessage-deflate`) não são suportadas.

### 19.9 Limitações conhecidas

- O backend C não implementa nada desta seção. `ws-route` é erro explícito; os builtins de banco caem em "unsupported operation".
- `db-query` materializa o resultado inteiro em memória. Não há cursor: expor um exigiria um tipo com ciclo de vida próprio na linguagem.
- `permessage-deflate`, `SCRAM-SHA-256-PLUS` e o papel de cliente WebSocket não estão implementados.
- No PostgreSQL, `sslmode=prefer` e `allow` cifram o tráfego mas não verificam o certificado do servidor, como no libpq. Use `verify-full` quando a verificação importar.
- Um `(db-transaction ...)` dentro de um laço acumula um `defer` por iteração, liberado só no fim da função. Todos são no-op depois do commit, mas um laço muito longo segura essas entradas.

### 19.10 Códigos de diagnóstico introduzidos

| Código | Significado |
|---|---|
| `E_SQL_INTERPOLATION` | SQL montado em tempo de execução em vez de literal |
| `E_SQL_PARAM_COUNT` | número de marcadores diferente do número de argumentos, ou `$1` e `?` misturados |
| `E_UNKNOWN_DRIVER` | driver que não é `postgres`, `mysql` nem `redis`, ou nome não literal |
| `E_WS_PARAM` | `ws-route` sem exatamente um `WSConn`, ou com parâmetro que não vem do caminho |

---

## 20. Núcleo básico da linguagem (v0.4.1 — issues #019 a #024)

Consolidação da biblioteca padrão e garantias semânticas fundamentais para geração por IA.

### 20.1 Tabela unificada de builtins (#019)
Todos os built-ins da linguagem são declarados e validados por uma fonte canônica única em `pkg/builtins/table.go`, garantindo paridade estrita entre verificação estática (checker), backend Go e backend nativo C.

### 20.2 Aritmética e funções matemáticas (#020)
- **Operadores variádicos:** `add`, `sub`, `mul` aceitam 2 ou mais argumentos: `(add 1 2 3 4)` -> `10`.
- **Operações inteiras e de ponto flutuante:**
  - `(mod a b)`: resto inteiro.
  - `(neg x)`: negação numérica unária (`-x`).
  - `(abs x)`: valor absoluto de inteiros e floats.
  - `(min a b ...)` e `(max a b ...)`: mínimo e máximo variádico para números.
  - `(pow base exp)`: exponenciação (`float`).
  - `(sqrt x)`: raiz quadrada (`float`).
  - `(floor x)`, `(ceil x)`, `(round x)`: arredondamento de `float` retornando `int`.
- **Literais numéricos estendidos:** Suporte léxico a hexadecimais (`0x1F`, `0XFF`) e notação científica (`1e3`, `2.5e-2`).

### 20.3 Conversões de tipo explícitas (#021)
- `(int-from-float f)`: trunca `float` para `int`.
- `(str-from-float f)`: formata `float` como `str`.
- `(float-from-str s)`: analisa string para ponto flutuante, retornando `(result float str)`.
- `(str-from-bool b)`: converte `bool` para `"true"` ou `"false"`.
- `(bool-from-str s)`: analisa `"true"` ou `"false"` retornando `(result bool str)`.

### 20.4 Semântica numérica segura (#022)
- **Divisão inteira segura:** `(div a b)` e `(mod a b)` com operandos inteiros retornam `(result int str)`. Se o divisor for zero, retornam `(err "div: divisao por zero")` ou `(err "mod: divisao por zero")`.
- **Proteção estrita contra overflow:** Operações aritméticas em `int` (`add`, `sub`, `mul`, `neg`, `abs`) que resultarem em estouro de representação (overflow ou underflow em 64 bits) abortam imediatamente o processo com código de saída `1`, impedindo corrupção silenciosa de memória ou cálculos financeiros errôneos.

### 20.5 Strings e UTF-8 nativo (#023)
- **Unidade de contagem e indexação em caracteres Unicode (runes):**
  - `(str-len s)`: quantidade de caracteres Unicode (runes).
  - `(str-byte-len s)`: tamanho bruto em bytes UTF-8.
  - `(str-get s i)`: retorna o caractere na posição `i` como string de 1 rune.
  - `(str-slice s inicio fim)`: subfatia baseada em contagem de runes.
  - `(str-index s substr)`: primeiro índice por runes onde `substr` ocorre (`-1` se ausente).
- **Operações e busca:**
  - `(str-contains s substr)`: booleano de presença.
  - `(str-starts-with s prefix)` e `(str-ends-with s suffix)`.
  - `(str-split s sep)`: fatia string em `(list str)`.
  - `(str-join lista sep)`: agrupa `(list str)` com delimitador.
  - `(str-trim s)`: remove espaços em branco das extremidades.
  - `(str-upper s)` e `(str-lower s)`: transformação de caixa.
  - `(str-replace s antigo novo)`: substitui todas as ocorrências.
- **Comparações de string:** Operadores `lt`, `gt`, `lte`, `gte` comparam strings em ordem lexicográfica.

### 20.6 Coleções completas e tipo Option (#024)
- **Literais de lista:** `(list 1 2 3)` infere `(list int)`. Lista vazia tipada com `(list-new T)`.
- **Manipulação de listas:**
  - `(list-remove lista indice)`: remove elemento pelo índice.
  - `(list-pop lista)`: desempilha o último elemento retornando `(result T str)`.
  - `(list-sort lista)`: ordena elementos in-place (para tipos ordenáveis).
  - `(list-contains lista elemento)`: checa pertinência booleana.
- **Manipulação de mapas:**
  - `(map-delete mapa chave)`: remove chave.
  - `(map-keys mapa)`: retorna `(list K)` com chaves ordenadas deterministicamente.
- **Tipo `(option T)`:**
  - Construtor com valor: `(some valor)`.
  - Construtor sem valor: `(none Tipo)`.
  - Desconstrução exaustiva por casamento de padrão:
    ```
    (match opt
      (some v (println v))
      (none (println "vazio")))
    ```

## 21. Headers e query string

Três builtins sem efeito, para autenticação, filtros e controle da resposta:

```
(request-header req "Authorization")          -> (result str str)
(request-query req "status")                  -> (result str str)
(response-set-header res "Cache-Control" "no-store") -> Response
```

- `request-header` não diferencia maiúsculas no nome, como o HTTP. Com o header repetido, devolve o primeiro valor.
- `request-query` lê a query string (`/api/pedidos?status=aberto`), que não faz parte de `request-path`. O nome diferencia maiúsculas, como a URL.
- As duas seguem a convenção de `env-get` e `map-get`: ausência é `(err ...)`. Um header presente e vazio devolve `(ok "")`; para quem autentica, "vazio" e "ausente" são casos diferentes.
- `response-set-header` devolve uma **nova** `Response`; a original não muda, então a mesma resposta base pode ser usada em vários ramos. Chamadas se aninham: `(response-set-header (response-set-header r "A" "1") "B" "2")`.
- Repetir um header substitui o valor anterior, exceto `Set-Cookie`, que acumula (cada cookie é um header). Definir `Content-Type` troca o `application/json` de `json-response`.
- Nome fora do token da RFC 9110 (espaço, dois-pontos, vazio) ou valor com quebra de linha ou NUL faz a rota responder `500`, sem o corpo e sem os outros headers. Isso impede injeção de header com um valor vindo do usuário.

Exemplo — autenticação por Bearer, com a rota recebendo o `Request`:

```liaf
(fn autenticar (params (req Request)) (returns (result str str)) (effects)
  (body
    (let header str (try (request-header req "Authorization")))
    (if (not (str-starts-with header "Bearer "))
      (then (return (err "esperado Authorization: Bearer <token>"))))
    (return (ok (try (str-slice header 7 (str-len header)))))))
```

Programa completo, com isolamento por restaurante, filtro por query, preflight de CORS e `Set-Cookie`: `pkg/codegen/testdata/pedidos_api.liaf`.

Numa `(ws-route ...)`, um parâmetro `Request` recebe o handshake, com os mesmos headers e query string (§19.7). Recusar a conexão é fechá-la em `on-open` com `(ws-close conn 4401 "motivo")`, antes de `ws-join`; `pkg/codegen/testdata/pedidos_api.liaf` faz isso no painel de cada restaurante.

## 22. Criptografia

Primitivas para senha, token de sessão, assinatura de webhook e JWT. Nenhuma aceita parâmetro de segurança vindo do programa (tamanho de token, custo do hash, algoritmo): o caminho fácil é também o seguro.

```
(sha256 texto "hex")                   -> str
(hmac-sha256 chave texto "base64url")  -> str
(secure-eq a b)                        -> bool
(base64url-encode texto)               -> str
(base64url-decode texto)               -> (result str str)
(random-token)                         -> str        ;; efeito rand
(password-hash senha)                  -> str        ;; efeito rand
(password-verify senha hash)           -> bool
```

- A codificação de `sha256` e `hmac-sha256` é um literal, `"hex"` ou `"base64url"`, conferido na compilação (`E_UNKNOWN_ENCODING`), como o driver de `db-connect`. Webhooks costumam assinar em hex; JWT usa base64url.
- **Compare assinaturas com `secure-eq`, nunca com `eq`/`str-eq`.** A comparação comum para no primeiro caractere diferente, e o tempo de resposta revela quanto da assinatura já está certo.
- `base64url` usa o alfabeto de URL, sem `=`; o decode aceita a entrada com ou sem padding. O resultado do decode precisa ser texto UTF-8, porque a linguagem ainda não tem tipo de bytes.
- `random-token` sorteia 32 bytes do gerador criptográfico e devolve 43 caracteres base64url: serve como token de sessão, de convite ou de recuperação de senha.
- `password-hash` usa argon2id com os parâmetros da OWASP (19 MiB, 2 passadas) e sal aleatório, no formato PHC: `$argon2id$v=19$m=19456,t=2,p=1$<sal>$<hash>`. Como os parâmetros ficam no próprio texto, um padrão mais forte no futuro não invalida senhas já gravadas.
- `password-verify` devolve `false` tanto para senha errada quanto para hash malformado ou com parâmetros absurdos.
- Os hashes de senha simultâneos são limitados ao número de CPUs: uma rajada de logins espera na fila em vez de esgotar a memória.

Exemplo — assinatura HS256 de um JWT:

```liaf
(let cabecalho str (base64url-encode "{\"alg\":\"HS256\",\"typ\":\"JWT\"}"))
(let corpo str (base64url-encode payload-json))
(let assinatura str (hmac-sha256 segredo (concat cabecalho "." corpo) "base64url"))
```

Os vetores conhecidos (SHA-256 da FIPS 180-2, HMAC da RFC 4231 e o JWT de exemplo do jwt.io) estão em `conformance/basics/027_crypto_*.liaf`.

## 23. Módulos e biblioteca padrão

`(import "caminho")` traz as declarações de outro arquivo para o módulo, como se estivessem escritas nele. O loader resolve os imports antes do checker, detecta ciclos (`E_CIRCULAR_IMPORT`) e carrega uma única vez um arquivo importado por dois caminhos.

- `"./util.liaf"` ou `"./util"`: arquivo relativo a quem importa. Um diretório importa todos os `.liaf` dele, em ordem alfabética.
- `"std/<nome>"`: módulo da **biblioteca padrão**, embutido no `liafc`. Não lê o disco nem a rede, e a versão é sempre a do compilador. Uma pasta local chamada `std` se importa com `"./std/..."`.
- Um módulo da std só importa outros da std (`E_STD_RELATIVE_IMPORT`).
- Módulo inexistente é `E_IMPORT_NOT_FOUND`, com a lista dos disponíveis na mensagem.

Módulos disponíveis:

| Módulo | Oferece |
|---|---|
| `std/auth` | `(auth-bearer-token req) -> (result str str)`: o token de `Authorization: Bearer ...`, com o nome do esquema sem diferenciar maiúsculas (`bearer` vale); erro se o header faltar, usar outro esquema ou vier vazio |
| `std/jwt` | `(jwt-sign segredo payload-json) -> (result str str)` e `(jwt-verify segredo token) -> (result str str)`, com efeito `clock`. Só HS256. Os dois recusam segredo com menos de 32 bytes (RFC 7518 §3.2), para que uma variável de ambiente ausente não vire token assinado com `""`. O verify confere a assinatura antes de interpretar qualquer parte, recusa `alg` diferente de HS256 (inclusive `none`), exige o claim `exp` e recusa token vencido; devolve o payload JSON para o programa decodificar no próprio struct |
| `std/cors` | `(cors-origin req permitidas) -> (result str str)`: a origem, se estiver na lista; `(cors-headers res origem) -> Response`: `Access-Control-Allow-Origin` + `Vary: Origin`; `(cors-preflight origem metodos cabecalhos) -> Response`: 204 para o `OPTIONS`. Nunca usa `*`. Para API autenticada por cookie, `cors-headers-credentials` e `cors-preflight-credentials` acrescentam `Access-Control-Allow-Credentials: true`, sem o qual o navegador descarta a resposta |

Exemplo — sessão com JWT:

```liaf
(import "std/jwt")
(struct Sessao (fields (sub str) (exp int)))

;; login
(let expira int (add (try (div (now-ms) 1000)) 3600))
(let token str (try (jwt-sign segredo (try (json-encode (new Sessao usuario expira))))))

;; em cada requisição
(let sessao Sessao (try (json-decode (try (jwt-verify segredo token)) Sessao)))
```

Nomes de campo aceitam palavras reservadas (`sub`, `return`, `if`...), exceto `true` e `false`: o nome do campo é só um rótulo e a chave no JSON, e é assim que os claims registrados do JWT (`sub`, `exp`) se decodificam com os nomes originais. `(field sessao sub)` lê o campo; `(sub a b)` continua sendo subtração.

Como a linguagem ainda não tem middleware, o preflight de CORS exige uma `(route OPTIONS ...)` por caminho; `pkg/codegen/testdata/pedidos_api.liaf` mostra o padrão.

**Colisões.** Todas as declarações dividem um único espaço de nomes, e funções e structs dividem o mesmo espaço entre si. Um nome declarado em dois arquivos é `E_DUPLICATE_DECL`, com os dois locais na mensagem; se um deles é da std, o nome pertence a ela e o seu precisa mudar. Por isso os módulos da std prefixam os nomes públicos com o próprio nome (`auth-`). Nome repetido no mesmo arquivo continua sendo `E_DUPLICATE_SYMBOL`, do checker.

**Rotas repetidas** são `E_DUPLICATE_ROUTE`, mesmo no mesmo arquivo: método e caminho iguais, sem diferenciar maiúsculas no método. Uma `ws-route` ocupa o `GET` do seu caminho. Sem essa checagem, o servidor do binário falharia ao subir.

## 24. Cliente HTTP

```
(http-fetch metodo url cabecalhos corpo)  -> (result HttpReply str)   ;; efeito net
(reply-status r)                          -> int
(reply-body r)                            -> str
(reply-header r "Nome")                   -> (result str str)
```

`http-get`, `http-post` e afins já são a forma v0.2 de registrar rotas; por isso o cliente se chama `http-fetch`. `HttpReply` é um tipo opaco, como `Request`.

- **Como o `fetch` do JavaScript:** `(err ...)` só quando não houve resposta — DNS, conexão, TLS, timeout, URL inválida. Um `404` ou `422` é resposta e chega como `(ok ...)`; o programa confere `reply-status`, e muitas APIs mandam o motivo do erro no corpo.
- `metodo` é `GET`, `POST`, `PUT`, `PATCH`, `DELETE`, `HEAD` ou `OPTIONS`. Um literal fora da lista é `E_UNKNOWN_METHOD` na compilação; um método vindo de variável é conferido na execução.
- `cabecalhos` é uma lista de `"Nome: valor"`, como no `curl -H`: `(list (concat "Authorization: Bearer " token))`; `(list)` para nenhum. Nome inválido ou quebra de linha no valor é erro, o que impede injeção de header.
- Corpo não vazio sem `Content-Type` declarado vai como `application/json`.
- A URL precisa ser `http://` ou `https://` com host. Redirecionamentos são seguidos (até 10); o `Authorization` não é repassado para outro domínio.
- Timeout de 30 segundos para a chamada inteira. Corpo de resposta limitado a 10 MiB e obrigatoriamente texto UTF-8 (a linguagem ainda não tem tipo de bytes); passar disso é erro, não truncamento.
- As mensagens de erro trazem método e host, nunca a URL inteira: tokens na query string não vão parar em log.

Exemplo completo, com token, chave de idempotência e tratamento do erro do provedor: `pkg/codegen/testdata/cobranca_pix.liaf`.

**Cuidado:** chamar uma URL que veio do usuário deixa ele apontar o servidor para endereços internos (SSRF). Monte a URL a partir de uma base fixa.

## 25. Sintaxe Canônica v0.5 (Compacta e Redução de Tokens)

A partir da v0.5, a LIAF adota uma sintaxe canônica compacta voltada para economia de tokens BPE em modelos de IA e eliminação de mutabilidade imperativa:

### 25.1 Structs compactas
Elimina a tag `fields`. Os campos são listados diretamente após o nome:
```liaf
(struct User (id int) (name str) (email str))
```

### 25.2 Funções e Rotas compactas
A assinatura segue a ordem posicional: `[nome] [params] [retorno] [efeitos] [corpo...]`.
Elimina as tags redundantes `params`, `returns` e `body`:
```liaf
;; Parâmetros vazios são expressos por ()
(fn main () void (effects io)
  (println "Olá mundo"))

;; Função tipada com parâmetros
(fn somar ((a int) (b int)) int (effects)
  (add a b))

;; Rota HTTP declarativa compacta
(route GET "/api/users/{id}" ((id int)) Response (effects db)
  (let user User (try (db-query db "SELECT id, name FROM users WHERE id = ?" User id)))
  (json-response 200 (try (json-encode user))))
```

### 25.3 Retorno Implícito
Em funções e rotas não-`void`, a última expressão avaliada no bloco é retornada implicitamente, eliminando a obrigatoriedade da tag `(return ...)` final quando a expressão é auto-contida.

### 25.4 `match` como expressão de valor
O bloco `match` pode ser avaliado como valor e atribuído diretamente a um `let`:
```liaf
(let token str
  (match (request-header req "Authorization")
    (ok t t)
    (err _ "")))
```

### 25.5 Combinador `unwrap-or`
Desempacota `(result T E)` ou `(option T)` diretamente com valor de fallback em 1 linha:
```liaf
(let status str (unwrap-or (request-query req "status") "todos"))
```

### 25.6 Interpolação `(fmt ...)`
Substitui cadeias profundas de `concat` por interpolação posicional `{}`:
```liaf
(println (fmt "Usuário {} conectado na sala {}" user room))
```

