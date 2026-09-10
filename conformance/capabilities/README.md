# Capability database

This directory tracks Guitar Pro capability gaps against pinned AlphaTab 1.8.4.
It contains development tools. The production Go library has no database dependency.

Start with [the audit](AUDIT.md), [the capability list](CAPABILITIES.md), and [the work backlog](BACKLOG.md).
The SQLite file is [capabilities.sqlite](capabilities.sqlite).

The database separates three stages: import, public model, and GP8 export.
Each capability includes its scope, formats, priority, finding, evidence, and completion criterion.
Format lists identify relevant formats. Ratings describe the combined scope, with narrower format differences explained in the finding.

`supported` means the stated slice has implementation and scoped evidence.
It does not certify every format variant or interaction.
`partial` identifies incomplete preservation. `missing` identifies an absent capability.
`unverified` identifies an unresolved assessment.
`out-of-scope` identifies other AlphaTab services, such as its audio player.

Partial and unverified rows receive no completion credit.
The progress views count capabilities without weights.
They do not measure the percentage of all musical behavior.

## Query the inventory

Run these commands from the repository root:

```sh
python3 conformance/capabilities/manage.py summary
python3 conformance/capabilities/manage.py gaps
python3 conformance/capabilities/manage.py query \
  "SELECT id,title,import_status,export_status FROM gaps WHERE priority=1"
```

The query command opens SQLite in read-only mode.
Any SQLite browser can open the same file.

Useful queries:

```sql
-- Compare our stages with each row in the public format documentation.
SELECT * FROM website_comparison WHERE format='gp8';

-- Find the exact values lost in a navigation example.
SELECT * FROM observed_differences
WHERE title='Navigation targets and jumps';

-- Find source fields that still need an explicit capability association.
SELECT name,path,line FROM unreviewed_constructs
WHERE kind='field' AND path LIKE '%/model/%';

-- Find missing fixtures by content, rather than filename.
SELECT * FROM missing_upstream_fixtures;

-- Inspect actual export failures, including files that parsed successfully.
SELECT fixture_path,export_error FROM probe_result
WHERE export_error IS NOT NULL;

-- Inspect existing matrix evidence for one public field.
SELECT * FROM obligation WHERE name='MeasureHeader.RepeatCount';

-- Check whether the saved runtime evidence matches the current source.
SELECT * FROM metadata WHERE key IN ('probe_fresh','probe_metadata');
```

## Update a capability

1. Edit the relevant row in [catalog.json](catalog.json).
2. Update its finding, formats, stages, evidence, and completion criterion.
3. Add the applicable public API tests and independent AlphaTab assertions.
4. Update the existing semantic ledger when the library contract changes.
5. Rebuild the database and readable report:

```sh
python3 conformance/capabilities/manage.py refresh
```

6. Run the repository gate:

```sh
prek run --all-files
```

`catalog.json` is the editable source for reviewed assessments.
The SQLite database combines that source with the existing ledger, upstream inventory, documentation snapshot, and runtime receipt.
Direct SQL edits are temporary: refresh replaces the generated database.
Commit the catalog, database, and generated report together.

The gate checks database integrity, evidence references, documentation mappings, source drift, and report synchronization.
It does not promote a rating when a test or probe passes.
New source constructs remain visible as unverified until a reviewer associates them with a capability.
A capability association alone does not prove an individual source construct.

The expanded source scan reads the pinned Git revision directly.
It also works with the conformance gate's checkout without working files.
It covers model, importer, exporter, MIDI source, and Guitar Pro importer tests.
The scan is lexical. It does not resolve TypeScript types or discover every semantic branch.

## Refresh runtime evidence

The runtime probe uses the public Go API and the pinned AlphaTab package.
It compares selected authored values before and after Go import plus GP8 export.
It includes the local corpus, missing upstream Guitar Pro fixtures, and one generated lowercase-key-mode case.

```sh
node conformance/capabilities/probe.mjs
python3 conformance/capabilities/manage.py refresh
```

The probe needs Go, Python 3, Node, Git, and the installed conformance npm package.
It writes temporary exports outside the repository and removes them after the run.
It records input hashes, tool hashes, the source commit, the oracle pin, diagnostics, and exact comparison paths.
The lowercase-key-mode case changes only key-mode text in the existing notes fixture.

The runtime receipt is [probe-results.json](probe-results.json).
It records failed comparisons as blocked, never as equal.
Non-default source counts distinguish exercised values from default-only cases.
An import-plus-export difference does not identify the failing stage by itself.
The source review supplies that distinction for confirmed gaps.

Source changes mark saved probe evidence as stale after database refresh.
The normal gate checks this metadata but does not rerun the broad probe.
Rerun the probe before claiming that a capability gap is resolved.

## Refresh the public documentation snapshot

```sh
python3 conformance/capabilities/sync-docs.py
```

Review the changed rows in [website-features.json](website-features.json).
Add mappings for new feature names in `catalog.json`, then run `manage.py refresh`.
An unmapped documentation row fails the build.

Website ratings are reference claims, not our ratings or the pinned oracle's contract.
Some website ratings conflict with pinned source behavior.
The original URL, retrieval time, and page hash remain available for review.

## Database tables

