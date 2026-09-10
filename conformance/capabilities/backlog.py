#!/usr/bin/env python3
"""Validate, inspect, and publish bounded capability work items."""

import argparse
import collections
from datetime import datetime, timezone
import json
from pathlib import Path
import re
import subprocess
import tempfile
import time

HERE = Path(__file__).resolve().parent
ROOT = HERE.parent.parent
REPO = 'CaliLuke/go-guitar-pro'
BASE = f'https://github.com/{REPO}'
META = HERE / 'backlog-meta.json'


def read(path):
    return json.loads(path.read_text())


def save(path, value):
    temporary = path.with_suffix('.tmp')
    temporary.write_text(json.dumps(value, indent=2, ensure_ascii=False) + '\n')
    temporary.replace(path)


def load():
    items = []
    for path in sorted((HERE / 'work-items').glob('*.json')):
        item = read(path)
        if path.stem != item['id']:
            raise ValueError(f'Work item filename must match its ID: {path}')
        items.append(item)
    return items


def validate(items, catalog, meta):
    caps = {c['id']: c for c in catalog['capabilities']}
    by_id = {w['id']: w for w in items}
    if len(by_id) != len(items):
        raise ValueError('Duplicate work item ID')
    numbers = set()
    for w in items:
        name = w['id']
        if not re.fullmatch(r'[a-z0-9]+(?:-[a-z0-9]+)*', name):
            raise ValueError(f'Invalid work ID: {name}')
        if w['schema_version'] != 1 or w['capability'] not in caps:
            raise ValueError(f'Unknown work schema or capability: {name}')
        if w['kind'] not in ('implementation', 'investigation') or w['status'] not in ('todo', 'in_progress', 'done'):
            raise ValueError(f'Invalid kind or status: {name}')
        if w['priority'] not in (1, 2, 3) or not w['formats'] or not w['stages']:
            raise ValueError(f'Missing work scope: {name}')
        if not set(w['stages']) <= {'import', 'model', 'export'}:
            raise ValueError(f'Invalid stages: {name}')
        for field in ('title', 'scope', 'acceptance', 'evidence', 'reproduction'):
            if not w.get(field):
                raise ValueError(f'Missing {field}: {name}')
        if len(w['acceptance']) < 2 or any(not isinstance(a, str) or len(a.strip()) < 20 for a in w['acceptance']):
            raise ValueError(f'Missing observable acceptance criteria: {name}')
        repro = w['reproduction']
        if not repro.get('instructions') or repro.get('kind') not in ('runtime-candidate', 'source-review'):
            raise ValueError(f'Invalid reproduction: {name}')
        if repro['kind'] == 'runtime-candidate' and not (repro.get('fixture') and repro.get('differences')):
            raise ValueError(f'Missing runtime candidate: {name}')
        if any(d not in by_id for d in w['depends_on']):
            raise ValueError(f'Missing dependency: {name}')
        if len(set(w['depends_on'])) != len(w['depends_on']):
            raise ValueError(f'Duplicate dependency: {name}')
        if w['status'] == 'in_progress' and not w.get('owner'):
            raise ValueError(f'Unowned active work: {name}')
        issue = w.get('issue')
        if issue:
            if issue['number'] in numbers or issue['url'] != f"{BASE}/issues/{issue['number']}" or issue['state'] not in ('OPEN', 'CLOSED'):
                raise ValueError(f'Invalid or duplicate issue: {name}')
            numbers.add(issue['number'])
        elif meta.get('published'):
            raise ValueError(f'Published backlog has an unticketed item: {name}')
        if w['status'] == 'done':
            if not w.get('resolution') or not w.get('verification'):
                raise ValueError(f'Done work requires resolution and verification: {name}')
            for receipt in w['verification']:
                if not re.fullmatch(r'[0-9a-f]{40}', receipt.get('commit', '')) or not receipt.get('commands') or not receipt.get('evidence'):
                    raise ValueError(f'Incomplete verification receipt: {name}')
            if any(by_id[d]['status'] != 'done' for d in w['depends_on']):
                raise ValueError(f'Done work has unfinished dependencies: {name}')
    order(items)  # Also rejects cycles.
    covered = {w['capability'] for w in items}
    gaps = {c['id'] for c in caps.values() if c['scope'] == 'guitar-pro' and any(s != 'supported' for s in c['stages'].values())}
    if gaps - covered:
        raise ValueError(f'Uncovered capability gaps: {sorted(gaps - covered)}')
    return items


