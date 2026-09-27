# Referência de Sintaxe da LIAF

## 1. Sintaxe Linear (LIAF v0.6 — Canônica)

### 1.1 Declaração de Tipos (`struct`)
```liaf
struct Usuario
  id int
  nome str
  email str
  ativo bool
  tags (list str)
end
```

### 1.2 Construtor de Struct
```liaf
let u = new Usuario(1, "Gabriel", "gabriel@exemplo.com", true, make-list(str))
```

### 1.3 Funções (`fn`)
```liaf
// Pura com tipos explícitos e retorno implícito
fn calcular_total(preco float, qtd int) float
  preco * qtd
end

// Função com efeitos e sem retorno (void)
fn notificar(msg str) void effects(io)
  println(fmt("Notificação: {}", msg))
end
```

### 1.4 Rotas HTTP Declarativas (`route`)
```liaf
route GET "/itens/{id}" (id int) Response effects(db)
  on-err err_msg
    json-response(500, err_msg)
  end
  let item = try buscar_item_no_banco(id)
  json-response(200, try json-encode(item))
end
```

### 1.5 WebSocket (`ws-route`)
```liaf
ws-route "/ws/canal" (req Request, conn WSConn) effects(io, net)
  on-open
    ws-join(conn, "sala_principal")
  end

  on-message texto
    try ws-broadcast("sala_principal", texto)
  end

  on-close
    println("Conexão encerrada")
  end
end
```

### 1.6 Estruturas de Controle
```liaf
// Condicional if / else
if total > 100
  println("Frete grátis")
else
  println("Cobrar frete")
end

// Loop while
while contador < limite
  set contador = contador + 1
end

// Loop for-range (exclusivo no limite superior)
for-range i 0 10
  println(i)
end

// Loop for-each
for-each item lista_de_itens
  println(item.nome)
end
```

---

## 2. Tabela de Equivalência: v0.5 S-expr vs v0.6 Linear

| Conceito | LIAF v0.5 (S-expression) | LIAF v0.6 (Linear) |
|---|---|---|
| Struct | `(struct Item (id int) (nome str))` | `struct Item \n id int \n nome str \n end` |
| Função | `(fn soma ((a int) (b int)) int (add a b))` | `fn soma(a int, b int) int \n a + b \n end` |
| Rota | `(route GET "/x" () Response (texto 200 "ok"))` | `route GET "/x" () Response \n texto(200, "ok") \n end` |
| Let | `(let x int 10)` | `let x = 10` ou `let x int = 10` |
| Operadores | `(add a (mul b c))` | `a + b * c` |
| Comentários | `; comentário` | `// comentário` ou `; comentário` |
