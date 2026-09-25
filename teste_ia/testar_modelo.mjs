import fs from 'fs';
import path from 'path';
import { execSync } from 'child_process';
import { fileURLToPath } from 'url';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

const useGemini = process.argv.includes('--gemini');
const modelArg = process.argv.find(a => a.startsWith('--model='));
const ollamaModel = modelArg ? modelArg.split('=')[1] : 'qwen2.5-coder:7b';

// Carregar .env da raiz ou da pasta atual caso use Gemini
function getApiKey() {
  if (process.env.GEMINI_API_KEY) return process.env.GEMINI_API_KEY;
  const envPaths = [
    path.join(__dirname, '.env'),
    path.join(__dirname, '..', '.env')
  ];
  for (const envPath of envPaths) {
    if (fs.existsSync(envPath)) {
      const content = fs.readFileSync(envPath, 'utf8');
      for (const line of content.split('\n')) {
        const match = line.trim().match(/^GEMINI_API_KEY=(.+)$/);
        if (match) return match[1].trim();
      }
    }
  }
  return null;
}

let API_KEY = null;
if (useGemini) {
  API_KEY = getApiKey();
  if (!API_KEY) {
    console.error("ERRO: GEMINI_API_KEY não encontrada no .env!");
    process.exit(1);
  }
}

// Arquivos de contexto
const spec = fs.readFileSync(path.join(__dirname, 'SPEC_ENXUTA.md'), 'utf8');
const exemplos = fs.readFileSync(path.join(__dirname, 'exemplos.liaf'), 'utf8');
const desafio = fs.readFileSync(path.join(__dirname, 'DESAFIO.md'), 'utf8');

const systemInstruction = `Você é um gerador de código para a linguagem LIAF (Language for AI First).
Sua resposta DEVE conter APENAS o código LIAF válido, sem explicações, dentro de um bloco de código markdown:
\`\`\`liaf
(module solucao
  ...
)
\`\`\`

Regras fundamentais:
- Sintaxe estrita S-expressions com parênteses.
- Toda função declara (params ...), (returns ...) e (effects ...).
- Se a função for pura, declare (effects). Se usar println, declare (effects io).
- Variáveis locais usam (let nome tipo valor) e mutações usam (set nome valor).
- Trate erros ou evite chamadas falíveis não tratadas.`;

async function callOllama(messages, model) {
  const url = 'http://localhost:11434/api/chat';
  const chatMessages = [
    { role: 'system', content: systemInstruction },
    ...messages.map(m => ({
      role: m.role === 'model' ? 'assistant' : m.role,
      content: m.text
    }))
  ];

  const res = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      model,
      messages: chatMessages,
      stream: false,
      options: {
        temperature: 0.2
      }
    })
  });

  if (!res.ok) {
    const errText = await res.text();
    throw new Error(`Erro na API do Ollama (${res.status}): ${errText}`);
  }

  const data = await res.json();
  return data.message?.content || '';
}

async function callGemini(messages) {
  const url = `https://generativelanguage.googleapis.com/v1beta/models/gemini-2.5-flash-lite:generateContent?key=${API_KEY}`;
  
  const contents = messages.map(m => ({
    role: m.role === 'user' ? 'user' : 'model',
    parts: [{ text: m.text }]
  }));

  const res = await fetch(url, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({
      contents,
      systemInstruction: { parts: [{ text: systemInstruction }] },
      generationConfig: {
        temperature: 0.2,
      }
    })
  });

  if (!res.ok) {
    const errText = await res.text();
    throw new Error(`Erro na API Gemini (${res.status}): ${errText}`);
  }

  const data = await res.json();
  const text = data.candidates?.[0]?.content?.parts?.[0]?.text || '';
  return text;
}

function extractLiafCode(rawText) {
  const match = rawText.match(/```(?:liaf)?\s*([\s\S]*?)```/i);
  if (match) {
    return match[1].trim();
  }
  return rawText.trim();
}

function checkLiaf() {
  try {
    const output = execSync('..\\liafc.exe check solucao.liaf --json', {
      cwd: __dirname,
      encoding: 'utf8',
      stdio: ['pipe', 'pipe', 'pipe']
    });
    return JSON.parse(output);
  } catch (err) {
    if (err.stdout) {
      try {
        return JSON.parse(err.stdout);
      } catch (_) {}
    }
    return { status: 'error', raw: err.message };
  }
}

async function main() {
  const provider = useGemini ? 'Gemini Flash' : `Ollama (${ollamaModel})`;
  console.log(`=== INICIANDO TESTE COM IA: ${provider} ===`);
  console.log("1. Carregando especificações e desafio...");

  const initialPrompt = `Aqui está a especificação da linguagem LIAF:
---
${spec}
---
Exemplo de código canônico válido:
---
\`\`\`liaf
${exemplos}
\`\`\`
---

Seu desafio é:
${desafio}

Escreva o código completo para o módulo solucao.`;

  const conversation = [
    { role: 'user', text: initialPrompt }
  ];

  const maxAttempts = 5;
  for (let attempt = 1; attempt <= maxAttempts; attempt++) {
    console.log(`\n--- Tentativa ${attempt} de ${maxAttempts} ---`);
    console.log("Solicitando código ao modelo...");
    
    const startTime = Date.now();
    const responseText = useGemini ? await callGemini(conversation) : await callOllama(conversation, ollamaModel);
    const durationSec = ((Date.now() - startTime) / 1000).toFixed(1);
    console.log(`Resposta recebida em ${durationSec}s.`);

    const code = extractLiafCode(responseText);

    const solucaoPath = path.join(__dirname, 'solucao.liaf');
    fs.writeFileSync(solucaoPath, code, 'utf8');
    console.log("Código salvo em solucao.liaf. Validando com liafc...");

    const checkResult = checkLiaf();
    if (checkResult.status === 'success') {
      console.log(`\n✅ SUCESSO! O modelo gerou código LIAF 100% válido na tentativa ${attempt}!`);
      console.log("\nCódigo final gerado:\n" + code);
      return;
    }

    console.log("❌ O compilador rejeitou o código. Erros encontrados:");
    console.log(JSON.stringify(checkResult, null, 2));

    if (attempt < maxAttempts) {
      console.log("\nEnviando feedback do compilador para auto-cura...");
      conversation.push({ role: 'model', text: responseText });
      conversation.push({
        role: 'user',
        text: `O compilador LIAF retornou o seguinte erro de compilação:\n${JSON.stringify(checkResult, null, 2)}\n\nPor favor, corrija o código de acordo com o erro e gere o arquivo solucao.liaf novamente.`
      });
    }
  }

  console.log("\n❌ O modelo atingiu o limite de tentativas sem convergir.");
}

main().catch(err => {
  console.error("Erro fatal:", err.message);
});