def order(items):
    by_id = {w['id']: w for w in items}
    result, seen, active = [], set(), set()

    def visit(name):
        if name in active:
            raise ValueError(f'Dependency cycle: {name}')
        if name in seen:
            return
        active.add(name)
        for dep in by_id[name]['depends_on']:
            visit(dep)
        active.remove(name)
        seen.add(name)
        result.append(by_id[name])

    for w in sorted(items, key=lambda w: (w['priority'], w['kind'], w['id'])):
        visit(w['id'])
    return result


def populate(con, catalog):
    meta = read(META)
    items = validate(load(), catalog, meta)
    con.execute("INSERT INTO metadata VALUES ('backlog_metadata',?)", (json.dumps(meta, sort_keys=True),))
    for w in items:
        issue = w.get('issue') or {}
        con.execute('INSERT INTO work_item VALUES (' + ','.join('?' for _ in range(20)) + ')', (
            w['id'], w['capability'], w['title'], w['kind'], w['priority'], w['status'], w['owner'], w['scope'],
            *[json.dumps(w[k], sort_keys=True) for k in ('formats', 'stages', 'acceptance', 'exclusions', 'reproduction', 'evidence', 'shared_files')],
            w['resolution'], json.dumps(w['verification'], sort_keys=True), issue.get('number'), issue.get('url'), issue.get('state')
        ))
    for w in items:
        con.executemany('INSERT INTO work_dependency VALUES (?,?)', [(w['id'], d) for d in w['depends_on']])
    return items


def link(w):
    issue = w.get('issue')
    return f"[#{issue['number']}]({issue['url']})" if issue else f"`{w['id']}`"


def marker(name):
    return f'<!-- go-guitar-pro-capability-work:{name} -->'


def issue_body(w, items, meta):
    by_id = {i['id']: i for i in items}
    oracle = read(ROOT / 'conformance/oracle.json')
    ref = meta.get('baseline_commit') or 'main'
    work_url = f"{BASE}/blob/main/conformance/capabilities/work-items/{w['id']}.json"
    lines = [marker(w['id']), '', w['scope'], '',
             f"**Type:** {w['kind']}. **Priority:** P{w['priority']}. **Capability:** `{w['capability']}`.",
             f"**Formats:** {', '.join(w['formats'])}. **Stages to assess:** {', '.join(w['stages'])}.", '',
             '## Acceptance criteria', '']
    lines += [f'- [ ] {a}' for a in w['acceptance']]
    if w['kind'] == 'investigation':
        lines += ['', 'Deliver a reproducible decision with stage-specific evidence. Classify valid source loss, intentional format limits, derived values, malformed input, and oracle defects separately. Record follow-up implementation tickets for confirmed unresolved defects. Closing this investigation does not establish feature support.']
    else:
        lines += ['', 'Any public API change must expose the authored value with a documented edit/reconciliation contract. Add a non-default public API regression and independent pinned-consumer export evidence. Preserve compatibility or explicitly document the change.']
    if w['exclusions']:
        lines += ['', '## Scope boundaries', ''] + [f'- {x}' for x in w['exclusions']]
    lines += ['', '## Reproduction and evidence', '', w['reproduction']['instructions'], '']
    repro = w['reproduction']
    if repro['kind'] == 'runtime-candidate':
        lines += [f"Fixture: `{repro['fixture']}`. Probe capability: `{repro['capability']}`.", '',
                  'Saved observed differences (absence is distinct from a null value):', '', '```json', json.dumps(repro['differences'], indent=2), '```', '',
                  'Inspect the complete receipt with:', '', '```sh',
                  'python3 conformance/capabilities/manage.py query ' + json.dumps("SELECT * FROM probe_comparison WHERE capability_id='" + repro['capability'] + "' AND fixture_path='" + repro['fixture'].replace("'", "''") + "'"), '```', '',
                  'For a `packages/alphatab/test-data/...` path, obtain the original from the pinned reference checkout. Synthetic fixture construction is recorded in `conformance/capabilities/probe.mjs`.']
    else:
        lines += ['This starts from a source review. Construct a valid non-default fixture that exercises the acceptance criteria; the lack of a saved probe is not evidence of support.']
    lines.append('')
    for e in w['evidence']:
        if e['kind'] == 'upstream-source':
            lines.append(f"- AlphaTab [{e['symbol']}]({e['url']})")
        elif e['kind'] == 'source':
            path = e['reference'].split(':')[0]
            lines.append(f"- Repository [{e['reference']}]({BASE}/blob/{ref}/{path})")
        else:
            lines.append(f"- {e['kind']}: `{e.get('reference', '')}`. {e.get('detail', '')}")
    lines += ['', f"Oracle: AlphaTab {oracle['version']}, revision `{oracle['sourceRevision']}`. Audit baseline: `{ref}`.", '',
              '## Dependencies and coordination', '']
    lines += [f'- Complete {link(by_id[d])}: {by_id[d]["title"]}.' for d in w['depends_on']] or ['No hard prerequisite.']
    lines += ['', 'Before claiming, refresh issue state and check `ready_work` and `work_conflicts`. Mark the work item `in_progress` with an owner. Source pointers are starting points; shared-file overlap is advisory and not an exhaustive conflict map.', '',
              '## Verification and inventory update', '',
              '- Follow `AGENTS.md`, `docs/semantic-model.md`, and `conformance/README.md` for semantic changes. Add domain-named tests and update `conformance/feature-ledger.json`.',
              '- Run `./conformance/check.sh` and `prek run --all-files` for semantic changes. Rerun the capability probe before claiming a gap is resolved.',
              f'- Update the [canonical work item]({work_url}) with the resolution, verified commit, command results, and evidence. Update the scoped catalog assessment separately; tests passing or issue closure never promote it automatically.',
              '- Run `python3 conformance/capabilities/manage.py refresh` and commit the JSON, database, and generated reports together.',
              '- Review every acceptance criterion against the exact final commit and completed prerequisites before closing. Record residual limits and open follow-up issues.', '',
              f"[Agent workflow]({BASE}/blob/main/conformance/capabilities/README.md) · [Full backlog]({BASE}/blob/main/conformance/capabilities/BACKLOG.md)", '']
    return '\n'.join(lines)


