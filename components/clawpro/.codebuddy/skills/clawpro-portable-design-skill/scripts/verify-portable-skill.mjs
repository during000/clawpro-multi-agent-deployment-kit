#!/usr/bin/env node
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const packRoot = path.resolve(__dirname, '..');
const manifestPath = path.resolve(packRoot, 'MANIFEST.json');

const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));

const packGroups = [
  ...manifest.rootFiles,
  ...(manifest.docs ?? []),
  ...Object.values(manifest.entry),
  ...manifest.references,
  ...manifest.componentSpecs,
  ...manifest.portableExamples,
  ...manifest.tokens,
  ...manifest.qa,
  ...manifest.assets,
  ...manifest.scripts,
];

const missing = packGroups.filter((relativePath) => !fs.existsSync(path.resolve(packRoot, relativePath)));
const duplicateCandidates = [
  ...manifest.rootFiles,
  ...(manifest.docs ?? []),
  ...manifest.references,
  ...manifest.componentSpecs,
  ...manifest.portableExamples,
  ...manifest.tokens,
  ...manifest.qa,
  ...manifest.assets,
  ...manifest.scripts,
];
const duplicates = duplicateCandidates.filter((relativePath, index) => duplicateCandidates.indexOf(relativePath) !== index);

function collectFiles(relativeDir, predicate = () => true) {
  const absoluteDir = path.resolve(packRoot, relativeDir);
  const result = [];

  function walk(currentDir) {
    for (const entry of fs.readdirSync(currentDir, { withFileTypes: true })) {
      if (entry.name === '.DS_Store') continue;
      const absolutePath = path.resolve(currentDir, entry.name);
      if (entry.isDirectory()) {
        walk(absolutePath);
        continue;
      }
      const relativePath = path.relative(packRoot, absolutePath).split(path.sep).join('/');
      if (predicate(relativePath)) result.push(relativePath);
    }
  }

  walk(absoluteDir);
  return result.sort();
}

const manifestRootFiles = new Set(manifest.rootFiles);
const actualRootFiles = fs
  .readdirSync(packRoot, { withFileTypes: true })
  .filter((entry) => entry.isFile() && entry.name !== '.DS_Store')
  .map((entry) => entry.name)
  .sort();
const unlistedRootFiles = actualRootFiles.filter((relativePath) => !manifestRootFiles.has(relativePath));

const manifestComponentSpecs = new Set(manifest.componentSpecs);
const actualComponentSpecs = collectFiles('component-specs', (relativePath) => relativePath.endsWith('.md'));
const unlistedComponentSpecs = actualComponentSpecs.filter((relativePath) => !manifestComponentSpecs.has(relativePath));

const manifestPortableExamples = new Set(manifest.portableExamples);
const actualPortableExamples = collectFiles('portable');
const unlistedPortableExamples = actualPortableExamples.filter((relativePath) => !manifestPortableExamples.has(relativePath));

if (
  missing.length > 0 ||
  duplicates.length > 0 ||
  unlistedRootFiles.length > 0 ||
  unlistedComponentSpecs.length > 0 ||
  unlistedPortableExamples.length > 0
) {
  console.error('Portable skill verification failed.');

  if (missing.length > 0) {
    console.error('\nMissing pack files:');
    for (const file of missing) console.error(`- ${file}`);
  }

  if (duplicates.length > 0) {
    console.error('\nDuplicate MANIFEST entries:');
    for (const file of [...new Set(duplicates)]) console.error(`- ${file}`);
  }

  if (unlistedRootFiles.length > 0) {
    console.error('\nRoot files not listed in MANIFEST.json rootFiles:');
    for (const file of unlistedRootFiles) console.error(`- ${file}`);
  }

  if (unlistedComponentSpecs.length > 0) {
    console.error('\ncomponent-specs files not listed in MANIFEST.json:');
    for (const file of unlistedComponentSpecs) console.error(`- ${file}`);
  }

  if (unlistedPortableExamples.length > 0) {
    console.error('\nportable files not listed in MANIFEST.json:');
    for (const file of unlistedPortableExamples) console.error(`- ${file}`);
  }

  process.exit(1);
}

console.log('Portable skill verification passed.');
