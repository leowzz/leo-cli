import { readdir } from 'node:fs/promises';
import { dirname, join, relative, sep } from 'node:path';
import { fileURLToPath } from 'node:url';

const contentRoot = join(dirname(fileURLToPath(import.meta.url)), '..', 'src', 'content', 'docs');

async function collectPages(root, skipEnglishRoot = false, current = root) {
  const pages = [];
  for (const entry of await readdir(current, { withFileTypes: true })) {
    if (skipEnglishRoot && current === root && entry.name === 'en') continue;
    const path = join(current, entry.name);
    if (entry.isDirectory()) {
      pages.push(...await collectPages(root, false, path));
    } else if (/\.mdx?$/.test(entry.name)) {
      pages.push(relative(root, path).split(sep).join('/'));
    }
  }
  return pages.sort();
}

export async function findLocaleMismatches(root = contentRoot) {
  const chinese = new Set(await collectPages(root, true));
  const english = new Set(await collectPages(join(root, 'en')));
  return {
    missingEnglish: [...chinese].filter((page) => !english.has(page)),
    missingChinese: [...english].filter((page) => !chinese.has(page)),
  };
}

export async function checkLocales(root = contentRoot) {
  const missing = await findLocaleMismatches(root);
  if (missing.missingEnglish.length || missing.missingChinese.length) {
    for (const page of missing.missingEnglish) console.error(`missing English page: en/${page}`);
    for (const page of missing.missingChinese) console.error(`missing Chinese page: ${page}`);
    return false;
  }
  return true;
}

if (process.argv[1] === fileURLToPath(import.meta.url) && !await checkLocales()) {
  process.exitCode = 1;
}
