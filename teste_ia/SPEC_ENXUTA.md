# Especificação Enxuta — LIAF v0.3 / v0.4

## 1. Módulo
Todo arquivo `.liaf` inicia com a declaração do módulo:
```liaf
(module meu_modulo
  ;; declarações de structs, funções e rotas aqui
)
```

## 2. Structs
São imutáveis.
```liaf
(struct Usuario
  (fields
    (id int)
    (nome str)
    (ativo bool)))
```
- Criar instância: `(new Usuario 1 "Gabriel" true)`
- Acessar campo: `(field u nome)`

## 3. Variáveis Locais (`let` e `set`)
**ATENÇÃO CRÍTICA:** `let` SEMPRE exige 3 partes: `(let <nome> <tipo> <valor>)`. NUNCA omita o tipo!
```liaf
(let x int 10)
(let nome str "Ana")
(let lista (list int) (make-list int))
(let usuario Usuario (new Usuario 1 "Ana" true))

;; Para alterar o valor de uma variável já criada:
(set x 20)
```

## 4. Funções
Forma canônica:
```liaf
(fn somar (params (a int) (b int)) (returns int) (effects)
  (body
    (return (add a b))))
```

Com efeito de I/O:
```liaf
(fn saudar (params (nome str)) (returns void) (effects io)
  (body
    (println (concat "Olá, " nome))))
```

Com tratamento de erro (`try` + `on-err`):
```liaf
(fn carregar (params (caminho str)) (returns str) (effects fs)
  (on-err err (return "erro ao ler"))
  (body
    (let conteudo str (try (fs-read-file caminho)))
    (return conteudo)))
```

## 5. Coleções

### Listas
```liaf
(let numeros (list int) (make-list int))
(list-push numeros 10)
(let tam int (list-len numeros))

(match (list-get numeros 0)
  (ok val (println (str-from-int val)))
  (err msg (println msg)))
```

### Iteração: `for-each`
```liaf
(for-each item lista
  (println item))
```

## 6. Bancos de Dados Nativos (PostgreSQL / MySQL / Redis)
Suporte nativo com efeito `db` e tipo de conexão `DBConnection`:

### Regras do SQL:
1. **O SQL DEVE ser uma string literal**. Não use `concat` para montar SQL (proteção contra SQL injection).
2. **Parâmetros usam `$1, $2` (Postgres) ou `?` (MySQL)**: os argumentos são passados após a string SQL.

### Conexão e Execução:
```liaf
;; Executar comando DDL ou DML:
(fn criar-tabela (params (db DBConnection)) (returns (result bool str)) (effects db)
  (body
    (try (db-exec db "CREATE TABLE IF NOT EXISTS users (id int PRIMARY KEY, name text NOT NULL)"))
    (return (ok true))))

;; Inserir com parâmetros:
(fn inserir-usuario (params (db DBConnection) (id int) (nome str)) (returns (result bool str)) (effects db)
  (body
    (try (db-exec db "INSERT INTO users (id, name) VALUES ($1, $2)" id nome))
    (return (ok true))))

;; Consultar e converter linhas em structs:
(fn listar-usuarios (params (db DBConnection)) (returns (result (list User) str)) (effects db)
  (body
    (let usuarios (list User) (try (db-query db "SELECT id, name FROM users" User)))
    (return (ok usuarios))))

;; Transação atômica (commit automático no fim, rollback automático em erro/try):
(fn salvar-duplo (params (db DBConnection)) (returns (result bool str)) (effects db)
  (body
    (db-transaction db
      (try (db-exec db "INSERT INTO users (id, name) VALUES ($1, $2)" 1 "A"))
      (try (db-exec db "INSERT INTO users (id, name) VALUES ($1, $2)" 2 "B")))
    (return (ok true))))
```

## 7. Efeitos Disponíveis
- `io`: Para `println` e saídas de console.
- `fs`: Para leitura/escrita em arquivos.
- `db`: Para operações em bancos de dados (`db-exec`, `db-query`, `db-transaction`, `redis-get`, etc.).
- `(effects)`: Declare vazio se a função for pura. **Não declare efeitos que a função não usa!**
