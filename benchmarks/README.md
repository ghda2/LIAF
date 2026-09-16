# Benchmark reproduzível

Requer Python 3.11+ e o `liafc` recém-compilado. Sem dependências Python externas.

```powershell
py -3 benchmarks/run_suite.py --dry-run
py -3 -m unittest discover -s benchmarks -p test_*.py
py -3 benchmarks/run_suite.py --provider ollama --model MODELO_INSTALADO --execute --output benchmarks/results/experimento-1
```

Os adaptadores OpenAI, Anthropic e Gemini usam respectivamente `OPENAI_API_KEY`, `ANTHROPIC_API_KEY` e `GEMINI_API_KEY` (ou `GOOGLE_API_KEY`). O runner carrega o `.env` da raiz sem substituir variáveis já definidas no ambiente. Modelo, preços e limite de chamadas são explícitos. `--input-price`/`--output-price` são USD por milhão de tokens. Sem preços/usage, custos ficam `null`, nunca inventados. Preços calculados não incluem infraestrutura, cache ou impostos.

Para diagnosticar falhas, acrescente `-v` ou `--verbose`: mostra comando, código de retorno, duração, stdout/stderr e saída esperada por etapa (`check`, `compile`, `run`). Mesmo sem `-v`, cada tentativa avaliada salva um arquivo `<fonte>.json` junto ao código gerado, antes de iniciar a próxima tentativa, e os detalhes também ficam no `history` de `records.jsonl`. Em caso de falha, o terminal indica a etapa e os caminhos da fonte e do log. Timeouts preservam a saída parcial. Use uma pasta `--output` nova por rodada; `--overwrite` substitui registros existentes e pode deixar fontes de tentativas antigas na pasta. Consulte o `history` para identificar as tentativas da rodada atual.

O runner registra prompts por hash, fontes de cada tentativa, usage, feedback, tempo de geração, compilação e execução, taxa de sucesso, pass@1 e custo até sucesso. Cada experimento precisa de uma pasta nova para preservar os dados brutos. Testes sem execução não contam como aprovação comportamental. Use isolamento apropriado para executar código produzido por modelos: o diretório temporário e o timeout não são um sandbox de sistema operacional.

`tasks.json` é uma suíte inicial de núcleo (loops, coleções e JSON). Não equivale aos três cenários de serviço completos propostos na issue #009. A medição comparativa com modelos reais, memória RSS, cenários HTTP/concorrência/hot-reload e testes ocultos continuam necessários antes de alegar superioridade AI-first.

## API de tarefas com persistência

`tasks-http.json` adiciona um cenário separado de integração. O modelo recebe o contrato HTTP e, para LIAF, a referência de recursos `http-context.md`. O runner compila o código gerado, inicia o servidor em uma porta local disponível e verifica CRUD, validação de entradas, IDs inexistentes, escape de títulos com Unicode/aspas, persistência de criação/edição/exclusão e continuidade dos IDs após dois reinícios. Os testes usam requisições sequenciais e param na primeira divergência; casos posteriores ficam sem avaliação. Cada tentativa usa dados novos em uma pasta temporária, preservada apenas entre os reinícios daquela tentativa. Os processos são encerrados ao final, inclusive em falhas e timeouts.

```powershell
py benchmarks/run_suite.py --provider gemini --model gemini-2.5-flash --language liaf --suite benchmarks/tasks-http.json --execute --max-tokens 8192 --max-calls 3 --timeout 90 --output benchmarks/results/task-api-experimento-1 -v
```

Os logs incluem as requisições, respostas, expectativas e stdout/stderr do servidor. `suite.json` e `<tarefa>-system.txt` preservam o contrato e o contexto enviados nesta rodada. O Gemini agora respeita `--max-tokens`; use 8192 neste cenário maior. `--timeout` limita chamadas ao provedor e compilação; a inicialização HTTP aguarda no máximo 15 segundos e cada requisição no máximo 5 segundos (ou o timeout menor configurado).

A referência Python em `fixtures/task_api.py` serve apenas para testar o avaliador, não é enviada ao modelo. Os testes do avaliador também verificam a rejeição de versões que perdem persistência, reutilizam IDs ou aceitam entradas inválidas. Este cenário não mede concorrência, autenticação, recuperação de arquivos corrompidos, carga, nem garante prontidão para produção. As respostas 500 em falhas de armazenamento fazem parte do contrato solicitado, mas ainda não há injeção dessas falhas no avaliador.

> **Nota (16/09/2026):** o diretório `results/` foi removido localmente e não faz parte do
> repositório. As rodadas descritas abaixo registram o que foi observado na ocasião, mas os
> dados brutos não são mais verificáveis a partir daqui. Medições futuras usam o contexto
> v0.3 de `http-context.md` e não são comparáveis com estas.

Rodada de 2026-09-16, `gemini-2.5-flash`, 8192 tokens de saída, até 3 tentativas: **nenhuma tentativa passou no checker**. A primeira usou `let` no topo do módulo (`E_INVALID_TOPLEVEL`); a segunda e a terceira usaram a expressão não suportada `not` (`E_INVALID_EXPR`). Portanto, nenhum caso HTTP foi executado contra a implementação do modelo. O resultado é uma falha da geração integrada nesta rodada, não uma medição do comportamento do servidor. Fontes, feedback e contexto estão em `results/gemini-flash-task-api-20260916-01/`. O código gerado foi mantido sem correções manuais para preservar a medição. Os testes locais do avaliador passaram (15 testes).

Implementação de referência em LIAF: `task_api.liaf` (sintaxe v0.2, removido da raiz em 16/09/2026). Passou no checker, compilou e passou nos 26 casos HTTP do mesmo avaliador, incluindo dois reinícios. A referência atual, já em v0.3 e com rotas declarativas, é [`../examples/task_api_v03.liaf`](../examples/task_api_v03.liaf), coberta por `TestExecuteV03TaskAPI`. O registro está em `results/reference-task-api-20260916-01/validation.json`. Essa implementação foi escrita e validada separadamente e não conta como sucesso do Gemini. Usa requisições sequenciais e escrita direta de arquivo, sem garantias de concorrência ou recuperação de gravações interrompidas.

Protocolos consultados: [OpenAI Responses](https://developers.openai.com/api/docs/guides/text), [Anthropic Messages](https://platform.claude.com/docs/en/api/messages/create), [Ollama Chat](https://docs.ollama.com/api/chat).
