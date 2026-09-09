import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';
import { execFileSync } from 'node:child_process';

const root = path.resolve(new URL('..', import.meta.url).pathname);
const read = file => JSON.parse(fs.readFileSync(path.join(root, file), 'utf8'));
const fail = message => {
  throw new Error(message);
};

const oracle = read('conformance/oracle.json');
const packageJSON = read('conformance/package.json');
const lock = read('conformance/package-lock.json');
const ledger = read('conformance/feature-ledger.json');
const cases = read('conformance/cases.json');
const fixtures = read('conformance/fixture-inventory.json');
const upstream = read('conformance/upstream-inventory.json');

execFileSync('node', ['conformance/oracle-contract.mjs'], { cwd: root, stdio: 'inherit' });

if (oracle.version !== lock.packages['node_modules/@coderline/alphatab'].version) {
  fail('AlphaTab package-lock version does not match oracle.json');
}
if (oracle.version !== packageJSON.dependencies[oracle.package]) {
  fail('AlphaTab package.json version does not match oracle.json');
}
if (oracle.integrity !== lock.packages['node_modules/@coderline/alphatab'].integrity) {
  fail('AlphaTab package-lock integrity does not match oracle.json');
}
if (oracle.sourceRevision !== upstream.sourceRevision) {
  fail('AlphaTab source inventory revision does not match oracle.json');
}

const allowedStatuses = new Set(['supported', 'partial', 'unsupported', 'out-of-scope', 'derived']);
const features = new Map();
for (const feature of ledger.features) {
  if (features.has(feature.id)) fail(`duplicate feature ${feature.id}`);
  for (const field of ['formats', 'upstreamParser', 'upstreamModel', 'upstreamTests', 'goDestination', 'fixtures', 'semanticAssertions']) {
    if (!Array.isArray(feature[field]) || feature[field].length === 0) fail(`${feature.id}.${field} is empty`);
  }
  if (!allowedStatuses.has(feature.status)) fail(`${feature.id} has invalid status ${feature.status}`);
  if (!feature.reason) fail(`${feature.id} has no reason`);
  if (feature.status !== 'supported' && (!Array.isArray(feature.issues) || feature.issues.length === 0)) {
    fail(`${feature.id} is ${feature.status} without an issue`);
  }
  features.set(feature.id, feature);
}

const allowedDiagnosticDispositions = new Set([
  'invalid-data',
  'unknown-syntax',
  'unsupported-feature',
  'lossy-projection',
  'deliberate-default',
  'deliberate-ignore'
]);
const diagnosticDispositions = new Map();
for (const disposition of ledger.diagnosticDispositions) {
  if (!allowedDiagnosticDispositions.has(disposition.id)) fail(`invalid diagnostic disposition ${disposition.id}`);
  if (diagnosticDispositions.has(disposition.id)) fail(`duplicate diagnostic disposition ${disposition.id}`);
  if (typeof disposition.strict !== 'boolean') fail(`${disposition.id}.strict is not a boolean`);
  if (!disposition.description) fail(`${disposition.id} has no description`);
  diagnosticDispositions.set(disposition.id, disposition);
}

const receiptBySource = new Map();
for (const receipt of ledger.diagnosticReceipts) {
  if (!receipt.source || receipt.source.includes('*') || receipt.source === 'GPIF') {
    fail(`diagnostic receipt has a blanket source: ${receipt.source}`);
  }
  if (receiptBySource.has(receipt.source)) fail(`duplicate diagnostic receipt ${receipt.source}`);
  if (!features.has(receipt.feature)) fail(`${receipt.source} has unknown feature ${receipt.feature}`);
  if (!diagnosticDispositions.has(receipt.disposition)) {
    fail(`${receipt.source} has unknown disposition ${receipt.disposition}`);
  }
  if (!receipt.reason) fail(`${receipt.source} has no reason`);
  receiptBySource.set(receipt.source, receipt);
}

