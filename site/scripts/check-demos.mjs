import { access } from 'node:fs/promises';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const publicRoot = join(dirname(fileURLToPath(import.meta.url)), '..', 'public');
const required = [
  'demos/join.webp',
  'demos/repo-picker.webp',
];

export async function findMissingDemos(root = publicRoot) {
  const missing = [];
  for (const path of required) {
    try {
      await access(join(root, path));
    } catch {
      missing.push(path);
    }
  }
  return missing;
}

if (process.argv[1] === fileURLToPath(import.meta.url)) {
  const missing = await findMissingDemos();
  for (const path of missing) console.error(`missing demo asset: ${path}`);
  if (missing.length) process.exitCode = 1;
}
