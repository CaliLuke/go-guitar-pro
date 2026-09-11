import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';
import { execFileSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';

const root = path.resolve(path.dirname(fileURLToPath(import.meta.url)), '..');
const checkout = path.join(root, 'references/alphaTab');
const oracle = JSON.parse(fs.readFileSync(path.join(root, 'conformance/oracle.json'), 'utf8'));
const inventoryPath = process.env.UPSTREAM_INVENTORY_OVERLAY ?? path.join(root, 'conformance/upstream-inventory.json');
const inventory = JSON.parse(fs.readFileSync(inventoryPath, 'utf8'));
const check = process.argv.includes('--check');

if (!fs.existsSync(path.join(checkout, '.git'))) {
  throw new Error('references/alphaTab is required to refresh the upstream inventory');
}

const modelNames = ['Automation', 'Bar', 'Beat', 'MasterBar', 'Note', 'Score', 'Staff', 'Track', 'Voice'];
const importerTestNames = ['Gp3Importer', 'Gp4Importer', 'Gp5Importer', 'GpxImporter', 'Gp7Importer', 'Gp8Importer'];
const sourcePaths = [
  'packages/alphatab/src/importer/GpifParser.ts',
  'packages/alphatab/src/importer/Gp3To5Importer.ts',
  ...modelNames.map(model => `packages/alphatab/src/model/${model}.ts`),
  'packages/alphatab/test/importer/GpImporterTestHelper.ts',
  ...importerTestNames.map(test => `packages/alphatab/test/importer/${test}.test.ts`)
];

function source(file) {
  return execFileSync('git', ['-C', checkout, 'show', `${oracle.sourceRevision}:${file}`], { encoding: 'utf8' });
}

function declaredModelSymbols(model) {
  const text = source(`packages/alphatab/src/model/${model}.ts`);
  const symbols = [];
  let owner = model;
  for (const [index, line] of text.split('\n').entries()) {
    const declaration = /^(?:export\s+)?(?:abstract\s+)?(?:class|interface)\s+([_$A-Za-z][_$A-Za-z0-9]*)/.exec(line);
    if (declaration) owner = declaration[1];
    const match = /^\s*public\s+(?:(?:static|readonly|override|abstract)\s+)*(?:(get|set)\s+)?([_$A-Za-z][_$A-Za-z0-9]*)/.exec(line);
    if (!match || match[2] === 'constructor') continue;
    const tail = line.slice(match.index + match[0].length).trimStart();
    const kind = match[1] ? match[1] : tail.startsWith('(') || tail.startsWith('<') ? 'method' : 'field';
    symbols.push({ name: `${owner}.${match[2]}`, member: match[2], kind, line: index + 1 });
  }
  return symbols;
}

function modelSymbols(model) {
  const candidates = declaredModelSymbols(model);
  const groups = new Map();
  for (const candidate of candidates) {
    const key = `${candidate.member}:${candidate.kind}`;
    if (!groups.has(key)) groups.set(key, []);
    groups.get(key).push(candidate);
  }
  // The historical inventory has one slot per filename/member/kind. Preserve
  // that 384-symbol surface while correcting unambiguous helper ownership. If
  // a helper and the filename's main class declare the same member, retain the
  // main declaration; the expanded capability scanner inventories both.
  return [...groups.values()].map(group => {
    const selected = group.find(symbol => symbol.name.startsWith(`${model}.`)) ?? group[0];
    return { name: selected.name, kind: selected.kind };
  }).sort((left, right) => left.name.localeCompare(right.name));
}

const previousSymbols = new Map(
  (inventory.modelSymbols ?? []).map(item => [`${item.name}:${item.kind}`, item])
);
const previousByFileMember = new Map(
  (inventory.modelSymbols ?? []).map(item => {
    const [owner, member] = item.name.split('.', 2);
    return [`${owner}:${member}:${item.kind}`, item];
  })
);
const discoveredSymbols = modelNames.flatMap(modelSymbols);
const priorMigrations = inventory.modelSymbolMigration?.renames ?? [];
const migratedNames = [];
inventory.modelSymbols = discoveredSymbols.map(symbol => {
  const previous = previousSymbols.get(`${symbol.name}:${symbol.kind}`);
  if (previous && previous.kind === symbol.kind) return previous;
  const [owner, member] = symbol.name.split('.', 2);
  const sourceModel = modelNames.find(model =>
    declaredModelSymbols(model).some(candidate => candidate.name === symbol.name && candidate.kind === symbol.kind)
  );
  const filenamePrevious = sourceModel
    ? previousByFileMember.get(`${sourceModel}:${member}:${symbol.kind}`)
    : undefined;
  if (filenamePrevious) {
    migratedNames.push({ from: filenamePrevious.name, to: symbol.name, kind: symbol.kind });
    return { ...filenamePrevious, name: symbol.name };
  }
  return { ...symbol, disposition: 'unclassified', feature: null, reason: '' };
});
const migrationsBySource = new Map(
  [...priorMigrations, ...migratedNames].map(item => [`${item.from}:${item.kind}`, item])
);
const allMigrations = [...migrationsBySource.values()];
const discoveredIdentities = new Set(inventory.modelSymbols.map(item => `${item.name}:${item.kind}`));
const migratedFrom = new Set(allMigrations.map(item => `${item.from}:${item.kind}`));
const unexplainedRemovals = [...previousSymbols.keys()].filter(key =>
  !discoveredIdentities.has(key) && !migratedFrom.has(key)
).sort();
inventory.modelSymbolMigration = {
  previous: inventory.modelSymbolMigration?.previous ?? previousSymbols.size,
  current: inventory.modelSymbols.length,
  unchanged: inventory.modelSymbols.length - allMigrations.length,
  renamed: allMigrations.length,
  added: Math.max(0, inventory.modelSymbols.length - (inventory.modelSymbolMigration?.previous ?? previousSymbols.size)),
  removed: Math.max(0, (inventory.modelSymbolMigration?.previous ?? previousSymbols.size) - inventory.modelSymbols.length),
  unexplainedRemovals,
  renames: allMigrations.sort((left, right) => left.from.localeCompare(right.from))
};
if (unexplainedRemovals.length > 0) {
  throw new Error(`upstream model symbols disappeared without a reviewed migration: ${unexplainedRemovals.join(', ')}`);
}

const helperSource = source('packages/alphatab/test/importer/GpImporterTestHelper.ts');
const discoveredChecks = [...helperSource.matchAll(/public static (check[_$A-Za-z][_$A-Za-z0-9]*)/g)].map(match => match[1]);
const previousChecks = new Map(inventory.testCases.map(item => [item.name, item]));
inventory.testCases = discoveredChecks.map(name => previousChecks.get(name) ?? {
  name, disposition: 'unclassified', feature: null, reason: ''
});

function discoverImporterCases(test) {
  const text = source(`packages/alphatab/test/importer/${test}.test.ts`);
  const matches = [...text.matchAll(/\bit\(\s*(['"`])([^'"`]+)\1/g)];
  return matches.map((match, index) => {
    const block = text.slice(match.index, matches[index + 1]?.index ?? text.length);
    const fixture = /prepareImporterWithFile\(\s*['"]([^'"]+)/.exec(block)?.[1] ?? null;
    const assertions = [...block.matchAll(/GpImporterTestHelper\.(check[_$A-Za-z][_$A-Za-z0-9]*)/g)]
      .map(assertion => assertion[1]);
    return { name: `${test}.${match[2]}`, fixture, assertions };
  });
}

const previousImporterCases = new Map((inventory.importerCases ?? []).map(item => [item.name, item]));
const discoveredImporterCases = importerTestNames.flatMap(discoverImporterCases);
inventory.importerCases = discoveredImporterCases.map(item => {
  const previous = previousImporterCases.get(item.name);
  if (previous && previous.fixture === item.fixture &&
      JSON.stringify(previous.assertions) === JSON.stringify(item.assertions)) {
    return previous;
  }
  return { ...item, disposition: 'unclassified', feature: null, reason: '' };
});

inventory.schemaVersion = 1;
inventory.sourceRevision = oracle.sourceRevision;
inventory.sources = sourcePaths.map(file => ({
  path: file,
  sha256: crypto.createHash('sha256').update(source(file)).digest('hex')
}));

if (check) {
  const current = JSON.parse(fs.readFileSync(inventoryPath, 'utf8'));
  const identity = value => ({
    sourceRevision: value.sourceRevision,
    sources: value.sources,
    testCases: value.testCases.map(item => item.name),
    importerCases: value.importerCases.map(item => ({
      name: item.name, fixture: item.fixture, assertions: item.assertions
    })),
    modelSymbols: value.modelSymbols.map(item => ({ name: item.name, kind: item.kind })),
    modelSymbolMigration: value.modelSymbolMigration
  });
  if (JSON.stringify(identity(current)) !== JSON.stringify(identity(inventory))) {
    throw new Error('AlphaTab upstream inventory is stale; refresh it and classify every new symbol');
  }
  process.exit(0);
}

fs.writeFileSync(inventoryPath, `${JSON.stringify(inventory, null, 2)}\n`);
