/* eslint-disable no-console */
const fs = require('node:fs');
const path = require('node:path');
const os = require('node:os');
const https = require('node:https');
const { pipeline } = require('node:stream/promises');
const { createWriteStream } = require('node:fs');
const tar = require('tar');
const AdmZip = require('adm-zip');

const pkg = require('../../package.json');

const owner = 'anchorbrowser';
const repo = 'cli';
const version = pkg.version;

const platformMap = {
  darwin: 'darwin',
  linux: 'linux',
  win32: 'windows',
};

const archMap = {
  x64: 'amd64',
  arm64: 'arm64',
};

async function main() {
  const platform = platformMap[process.platform];
  const arch = archMap[process.arch];

  if (!platform || !arch) {
    console.warn(`Skipping AnchorBrowser binary install for unsupported target ${process.platform}/${process.arch}`);
    return;
  }

  const isWindows = platform === 'windows';
  const ext = isWindows ? 'zip' : 'tar.gz';
  const artifact = `anchorbrowser_${version}_${platform}_${arch}.${ext}`;
  const url = `https://github.com/${owner}/${repo}/releases/download/v${version}/${artifact}`;

  const root = path.resolve(__dirname, '..', '..');
  const outDir = path.join(root, 'npm', 'bin', 'native');
  const tmpDir = fs.mkdtempSync(path.join(os.tmpdir(), 'anchorbrowser-'));
  const archivePath = path.join(tmpDir, artifact);

  fs.mkdirSync(outDir, { recursive: true });

  try {
    console.log(`Downloading ${url}`);
    await download(url, archivePath);

    if (isWindows) {
      const zip = new AdmZip(archivePath);
      zip.extractAllTo(tmpDir, true);
    } else {
      await tar.x({ file: archivePath, cwd: tmpDir });
    }

    const binaryName = isWindows ? 'anchorbrowser.exe' : 'anchorbrowser';
    const extracted = path.join(tmpDir, binaryName);
    const target = path.join(outDir, binaryName);

    if (!fs.existsSync(extracted)) {
      throw new Error(`Expected binary not found in release artifact: ${binaryName}`);
    }

    fs.copyFileSync(extracted, target);
    if (!isWindows) {
      fs.chmodSync(target, 0o755);
    }

    console.log(`Installed ${binaryName}`);
  } catch (err) {
    console.error('Failed to install AnchorBrowser CLI binary:', err.message);
    console.error('You can still use Homebrew or download binaries from GitHub releases.');
    process.exitCode = 1;
  } finally {
    fs.rmSync(tmpDir, { recursive: true, force: true });
  }
}

function download(url, destination) {
  return new Promise((resolve, reject) => {
    const request = https.get(url, (response) => {
      if (response.statusCode >= 300 && response.statusCode < 400 && response.headers.location) {
        response.resume();
        download(response.headers.location, destination).then(resolve).catch(reject);
        return;
      }

      if (response.statusCode !== 200) {
        response.resume();
        reject(new Error(`Download failed with status ${response.statusCode}`));
        return;
      }

      pipeline(response, createWriteStream(destination)).then(resolve).catch(reject);
    });

    request.on('error', reject);
  });
}

main();
