# Benchmark reproduzível

Requer Python 3.11+ e o `liafc` recém-compilado. Sem dependências Python externas.

```powershell
py -3 benchmarks/run_suite.py --dry-run
py -3 -m unittest discover -s benchmarks -p test_*.py
py -3 benchmarks/run_suite.py --provider ollama --model MODELO_INSTALADO --execute --output benchmarks/results/experimento-1
```

Os adaptadores OpenAI e Anthropic usam respectivamente `OPENAI_API_KEY` e `ANTHROPIC_API_KEY`; não leem `.env` automaticamente. Modelo, preços e limite de chamadas são explícitos. `--input-price`/`--output-price` são USD por milhão de tokens. Sem preços/usage, custos ficam `null`, nunca inventados. Preços calculados não incluem infraestrutura, cache ou impostos.

O runner registra prompts por hash, fontes de cada tentativa, usage, feedback, tempo de geração, compilação e execução, taxa de sucesso, pass@1 e custo até sucesso. Cada experimento precisa de uma pasta nova para preservar os dados brutos. Testes sem execução não contam como aprovação comportamental. Use isolamento apropriado para executar código produzido por modelos: o diretório temporário e o timeout não são um sandbox de sistema operacional.

`tasks.json` é uma suíte inicial de núcleo (loops, coleções e JSON). Não equivale aos três cenários de serviço completos propostos na issue #009. A medição comparativa com modelos reais, memória RSS, cenários HTTP/concorrência/hot-reload e testes ocultos continuam necessários antes de alegar superioridade AI-first.

Protocolos consultados: [OpenAI Responses](https://developers.openai.com/api/docs/guides/text), [Anthropic Messages](https://platform.claude.com/docs/en/api/messages/create), [Ollama Chat](https://docs.ollama.com/api/chat).
