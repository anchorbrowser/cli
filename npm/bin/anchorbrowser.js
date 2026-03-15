#!/usr/bin/env node

const { spawnSync } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');

const root = path.resolve(__dirname, '..', '..');
const binaryName = process.platform === 'win32' ? 'anchorbrowser.exe' : 'anchorbrowser';
const binaryPath = path.join(root, 'npm', 'bin', 'native', binaryName);

if (!fs.existsSync(binaryPath)) {
  console.error('anchorbrowser binary is not installed yet. Re-run: npm install @anchor-browser/cli');
  process.exit(1);
}

const child = spawnSync(binaryPath, process.argv.slice(2), { stdio: 'inherit' });
if (child.error) {
  console.error(child.error.message);
  process.exit(1);
}
process.exit(child.status === null ? 1 : child.status);
