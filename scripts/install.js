const fs = require('fs');
const path = require('path');
const https = require('https');

const REPO = 'ghda2/LIAF';
const pkg = require('../package.json');
const VERSION = pkg.version;

const PLATFORM_MAP = {
  win32: 'windows',
  darwin: 'darwin',
  linux: 'linux'
};

const ARCH_MAP = {
  x64: 'amd64',
  arm64: 'arm64'
};

const os = PLATFORM_MAP[process.platform];
const arch = ARCH_MAP[process.arch];

if (!os || !arch) {
  console.error(`[liaf] Plataforma não suportada: ${process.platform} (${process.arch})`);
  process.exit(1);
}

const ext = os === 'windows' ? '.exe' : '';
const binaryFileName = `liafc-${os}-${arch}${ext}`;
const destName = os === 'windows' ? 'liafc.exe' : 'liafc';
const binDir = path.join(__dirname, '..', 'bin');
const destPath = path.join(binDir, destName);

if (!fs.existsSync(binDir)) {
  fs.mkdirSync(binDir, { recursive: true });
}

// Se já houver binário local (ex: durante dev), não baixa
if (fs.existsSync(destPath)) {
  console.log(`[liaf] Binário nativo já presente em: ${destPath}`);
  process.exit(0);
}

const url = `https://github.com/${REPO}/releases/download/v${VERSION}/${binaryFileName}`;

console.log(`[liaf] Baixando binário v${VERSION} para ${os}/${arch}...`);

function downloadFile(targetUrl, destination, callback) {
  https.get(targetUrl, (res) => {
    if (res.statusCode >= 300 && res.statusCode < 400 && res.headers.location) {
      return downloadFile(res.headers.location, destination, callback);
    }

    if (res.statusCode !== 200) {
      return callback(new Error(`Falha no download (HTTP ${res.statusCode}): ${targetUrl}`));
    }

    const fileStream = fs.createWriteStream(destination);
    res.pipe(fileStream);

    fileStream.on('finish', () => {
      fileStream.close(() => callback(null));
    });

    fileStream.on('error', (err) => {
      fs.unlink(destination, () => callback(err));
    });
  }).on('error', (err) => {
    callback(err);
  });
}

downloadFile(url, destPath, (err) => {
  if (err) {
    console.warn(`[liaf] Não foi possível baixar da release: ${err.message}`);
    console.warn('[liaf] Se estiver em ambiente local de desenvolvimento, compile com: go build -o bin/liafc ./cmd/liafc');
    process.exit(0);
  }

  if (os !== 'windows') {
    fs.chmodSync(destPath, 0o755);
  }

  console.log(`[liaf] Binário instalado com sucesso em ${destPath}`);
});
