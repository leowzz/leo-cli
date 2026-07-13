import assert from 'node:assert/strict';
import { mkdtemp, mkdir, rm, writeFile } from 'node:fs/promises';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import test from 'node:test';

import { findLocaleMismatches } from './check-locales.mjs';

test('reports pages missing from either locale', async () => {
  const root = await mkdtemp(join(tmpdir(), 'leo-docs-locales-'));
  try {
    await mkdir(join(root, 'guides'), { recursive: true });
    await mkdir(join(root, 'en'), { recursive: true });
    await writeFile(join(root, 'index.md'), '# Chinese');
    await writeFile(join(root, 'guides', 'join.mdx'), '# Join');
    await writeFile(join(root, 'en', 'index.md'), '# English');
    await writeFile(join(root, 'en', 'only-english.md'), '# English only');

    assert.deepEqual(await findLocaleMismatches(root), {
      missingEnglish: ['guides/join.mdx'],
      missingChinese: ['only-english.md'],
    });
  } finally {
    await rm(root, { recursive: true, force: true });
  }
});
