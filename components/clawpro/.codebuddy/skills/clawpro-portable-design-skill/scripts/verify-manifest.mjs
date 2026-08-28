#!/usr/bin/env node
import fs from 'node:fs';
import path from 'node:path';
import { fileURLToPath } from 'node:url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);
const packageRoot = path.resolve(__dirname, '..');

// 读取 MANIFEST.json
const manifestPath = path.resolve(packageRoot, 'MANIFEST.json');
const manifest = JSON.parse(fs.readFileSync(manifestPath, 'utf8'));

console.log('\n=== MANIFEST Verification Report ===\n');

let allValid = true;
const errors = [];
const warnings = [];

// 验证 docs
console.log('Checking docs files...');
for (const docFile of manifest.docs || []) {
  const fullPath = path.resolve(packageRoot, docFile);
  if (!fs.existsSync(fullPath)) {
    errors.push(`Missing doc: ${docFile}`);
    allValid = false;
  }
}

// 验证 references
console.log('Checking reference files...');
for (const refFile of manifest.references || []) {
  const fullPath = path.resolve(packageRoot, refFile);
  if (!fs.existsSync(fullPath)) {
    warnings.push(`Missing reference: ${refFile} (may be auto-generated)`);
  }
}

// 验证 componentSpecs
console.log('Checking component specs...');
const specsByName = new Map();
for (const specFile of manifest.componentSpecs || []) {
  const fullPath = path.resolve(packageRoot, specFile);
  if (!fs.existsSync(fullPath)) {
    errors.push(`Missing component spec: ${specFile}`);
    allValid = false;
  } else {
    const baseName = path.basename(specFile, '.md');
    specsByName.set(baseName, specFile);
  }
}

// 验证 portableExamples
console.log('Checking portable examples...');
let exampleCount = 0;
for (const exampleFile of manifest.portableExamples || []) {
  const fullPath = path.resolve(packageRoot, exampleFile);
  if (!fs.existsSync(fullPath)) {
    warnings.push(`Missing portable example: ${exampleFile}`);
  } else {
    exampleCount++;
  }
}

// 验证 portable 目录中的文件是否都在 MANIFEST 中
console.log('Checking for unmapped portable files...');
const portableReactDir = path.resolve(packageRoot, 'portable/react');
const portableCssDir = path.resolve(packageRoot, 'portable/css');

const manifestedReactFiles = new Set((manifest.portableExamples || [])
  .filter(f => f.includes('portable/react'))
  .map(f => path.basename(f)));

const manifestedCssFiles = new Set((manifest.portableExamples || [])
  .filter(f => f.includes('portable/css'))
  .map(f => path.basename(f)));

// React 文件检查（可选警告）
if (fs.existsSync(portableReactDir)) {
  const reactFiles = fs.readdirSync(portableReactDir).filter(f => f.endsWith('.tsx'));
  for (const file of reactFiles) {
    if (!manifestedReactFiles.has(file) && file !== 'index.tsx') {
      warnings.push(`Unmapped React file: portable/react/${file}`);
    }
  }
}

// CSS 文件检查（可选警告）
if (fs.existsSync(portableCssDir)) {
  const cssFiles = fs.readdirSync(portableCssDir).filter(f => f.endsWith('.css'));
  for (const file of cssFiles) {
    if (!manifestedCssFiles.has(file) && file !== 'index.css') {
      warnings.push(`Unmapped CSS file: portable/css/${file}`);
    }
  }
}

// 输出报告
if (errors.length === 0) {
  console.log('\n✓ MANIFEST validation PASSED\n');
} else {
  console.log('\n✗ MANIFEST validation FAILED\n');
}

if (errors.length > 0) {
  console.log('ERRORS:');
  errors.forEach(err => console.log(`  - ${err}`));
  console.log();
}

if (warnings.length > 0) {
  console.log('WARNINGS:');
  warnings.slice(0, 10).forEach(warn => console.log(`  - ${warn}`));
  if (warnings.length > 10) {
    console.log(`  ... and ${warnings.length - 10} more warnings`);
  }
  console.log();
}

// 统计
console.log('Statistics:');
console.log(`  Docs: ${manifest.docs?.length || 0}`);
console.log(`  References: ${manifest.references?.length || 0}`);
console.log(`  Component Specs: ${manifest.componentSpecs?.length || 0}`);
console.log(`  Portable Examples: ${exampleCount}/${manifest.portableExamples?.length || 0}`);
console.log(`  Errors: ${errors.length}`);
console.log(`  Warnings: ${warnings.length}`);

process.exit(allValid ? 0 : 1);
