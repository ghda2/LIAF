# Referência de Sintaxe da LIAF (v0.6 Canônica)

A LIAF v0.6 adota sintaxe linear baseada em linhas, operadores infixos naturais e delimitadores explícitos (`end`).

---

## 1. Módulos e Visibilidade (`pub`)

Todo arquivo LIAF inicia com a declaração do seu módulo.
- **Privado por padrão:** Todas as declarações (`fn`, `struct`, `enum`) são privadas ao arquivo/módulo.
- **Exportação explícita com `pub`:** Apenas o que tiver `pub` na frente é exposto para outros módulos que importarem este arquivo.

```liaf
module loja

// Importações
import "std/auth"
import "std/cli"
import "./database"

// Struct pública (visível fora deste módulo)
pub struct Produto
  id int
  nome str
  preco float
  ativo bool
end

// Struct privada (helper interno deste módulo)
struct ItemInterno
  chave str
  hash str
end

// Enum público
pub enum StatusPedido
  Pendente
  Processando
  Entregue
  Cancelado
end
```

---

## 2. Funções (`fn` e `pub fn`)

```liaf
// 1. Função pública pura (sem efeitos colaterais) com retorno implícito
pub fn somar(a int, b int) int
  a + b
end

// 2. Função privada com efeitos e sem retorno (void)
fn registrar_log(msg str) void effects(io)
  println(fmt("[LOG] {}", msg))
end

// 3. Função com tratamento de erros integrado (on-err e try)
pub fn carregar_produto(id int) (result Produto str) effects(fs, db)
  on-err err_msg
    return err(concat("falha ao carregar produto: ", err_msg))
  end
  let dados = try db-query-one("SELECT id, nome, preco, ativo FROM produtos WHERE id = ?1", Produto, id)
  ok(dados)
end
```

---

## 3. Rotas HTTP Declarativas (`route`)

```liaf
// Rota GET com parâmetro tipado na URL
route GET "/produtos/{id}" (id int) Response effects(db)
  on-err msg
    json-error(400, msg)
  end
  let item = try buscar_produto_por_id(id)
  json-response(200, try json-encode(item))
end

// Rota POST com corpo JSON e guarda de autenticação
route POST "/produtos" (req Request) Response effects(db, clock)
  on-err msg
    json-error(401, msg)
  end
  // Guarda declarativa JWT (retorna 401 automático se inválido/expirado)
  let claims = try auth-guard-jwt(req, "segredo-militar-32-bytes-minimo!")
  let novo = try json-decode(request-body(req), Produto)
  try salvar_produto(novo)
  json-response(201, "{\"ok\":true}")
end
```

---

## 4. WebSockets em Tempo Real (`ws-route`)

```liaf
ws-route "/ws/chat" (req Request, conn WSConn) effects(io, net)
  on-open
    ws-join(conn, "sala_geral")
    ws-send(conn, "Conectado ao chat")
  end

  on-message texto
    try ws-broadcast("sala_geral", texto)
  end

  on-close
    println("Conexão encerrada pelo cliente")
  end
end
```

---

## 5. Estruturas de Controle

### 5.1 Condicionais (`if` / `else`)
```liaf
if total > 100 && cliente_vip
  println("Desconto de 20% aplicado")
else if total > 50
  println("Desconto de 10% aplicado")
else
  println("Sem desconto")
end
```

### 5.2 Pattern Matching (`match`)
```liaf
// 1. Match em Enum (checagem de exaustividade estrita)
match pedido.status
  case StatusPedido.Pendente
    "Aguardando confirmação"
  case StatusPedido.Processando
    "Preparando pacote"
  case StatusPedido.Entregue
    "Finalizado"
  case StatusPedido.Cancelado
    "Cancelado"
end

// 2. Match em Result (ok / err)
match list-get(itens, 0)
  case ok primeiro
    println(primeiro)
  case err _
    println("Lista vazia")
end
```

### 5.3 Loops (`while`, `for-range`, `for-each`)
```liaf
// While
while contador < 10
  set contador = contador + 1
end

// For-range (exclusivo no limite superior: 0 até 9)
for-range i 0 10
  println(i)
end

// For-each em coleções
for-each item lista_de_produtos
  println(item.nome)
end
```

---

## 6. Operadores Infixos e Precedência

| Operador | Tipo | Descrição |
|---|---|---|
| `+`, `-`, `*`, `/` | Aritmético | Operações matemáticas naturais |
| `==`, `!=`, `<`, `<=`, `>`, `>=` | Comparação | Comparações escalares |
| `&&`, `||`, `!` | Lógico | Operadores booleanos de curto-circuito |
| `objeto.campo` | Acesso | Acesso a campos e encadeamento pós-fixo (`f().campo`) |
| `<-canal` | Concorrência | Leitura de canal Go embutido |

---

## 7. Ponto de Entrada Standalone / CLI (`main`)

Para programas não-web (utilitários de terminal, scripts de dados, workers locais):

```liaf
module main

import "std/cli"
import "std/fs"

fn main() void effects(io, fs)
  let args = cli-args-list()
  if (cli-has-flag "--ajuda")
    println("Uso: app --entrada=arquivo.txt")
    cli-exit(0)
  end
  let entrada = cli-get-flag "--entrada" "dados.txt"
  println(fmt("Processando {}", entrada))
end
```