const diagnosticKindDisposition = new Map([
  ['ParseDiagnosticInvalidData', 'invalid-data'],
  ['ParseDiagnosticUnknownSyntax', 'unknown-syntax'],
  ['ParseDiagnosticUnsupportedFeature', 'unsupported-feature'],
  ['ParseDiagnosticLossyProjection', 'lossy-projection'],
  ['ParseDiagnosticDeliberateDefault', 'deliberate-default'],
  ['ParseDiagnosticDeliberateIgnore', 'deliberate-ignore']
]);
const sourceDeclaration = /diagnosticSource\(\s*"([^"]+)"\s*,\s*"([^"]+)"\s*,\s*(ParseDiagnostic[A-Za-z]+)\s*\)/g;
const declaredSources = new Map();
for (const file of fs.readdirSync(root).filter(file => file.endsWith('.go'))) {
  const contents = fs.readFileSync(path.join(root, file), 'utf8');
  for (const match of contents.matchAll(sourceDeclaration)) {
    const [, source, feature, kind] = match;
    const disposition = diagnosticKindDisposition.get(kind);
    if (!disposition) fail(`${file} declares ${source} with unknown diagnostic kind ${kind}`);
    if (declaredSources.has(source)) fail(`duplicate diagnostic source declaration ${source}`);
    declaredSources.set(source, { feature, disposition });
  }
}
for (const [source, declaration] of declaredSources) {
  const receipt = receiptBySource.get(source);
  if (!receipt) fail(`${source} has no diagnostic receipt`);
  if (receipt.feature !== declaration.feature) fail(`${source} changed feature to ${declaration.feature}`);
  if (receipt.disposition !== declaration.disposition) fail(`${source} changed disposition to ${declaration.disposition}`);
}
for (const source of receiptBySource.keys()) {
  if (!declaredSources.has(source)) fail(`${source} receipt has no source declaration`);
}
const allCases = [...cases.inputCases, ...cases.exportCases];
const caseIDs = new Set();
for (const item of allCases) {
  if (caseIDs.has(item.id)) fail(`duplicate conformance case ${item.id}`);
  caseIDs.add(item.id);
  for (const feature of item.features) {
    if (!features.has(feature)) fail(`${item.id} references unknown feature ${feature}`);
  }
}

const fixtureByPath = new Map();
for (const fixture of fixtures.fixtures) {
  if (fixtureByPath.has(fixture.path)) fail(`duplicate fixture inventory entry ${fixture.path}`);
  if (!allowedStatuses.has(fixture.disposition)) fail(`${fixture.path} has invalid disposition`);
  if (!fixture.reason) fail(`${fixture.path} has no disposition reason`);
  if (!Array.isArray(fixture.features) || fixture.features.length === 0) {
    fail(`${fixture.path} has no explicit feature mapping`);
  }
  for (const feature of fixture.features) {
    if (!features.has(feature)) fail(`${fixture.path} references unknown feature ${feature}`);
  }
  const digest = crypto.createHash('sha256').update(fs.readFileSync(path.join(root, fixture.path))).digest('hex');
  if (digest !== fixture.sha256) fail(`${fixture.path} hash changed; review the fixture before updating inventory`);
  fixtureByPath.set(fixture.path, fixture);
}
for (const item of cases.inputCases) {
  const fixture = fixtureByPath.get(item.fixture);
  if (!fixture?.semanticSnapshot) fail(`${item.fixture} is not marked as a semantic snapshot fixture`);
  if (JSON.stringify(fixture.features) !== JSON.stringify(item.features)) fail(`${item.fixture} feature inventory is stale`);
}

for (const group of ['testCases', 'importerCases', 'modelFields', 'modelSymbols']) {
  const seen = new Set();
  for (const item of upstream[group]) {
    const identity = group === 'modelSymbols' ? `${item.name}:${item.kind}` : item.name;
    if (seen.has(identity)) fail(`duplicate upstream ${group} entry ${identity}`);
    if (!allowedStatuses.has(item.disposition)) fail(`${item.name} has invalid upstream disposition`);
    if (!features.has(item.feature)) fail(`${item.name} references unknown feature ${item.feature}`);
    if (!item.reason) fail(`${item.name} has no upstream disposition reason`);
    if (group === 'modelSymbols' && ['partial', 'unsupported'].includes(item.disposition)) {
      const parent = features.get(item.feature);
      if (parent.status === 'supported') {
        fail(`${item.name} is ${item.disposition} but parent feature ${item.feature} claims full support`);
      }
      if (parent.issues.length === 0) fail(`${item.name} has no issue-linked parent feature`);
    }
    seen.add(identity);
  }
}

const snapshotCaseIDs = new Set();
for (const file of fs.readdirSync(path.join(root, 'conformance/snapshots')).filter(file => file.endsWith('.json'))) {
  const snapshot = read(`conformance/snapshots/${file}`);
  if (!caseIDs.has(snapshot.case)) fail(`${file} has no case declaration`);
  if (snapshotCaseIDs.has(snapshot.case)) fail(`multiple snapshots declare case ${snapshot.case}`);
  snapshotCaseIDs.add(snapshot.case);
  for (const difference of snapshot.differences) {
    const feature = features.get(difference.feature);
    if (!feature) fail(`${file} has unclassified difference ${difference.path}`);
    if (feature.status === 'supported') fail(`${file} baselines a difference for supported feature ${feature.id}`);
    if (feature.issues.length === 0) fail(`${file} difference ${difference.path} has no issue-linked feature`);
  }
}
for (const caseID of caseIDs) {
  if (!snapshotCaseIDs.has(caseID)) fail(`conformance case ${caseID} has no semantic snapshot`);
}

const referenceCheckout = path.join(root, 'references/alphaTab');
if (!fs.existsSync(path.join(referenceCheckout, '.git'))) {
  fail('references/alphaTab is required to verify the pinned AlphaTab source inventory');
}
for (const source of upstream.sources) {
  const contents = execFileSync('git', ['-C', referenceCheckout, 'show', `${oracle.sourceRevision}:${source.path}`]);
  const digest = crypto.createHash('sha256').update(contents).digest('hex');
  if (digest !== source.sha256) fail(`pinned AlphaTab source hash changed for ${source.path}`);
}

function supportDocument() {
  const rows = ledger.features.map(feature => {
    const issues = feature.issues.length > 0 ? feature.issues.map(issue => `[#${issue}](https://github.com/CaliLuke/go-guitar-pro/issues/${issue})`).join(', ') : 'none';
    return `| \`${feature.id}\` | ${feature.formats.join(', ')} | ${feature.status} | ${issues} | ${feature.reason} |`;
  });
  const dispositionRows = ledger.diagnosticDispositions.map(disposition =>
    `| \`${disposition.id}\` | ${disposition.strict ? 'yes' : 'no'} | ${disposition.description} |`
  );
  const receiptRows = ledger.diagnosticReceipts.map(receipt =>
    `| \`${receipt.source}\` | \`${receipt.feature}\` | \`${receipt.disposition}\` | ${receipt.reason} |`
  );
  return `# Guitar Pro semantic support\n\n` +
    `Generated from [\`conformance/feature-ledger.json\`](../conformance/feature-ledger.json). Do not edit these tables by hand.\n\n` +
    `AlphaTab oracle: \`${oracle.package}@${oracle.version}\`, source \`${oracle.sourceRevision}\`.\n\n` +
    `| Feature | Formats | Status | Issues | Reason |\n| --- | --- | --- | --- | --- |\n${rows.join('\n')}\n\n` +
    `## Parse diagnostics\n\nThe diagnostic disposition is separate from semantic feature support. Strict parsing rejects only dispositions marked Yes.\n\n` +
    `| Disposition | Strict rejection | Meaning |\n| --- | --- | --- |\n${dispositionRows.join('\n')}\n\n` +
    `Each diagnostic receipt names one source construct. Its feature value uses an ID from the semantic support table.\n\n` +
    `| Source construct | Feature | Disposition | Reason |\n| --- | --- | --- | --- |\n${receiptRows.join('\n')}\n`;
}

const docsPath = path.join(root, 'docs/format-support.md');
const generated = supportDocument();
if (process.argv.includes('--write-docs')) {
  fs.mkdirSync(path.dirname(docsPath), { recursive: true });
  fs.writeFileSync(docsPath, generated);
} else if (!fs.existsSync(docsPath) || fs.readFileSync(docsPath, 'utf8') !== generated) {
  fail('docs/format-support.md is stale; run node conformance/verify.mjs --write-docs and review it');
}
