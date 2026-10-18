#!/usr/bin/env node

const path = require('path');
const fs = require('fs');
const { spawnSync } = require('child_process');

const exeName = process.platform === 'win32' ? 'liafc.exe' : 'liafc';
const binPath = path.join(__dirname, exeName);

if (!fs.existsSync(binPath)) {
  console.error(`[liaf] Binário nativo não encontrado em: ${binPath}`);
  console.error('[liaf] Tente reinstalar o pacote ou execute: node scripts/install.js');
  process.exit(1);
}

const result = spawnSync(binPath, process.argv.slice(2), {
  stdio: 'inherit',
  shell: false
});

if (result.error) {
  console.error('[liaf] Erro ao executar o compilador:', result.error.message);
  process.exit(1);
}

process.exit(result.status !== null ? result.status : 0);