def report(items, meta):
    by_id = {w['id']: w for w in items}
    counts = collections.Counter(w['kind'] for w in items)
    lines = ['# Capability work backlog', '', 'Generated from `work-items/*.json`. Regenerate with `manage.py refresh`.', '',
             f"{len(items)} work items: {counts['implementation']} implementation tasks and {counts['investigation']} investigations.",
             'Investigation closure records a decision; it does not certify capability support.', '',
             'Use `manage.py ready` for work without unfinished dependencies. Check shared files before dispatching concurrent work.', '']
    if meta.get('index_issue'):
        i = meta['index_issue']
        lines += [f"GitHub tracking issue: [#{i['number']}]({i['url']}).", '']
    lines += ['| Work item | Kind | P | State | Prerequisites | Ticket |', '| --- | --- | ---: | --- | --- | --- |']
    for w in order(items):
        deps = ', '.join(link(by_id[d]) for d in w['depends_on']) or '—'
        state = w['status']
        if state == 'todo' and any(by_id[d]['status'] != 'done' for d in w['depends_on']):
            state = 'blocked'
        lines.append(f"| [{w['title']}](work-items/{w['id']}.json) | {w['kind']} | {w['priority']} | {state} | {deps} | {link(w)} |")
    return '\n'.join(lines) + '\n'


def gh(*args):
    return subprocess.check_output(['gh', *args], text=True).strip()


def existing_issues():
    return json.loads(gh('issue', 'list', '--repo', REPO, '--state', 'all', '--limit', '1000', '--json', 'number,url,state,body,title'))


def find_existing(issues, name):
    matches = [i for i in issues if marker(name) in (i.get('body') or '')]
    if len(matches) > 1:
        raise ValueError(f'Duplicate GitHub markers for {name}; reconcile manually')
    return matches[0] if matches else None


def receipt(issue):
    return {k: issue[k] for k in ('number', 'url', 'state')} | {'checked_at': datetime.now(timezone.utc).isoformat()}


def create(title, body):
    with tempfile.TemporaryDirectory(prefix='capability-issue-') as temp:
        path = Path(temp) / 'body.md'
        path.write_text(body)
        url = gh('issue', 'create', '--repo', REPO, '--title', title, '--body-file', str(path))
    number = int(url.rsplit('/', 1)[1])
    return {'number': number, 'url': url, 'state': 'OPEN', 'title': title, 'body': body}


