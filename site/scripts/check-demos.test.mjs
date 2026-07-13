import assert from 'node:assert/strict';
import { mkdtemp, mkdir, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import test from 'node:test';

import { findMissingDemos } from './check-demos.mjs';

test('reports missing demo outputs', async () => {
  const root = await mkdtemp(join(tmpdir(), 'leo-docs-demos-'));
  try {
    await mkdir(join(root, 'demos'), { recursive: true });
    await writeFile(join(root, 'demos', 'repo-picker.webm'), 'video');
    assert.deepEqual(await findMissingDemos(root), [
      'demos/join.png',
      'demos/join.webm',
      'demos/repo-picker.png',
    ]);
  } finally {
    await rm(root, { recursive: true, force: true });
  }
});
