import assert from 'node:assert/strict';
import { mkdtemp, mkdir, readFile, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { dirname, join } from 'node:path';
import test from 'node:test';
import { fileURLToPath } from 'node:url';

import { findMissingDemos } from './check-demos.mjs';

const demosRoot = join(dirname(fileURLToPath(import.meta.url)), '..', 'public', 'demos');

async function readAnimationBackground(path) {
  const data = await readFile(path);
  for (let offset = 12; offset + 8 <= data.length;) {
    const size = data.readUInt32LE(offset + 4);
    if (data.toString('ascii', offset, offset + 4) === 'ANIM') {
      return [...data.subarray(offset + 8, offset + 12)];
    }
    offset += 8 + size + (size % 2);
  }
  throw new Error(`${path} has no ANIM chunk`);
}

test('demo animations use the terminal background', async () => {
  for (const name of ['join.webp', 'repo-picker.webp']) {
    assert.deepEqual(await readAnimationBackground(join(demosRoot, name)), [22, 22, 22, 255]);
  }
});

test('reports all demo outputs missing from an empty directory', async () => {
  const root = await mkdtemp(join(tmpdir(), 'leo-docs-demos-'));
  try {
    assert.deepEqual(await findMissingDemos(root), [
      'demos/join.webp',
      'demos/repo-picker.webp',
    ]);
  } finally {
    await rm(root, { recursive: true, force: true });
  }
});

test('reports missing demo outputs', async () => {
  const root = await mkdtemp(join(tmpdir(), 'leo-docs-demos-'));
  try {
    await mkdir(join(root, 'demos'), { recursive: true });
    await writeFile(join(root, 'demos', 'repo-picker.webp'), 'image');
    assert.deepEqual(await findMissingDemos(root), ['demos/join.webp']);
  } finally {
    await rm(root, { recursive: true, force: true });
  }
});
