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
  return `# Guitar Pro semantic support\n\n` +
    `Generated from [\`conformance/feature-ledger.json\`](../conformance/feature-ledger.json). Do not edit this table by hand.\n\n` +
    `AlphaTab oracle: \`${oracle.package}@${oracle.version}\`, source \`${oracle.sourceRevision}\`.\n\n` +
    `| Feature | Formats | Status | Issues | Reason |\n| --- | --- | --- | --- | --- |\n${rows.join('\n')}\n`;
}

const docsPath = path.join(root, 'docs/format-support.md');
const generated = supportDocument();
if (process.argv.includes('--write-docs')) {
  fs.mkdirSync(path.dirname(docsPath), { recursive: true });
  fs.writeFileSync(docsPath, generated);
} else if (!fs.existsSync(docsPath) || fs.readFileSync(docsPath, 'utf8') !== generated) {
  fail('docs/format-support.md is stale; run node conformance/verify.mjs --write-docs and review it');
}
