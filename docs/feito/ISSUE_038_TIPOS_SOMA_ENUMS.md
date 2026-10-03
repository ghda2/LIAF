# Issue #038: Tipos Soma (Enums) e Pattern Matching Exaustivo

**Status:** Concluída  
**Componente:** `pkg/token`, `pkg/ast`, `pkg/parser`, `pkg/checker`, `pkg/codegen`, `docs/linguagem/SPEC.md`  
**Data:** 03 de outubro de 2026  

---

## 1. Contexto e Motivação

Atualmente, o controle de fluxo por desconstrução na LIAF (`match`) é limitado exclusivamente a tipos embutidos `result` (`ok`/`err`) e `option` (`some`/`none`).
Estados de domínio — como status de pedido (`Pendente`, `Pago`, `Cancelado`), eventos de fila ou categorias — precisam ser representados como strings soltas (`str`) ou inteiros mágicos (`int`).

Isso causa duas fragilidades severas para agentes de IA:
1. **Alucinação de valores:** O modelo pode digitar `"pendente"`, `"PENDING"` ou `"Pendente"`, gerando bugs silenciosos de comparação sem aviso do compilador.
2. **Falta de garantia de tratamento de casos:** Quando um novo estado é adicionado ao sistema, o modelo de IA esquece de tratar ramos em outros pontos da aplicação.

Com tipos soma (enums) e checagem de exaustividade, qualquer estado não tratado falha em tempo de compilação com `E_NONEXHAUSTIVE_MATCH`.

---

## 2. Proposta Técnica

### 2.1 Declaração de `enum` (Sintaxe Linear v0.6 e S-expressions)

Sintaxe Linear:
```liaf
enum Status
  Pendente
  Pago
  Cancelado
end
```

S-expression canônica:
```liaf
(enum Status
  Pendente
  Pago
  Cancelado)
```

Variantes com dados (sum types):
```liaf
enum Evento
  Desconectado
  Mensagem(texto str)
end
```

### 2.2 Construção e Acesso

Construção através do namespace do tipo:
```liaf
let s = Status.Pendente
let e = Evento.Mensagem("Olá")
```

### 2.3 Desconstrução com `match` Exaustivo

```liaf
match s
  case Status.Pendente
    "Aguardando"
  case Status.Pago
    "Confirmado"
  case Status.Cancelado
    "Cancelado"
end
```
Também aceitando a variante sem prefixo se inequívoca:
```liaf
match s
  case Pendente
    "Aguardando"
  case Pago
    "Confirmado"
  case Cancelado
    "Cancelado"
end
```
Ou com curinga padrão:
```liaf
match s
  case Status.Pago
    "Confirmado"
  case _
    "Outro"
end
```

### 2.4 Diagnósticos e Regras Estáticas

- `E_NONEXHAUSTIVE_MATCH`: Quando um `match` sobre um enum não cobre todas as variantes declaradas e não possui um ramo curinga (`_`).
- `E_UNKNOWN_VARIANT`: Tentativa de acessar uma variante inexistente (ex: `Status.Inexistente`).
- `E_DUPLICATE_CASE`: Mais de um ramo tratando a mesma variante no mesmo `match`.

---

## 3. Fora do Escopo

- Genéricos em enums (ex: `enum Option[T]`) — reservado para a Fase 5.
- Enums com métodos acoplados.

---

## 4. Casos de Aceitação e Conformidade

- `conformance/basics/038_enum_declaracao.liaf`: Declaração de enums e instanciação de variantes.
- `conformance/basics/038_enum_match_exaustivo.liaf`: `match` cobrindo todas as variantes como statement e expression.
- `conformance/basics/038_enum_match_wildcard.liaf`: `match` com ramo default `case _`.
- `conformance/basics/038_enum_nao_exaustivo_err.liaf`: Teste de rejeição estática com `E_NONEXHAUSTIVE_MATCH`.

---

## 5. Critério de Conclusão

1. Reconhecimento sintático completo no parser linear (`linear.go`) e no parser S-expr.
2. Checker registrando enums em `Info.Enums`, validando acesso a variantes e verificando exaustividade no `match`.
3. Codegen Go gerando tipos tipados e serializáveis em JSON.
4. Todos os testes de unidade e conformidade passando em `go test ./...`.