| Tables or views | Purpose |
| --- | --- |
| `capability`, `assessment`, `evidence` | Reviewed capability scope and stage ratings |
| `capability_status`, `progress`, `gaps` | Stage comparisons and remaining work |
| `upstream_construct`, `construct_capability`, `unreviewed_constructs` | Expanded source inventory and unresolved associations |
| `source_file`, `metadata` | Revision, hashes, scope, and receipt freshness |
| `website_feature`, `website_capability`, `website_comparison` | Every captured format-documentation row |
| `obligation`, `matrix_case` | Existing public fields, enums, wire fields, dispatches, and test cases |
| `fixture`, `corpus_diagnostic` | Local fixture inventory and existing diagnostic receipt |
| `upstream_test`, `upstream_fixture`, `missing_upstream_fixtures` | Upstream tests and fixture coverage by content |
| `probe_result`, `probe_comparison`, `probe_coverage`, `observed_differences` | Runtime results, failures, and exact differences |
| `issue` | Explicit issue state at the review date |

## Work through the backlog

The backlog contains one or more bounded tasks for every incomplete capability.
Implementation tasks describe confirmed work. Investigations require a reproducible decision before implementation scope is accepted.
The remaining AlphaTab products, such as rendering and playback, remain explicitly outside the Guitar Pro library scope.

```sh
python3 conformance/capabilities/backlog.py sync
python3 conformance/capabilities/manage.py refresh
python3 conformance/capabilities/manage.py ready
python3 conformance/capabilities/backlog.py show key
```

`sync` reads GitHub issue states and saves them locally. It sends no messages and changes no issue state.
Skip it when offline; saved issue states include their last check time.
`show` prints a complete task packet, including acceptance criteria and evidence.
The original probe is a baseline. Reproduce on the current commit before changing code.

1. Select a task from `ready_work`. Check its GitHub issue for new scope or an existing owner.
2. Check `work_conflicts` and coordinate overlapping changes. Source pointers are not an exhaustive conflict map.
3. Set `status` to `in_progress` and `owner` in the relevant `work-items/<id>.json` file.
4. Follow the repository semantic workflow and the ticket's exact acceptance criteria.
5. Record the resolution and verification receipt. Update capability ratings only for the proven scope.
6. Rebuild the database and reports, then run the required gates.
7. Review the final commit and prerequisites against every acceptance criterion before closing the issue.
8. Sync issue state and regenerate the database after closure.

A `done` item requires a resolution and at least one verification receipt:

```json
{
  "commit": "<full 40-character commit that was verified>",
  "commands": ["<command>: PASS; <relevant result>"],
  "evidence": ["<test, oracle assertion, or investigation decision reference>"]
}
```

Record confirmed residual defects as individual follow-up items and issues.
An investigation can close with an explicit target-format limit or a documented follow-up decision.
Its closure does not establish implementation support.
Issue closure never changes a capability rating or marks a work item done automatically.

The canonical inputs are `catalog.json`, `work-items/*.json`, and `backlog-meta.json`.
SQLite and both Markdown inventories are generated artifacts.
Resolve edits in the JSON inputs, then regenerate; do not select one side of a binary database conflict.
Integrate changes serially when they share the catalog, semantic ledger, or generated database.

```sql
SELECT id,kind,priority,title,issue_url FROM ready_work ORDER BY priority,id;
SELECT * FROM blocked_work;
SELECT * FROM work_conflicts WHERE work_id='fermata' OR other_work_id='fermata';
SELECT * FROM uncovered_gaps;
SELECT * FROM unticketed_work;
SELECT * FROM backlog_state_drift;
```

`ready_work` means no unfinished prerequisite, no active owner status, and no recorded closed issue.
It does not mean independent files or permission to skip the task's evidence requirements.
The build rejects uncovered gaps, unknown dependencies, cycles, duplicate issue links, and incomplete completion receipts.
Once published, it also rejects work items without issue links.

## Publish or extend the tickets

Create a new JSON work item with a stable ID. Give it bounded scope, observable acceptance criteria,
formats, stages, evidence, a reproduction path, and any hard dependencies.
Render ticket bodies for review before publishing:

```sh
python3 conformance/capabilities/backlog.py render /tmp/capability-ticket-drafts
```

`backlog.py publish` creates GitHub issues and updates the tracking issue.
Run it only when ticket publication is authorized. It requires a published baseline commit in `backlog-meta.json`.
It finds existing tickets by stable body markers, including closed tickets, and saves each link as it proceeds.
On failure, rerun the command; it reuses existing tickets and does not overwrite their descriptions.
The tracking issue is regenerated. Individual acceptance changes require a deliberate GitHub edit.
For a newly authorized batch, temporarily set `published` to false while preparing its new item files.
Publish all items, regenerate the database, then commit the links and reports together.

| Tables or views | Purpose |
| --- | --- |
| `work_item`, `work_dependency` | Acceptance, reproduction, evidence, ownership, issue links, and prerequisites |
| `ready_work`, `blocked_work` | Dependency-aware dispatch |
| `work_conflicts` | Advisory overlap between source starting points |
| `unticketed_work`, `uncovered_gaps` | Missing tickets or incomplete backlog coverage |
| `backlog_state_drift` | Mismatch between reviewed work status and saved GitHub state |

## Enriched investigation review

The [investigation review](INVESTIGATION-REVIEW.md) records 15 implementation conversions and the reasons 16 tickets remain investigations.
Review decisions and links to the original enrichment comments are stored in each work item and in SQLite `evidence_json`.
They do not change capability support ratings or close tickets automatically.