def index_body(items, meta):
    caps = read(HERE / 'catalog.json')['capabilities']
    scoped = [c for c in caps if c['scope'] == 'guitar-pro']
    incomplete = sum(any(s != 'supported' for s in c['stages'].values()) for c in scoped)
    lines = [marker('backlog-index'), '',
             'Track the remaining Guitar Pro import, public model, and GP8 export differences against pinned AlphaTab 1.8.4.', '',
             f"The audit contains {len(caps)} capabilities ({len(scoped)} in scope). These {len(items)} work items cover all {incomplete} currently incomplete capability rows, including separately scoped diagnostic and audit blockers.",
             'Implementation tickets have bounded observable requirements. Investigation tickets first classify unresolved behavior and record follow-up defects or explicit limits.', '',
             f"[Database and agent workflow]({BASE}/blob/main/conformance/capabilities/README.md) · [Inventory]({BASE}/blob/main/conformance/capabilities/CAPABILITIES.md) · [Backlog]({BASE}/blob/main/conformance/capabilities/BACKLOG.md)", '',
             'Run `python3 conformance/capabilities/manage.py ready` after checkout. Honor hard dependencies and coordinate shared files. Each ticket contains its evidence and acceptance checklist. Do not close from a passing self-round-trip or coverage count.', '',
             'Ordinary repeat count semantics already have #38 and are not duplicated here. Simile marks and navigation directions have separate tickets.', '']
    for kind in ('implementation', 'investigation'):
        lines += [f'## {kind.capitalize()}', '']
        lines += [f"- [{'x' if w['status'] == 'done' else ' '}] {link(w)} — {w['title']} (P{w['priority']})" for w in order(items) if w['kind'] == kind]
        lines.append('')
    return '\n'.join(lines)


def publish():
    meta = read(META)
    items = validate(load(), read(HERE / 'catalog.json'), meta)
    if not re.fullmatch(r'[0-9a-f]{40}', meta.get('baseline_commit', '')):
        raise ValueError('Publish and record the baseline commit before filing issues')
    gh('api', f"repos/{REPO}/commits/{meta['baseline_commit']}", '--jq', '.sha')
    issues = existing_issues()
    index = find_existing(issues, 'backlog-index')
    if index is None:
        index = create('Track Guitar Pro capability parity backlog', index_body(items, meta))
        time.sleep(2)
    meta['index_issue'] = receipt(index)
    save(META, meta)
    for w in order(items):
        issue = find_existing(issues, w['id'])
        if w.get('issue') and (not issue or issue['number'] != w['issue']['number']):
            raise ValueError(f"Issue link/marker drift: {w['id']}")
        if issue is None:
            issue = create(w['title'], issue_body(w, items, meta))
            issues.append(issue)
            print(f"Created #{issue['number']} {w['id']}", flush=True)
        w['issue'] = receipt(issue)
        save(HERE / 'work-items' / (w['id'] + '.json'), w)
        # Serialize writes and leave time between GitHub content creation calls.
        time.sleep(2)
    meta['published'] = True
    save(META, meta)
    with tempfile.TemporaryDirectory(prefix='capability-index-') as temp:
        path = Path(temp) / 'body.md'
        path.write_text(index_body(items, meta))
        gh('issue', 'edit', str(index['number']), '--repo', REPO, '--body-file', str(path))
    print(f"Linked {len(items)} tickets in {index['url']}")


def sync():
    issues = existing_issues()
    for w in load():
        issue = find_existing(issues, w['id'])
        if not issue or not w.get('issue') or issue['number'] != w['issue']['number']:
            raise ValueError(f"Missing or mismatched published issue: {w['id']}")
        w['issue'] = receipt(issue)
        save(HERE / 'work-items' / (w['id'] + '.json'), w)
    print('Refreshed issue state. Review backlog_state_drift; no work status or capability rating was changed.')


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest='command', required=True)
    sub.add_parser('check')
    show = sub.add_parser('show')
    show.add_argument('id')
    render = sub.add_parser('render')
    render.add_argument('directory', type=Path)
    sub.add_parser('publish')
    sub.add_parser('sync')
    args = parser.parse_args()
    items = validate(load(), read(HERE / 'catalog.json'), read(META))
    if args.command == 'check':
        print(f'{len(items)} bounded work items; all capability gaps covered; dependency graph valid.')
    elif args.command == 'show':
        matches = [w for w in items if w['id'] == args.id]
        if not matches:
            parser.error(f'Unknown work item: {args.id}')
        print(issue_body(matches[0], items, read(META)))
    elif args.command == 'render':
        args.directory.mkdir(parents=True, exist_ok=True)
        for w in items:
            (args.directory / (w['id'] + '.md')).write_text(issue_body(w, items, read(META)))
        (args.directory / 'index.md').write_text(index_body(items, read(META)))
        print(f'Rendered {len(items)} tickets to {args.directory}')
    elif args.command == 'publish':
        publish()
    else:
        sync()


if __name__ == '__main__':
    main()
