# LIAF v0.2 — Especificação proposta para agentes de IA

**Status:** especificação de design parcialmente implementada; consulte [IMPLEMENTATION.md](IMPLEMENTATION.md) para o contrato executável atual
**Compatibilidade:** incompatível com a sintaxe LIAF v0.1  
**Público-alvo:** modelos de linguagem e agentes autônomos  
**Objetivo:** permitir que modelos menores e de menor custo produzam software correto, minimizando o esforço de inferência e o custo total até uma solução validada

> O compilador atual aceita a sintaxe v0.2. Recursos propostos neste documento, como protocolo completo de patches estruturais, não devem ser considerados implementados sem confirmação em IMPLEMENTATION.md e testes. A sintaxe v0.1 é histórica.

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

Chamadas sempre usam o nó `call`. Assim, `(add a b)` é inequivocamente um operador do núcleo, enquanto `(call add a b)` seria uma chamada a uma função chamada `add`, caso esse identificador não seja reservado.

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
(call str-from-int count)
(call float-from-int count)
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
- Chamadas genéricas usam obrigatoriamente `call`.
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
