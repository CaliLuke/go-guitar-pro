#!/usr/bin/env python3
"""Build and query the development capability database using Python's standard library."""

import argparse
import backlog
import collections
from contextlib import closing
import fnmatch
import hashlib
import io
import json
import os
from pathlib import Path
import re
import sqlite3
import subprocess
import tarfile
import tempfile

HERE = Path(__file__).resolve().parent
ROOT = HERE.parent.parent
DATABASE = HERE / "capabilities.sqlite"
REFERENCE = ROOT / "references/alphaTab"
STATUSES = ("supported", "partial", "missing", "unverified", "out-of-scope")


def read_json(path):
    return json.loads(path.read_text())


def packed(value):
    return json.dumps(value, sort_keys=True, separators=(",", ":"), ensure_ascii=False)


def sha(path):
    return hashlib.sha256(path.read_bytes()).hexdigest()


def input_hashes():
    paths = sorted(ROOT.glob("*.go")) + sorted((ROOT / "integration").glob("*.go"))
    paths += [ROOT / "go.mod", ROOT / "go.sum", ROOT / "docs/semantic-model.md"]
    paths += [ROOT / "conformance" / name for name in (
        "oracle.json", "feature-ledger.json", "upstream-inventory.json",
        "fixture-inventory.json", "corpus-snapshot.json", "oracle.mjs")]
    return {str(p.relative_to(ROOT)): sha(p) for p in paths if p.exists()}


def upstream_fixtures():
    oracle = read_json(ROOT / "conformance/oracle.json")
    matches = collections.defaultdict(list)
    for fixture in read_json(ROOT / "conformance/fixture-inventory.json")["fixtures"]:
        data = (ROOT / fixture["path"]).read_bytes()
        blob = hashlib.sha1(b"blob " + str(len(data)).encode() + b"\0" + data).hexdigest()
        matches[blob].append(fixture["path"])
    listing = subprocess.check_output(["git", "-C", str(REFERENCE), "ls-tree", "-r", oracle["sourceRevision"], "packages/alphatab/test-data"], text=True)
    result = []
    for line in listing.splitlines():
        info, path = line.split("\t", 1)
        if not re.match(r"packages/alphatab/test-data/guitarpro[3-8]/.*\.(gp|gpx|gp3|gp4|gp5)$", path):
            continue
        blob = info.split()[2]
        result.append({"path": path, "blob_sha1": blob, "local_matches": matches.get(blob, [])})
    return result


def discover():
    """Lexical discovery for pinned TS style; uncertain symbols remain unverified.

    Unlike the old scanner, preserve the declaring class, including multiple
    classes in one file. Include every enum member, not just the primary model.
    This is a source inventory, not a TypeScript type checker or behavior proof.
    """
    oracle = read_json(ROOT / "conformance/oracle.json")
    legacy = {x["name"]: x["disposition"] for x in
              read_json(ROOT / "conformance/upstream-inventory.json")["modelSymbols"]}
    folders = [f"packages/alphatab/src/{folder}" for folder in ("model", "importer", "exporter", "midi")]
    folders.append("packages/alphatab/test/importer")
    archive = subprocess.check_output(["git", "-C", str(REFERENCE), "archive", oracle["sourceRevision"], *folders])
    files = {}
    with tarfile.open(fileobj=io.BytesIO(archive)) as tar:
        for entry in tar:
            if not entry.isfile() or not entry.name.endswith(".ts"):
                continue
            if "/test/" in entry.name and not Path(entry.name).name.startswith("Gp"):
                continue
            files[entry.name] = tar.extractfile(entry).read()
    constructs, sources = [], {}
    for relative, content in sorted(files.items()):
        sources[relative] = hashlib.sha256(content).hexdigest()
        owner, owner_kind, method = Path(relative).stem, "", ""
        counts = collections.Counter()
        in_comment = False
        for number, original in enumerate(content.decode().splitlines(), 1):
            line = original
            # Pinned declarations are line-oriented. Ignore comments and examples.
            if in_comment:
                if "*/" not in line:
                    continue
                line = line.split("*/", 1)[1]
                in_comment = False
            while "/*" in line:
                before, after = line.split("/*", 1)
                if "*/" in after:
                    line = before + after.split("*/", 1)[1]
                else:
                    line, in_comment = before, True
                    break
            if line.lstrip().startswith("//"):
                continue
            declaration = re.match(r"^(?:export )?(?:abstract )?(class|enum|interface) (\w+)", line)
            found = []
            if declaration:
                owner_kind, owner = declaration.groups()
                method = ""
                found.append((owner_kind, owner))
            member = re.match(r"^    public\s+(?:(?:static|readonly|override|abstract|async)\s+)*(?:(get|set)\s+)?([\w$]+)(.*)", line)
            if member:
                accessor, name, tail = member.groups()
                kind = accessor or ("method" if tail.lstrip().startswith(("(", "<")) else "field")
                if name != "constructor":
                    found.append((kind, f"{owner}.{name}"))
            if owner_kind in ("enum", "interface"):
                member = re.match(r"^    ([\w$]+)\s*(?:[=,}:?(]|$)", line)
                if member:
                    found.append(("enum-member" if owner_kind == "enum" else "interface-member",
                                  f"{owner}.{member[1]}"))
            func = re.match(r"^    (?:private|public|protected)\s+(?:(?:static|async|override)\s+)*(\w+)\s*[(<]", line)
            if func:
                method = func[1]
            case = re.match(r"\s*case\s+(.+?):\s*(?://.*)?$", line)
            if case:
                found.append(("dispatch", f"{owner}.{method}:{case[1].strip(chr(39) + chr(34))}"))
            for kind, name in found:
                key = f"{relative}::{kind}::{name}"
                counts[key] += 1
                constructs.append({"id": f"{key}::{counts[key]}", "path": relative,
                                   "line": number, "kind": kind, "name": name,
                                   "declaration": original.strip(),
                                   "legacy_disposition": legacy.get(name)})
    return sources, constructs


def validate_catalog(catalog, constructs):
    ids = set()
    go_sources = "\n".join(p.read_text() for p in ROOT.rglob("*_test.go") if "references" not in p.parts)
    fields = {f"{t['type']}.{f}" for t in read_json(ROOT / "conformance/feature-ledger.json")["semanticContracts"]["modelTypes"] for f in t["fields"]}
    for cap in catalog["capabilities"]:
        if cap["id"] in ids:
            raise ValueError(f"Duplicate capability {cap['id']}")
        ids.add(cap["id"])
        if set(cap["stages"]) != {"import", "model", "export"} or any(s not in STATUSES for s in cap["stages"].values()):
            raise ValueError(f"Invalid stages: {cap['id']}")
        if not cap["finding"] or not cap["acceptance"] or not cap["evidence"]:
            raise ValueError(f"Missing review evidence: {cap['id']}")
        for pattern in cap.get("upstream", []):
            if not any(fnmatch.fnmatchcase(c["name"], pattern) for c in constructs):
                raise ValueError(f"Unmatched upstream selector {cap['id']}: {pattern}")
        for item in cap["evidence"]:
            if item["kind"] == "source" and not (ROOT / item["reference"].split(":")[0]).is_file():
                raise ValueError(f"Missing source: {item}")
            if item["kind"] == "test" and not re.search(r"func\s+" + re.escape(item["reference"]) + r"\(", go_sources):
                raise ValueError(f"Missing test: {item}")
            if item["kind"] == "field" and item["reference"] not in fields:
                raise ValueError(f"Missing public field: {item}")


def build(path):
    catalog = read_json(HERE / "catalog.json")
    sources, constructs = discover()
    validate_catalog(catalog, constructs)
    con = sqlite3.connect(path)
    con.executescript((HERE / "schema.sql").read_text())
    oracle = read_json(ROOT / "conformance/oracle.json")
    metadata = {"schema_version": "2", "oracle": packed(oracle),
                "reviewed_at": catalog["reviewed_at"], "reviewed_commit": catalog["reviewed_commit"],
                "input_hashes": packed(input_hashes()),
                "scope": "GP3-GP8 import, public authored model, GP8 export; excluded capabilities listed separately",
                "status_basis": "Source review and linked scoped tests. A supported row does not prove all variants.",
                "discovery_limit": "Lexical pinned-source declarations and case labels; not a type checker or every semantic branch."}
    con.executemany("INSERT INTO metadata VALUES (?,?)", metadata.items())
    con.executemany("INSERT INTO source_file VALUES (?,?)", sources.items())
    for item in constructs:
        con.execute("INSERT INTO upstream_construct(id,path,line,kind,name,declaration,legacy_disposition) VALUES (:id,:path,:line,:kind,:name,:declaration,:legacy_disposition)", item)
    for cap in catalog["capabilities"]:
        con.execute("INSERT INTO capability VALUES (?,?,?,?,?,?,?,?)", (
            cap["id"], cap["domain"], cap["title"], cap["scope"], cap["priority"],
            packed(cap["formats"]), cap["finding"], cap["acceptance"]))
        con.executemany("INSERT INTO assessment VALUES (?,?,?)", [(cap["id"], stage, status) for stage, status in cap["stages"].items()])
        con.executemany("INSERT INTO evidence VALUES (?,?,?,?)", [(cap["id"], e["kind"], e["reference"], e.get("detail", "")) for e in cap["evidence"]])
        for c in constructs:
            if any(fnmatch.fnmatchcase(c["name"], pattern) for pattern in cap.get("upstream", [])):
                con.execute("INSERT INTO construct_capability VALUES (?,?)", (c["id"], cap["id"]))
                # Association gives a route to the review; it is not individual behavior evidence.
                con.execute("UPDATE upstream_construct SET review_status='linked' WHERE id=?", (c["id"],))
    ledger = read_json(ROOT / "conformance/feature-ledger.json")
    for kind, dispositions, cases in (
        ("public-field", "fieldDispositions", "fieldCases"),
        ("wire-field", "wireFieldDispositions", "wireFieldCases")):
        for disposition, names in ledger["semanticContracts"][dispositions].items():
            for name in names:
                con.execute("INSERT INTO obligation VALUES (?,?,?,?)", (kind, name, disposition, packed(ledger["semanticMatrix"][cases].get(name, []))))
    for dispatch in ledger["semanticContracts"]["sourceDispatches"]:
        for name, disposition in dispatch["cases"].items():
            con.execute("INSERT INTO obligation VALUES (?,?,?,?)", ("go-dispatch", f"{dispatch['function']}:{dispatch['selector']}:{name}", disposition, packed(dispatch)))
    for case in ledger["semanticMatrix"]["cases"]:
        con.execute("INSERT INTO matrix_case VALUES (?,?)", (case["id"], packed(case)))
    for name, cases in ledger["semanticMatrix"]["enumCases"].items():
        con.execute("INSERT INTO obligation VALUES (?,?,?,?)", ("public-enum", name, "behavior-linked", packed(cases)))
    for f in read_json(ROOT / "conformance/fixture-inventory.json")["fixtures"]:
        con.execute("INSERT INTO fixture VALUES (?,?,?,?)", (f["path"], f["format"], f["sha256"], packed(f)))
    for f in read_json(ROOT / "conformance/corpus-snapshot.json")["fixtures"]:
        for d in f["diagnostics"]:
            con.execute("INSERT INTO corpus_diagnostic VALUES (?,?,?,?,?)", (f["path"], d["code"], d["kind"], d["count"], packed(d)))
    for t in read_json(ROOT / "conformance/upstream-inventory.json")["importerCases"]:
        con.execute("INSERT INTO upstream_test VALUES (?,?)", (t["name"], packed(t)))
    for f in upstream_fixtures():
        con.execute("INSERT INTO upstream_fixture VALUES (?,?,?)", (f["path"], f["blob_sha1"], packed(f["local_matches"])))
    for issue in catalog.get("issues", []):
        con.execute("INSERT INTO issue VALUES (:number,:state,:url,:title,:checked_at)", issue)
    website = read_json(HERE / "website-features.json")
    con.execute("INSERT INTO metadata VALUES ('website_fetched_at',?)", (website["fetched_at"],))
    for page in website["pages"]:
        for row in page["rows"]:
            mapping = catalog["website_mapping"].get(row["feature"])
            if not mapping:
                raise ValueError(f"Unmapped website feature: {row['feature']}")
            cursor = con.execute("INSERT INTO website_feature(url,format,domain,feature,model,reading,rendering,audio,alphatex) VALUES (?,?,?,?,?,?,?,?,?)", (
                page["url"], page["format"], row["domain"], row["feature"], row["model"], row["reading"], row["rendering"], row["audio"], row["alphaTex"]))
            con.executemany("INSERT INTO website_capability VALUES (?,?)", [(cursor.lastrowid, cap) for cap in mapping])
    if (HERE / "probe-results.json").exists():
        probe = read_json(HERE / "probe-results.json")
        con.execute("INSERT INTO metadata VALUES ('probe_metadata',?)", (packed({k: v for k, v in probe.items() if k != "fixtures"}),))
        fresh = (probe["input_hashes"] == input_hashes() and probe["oracle"] == oracle
                 and probe["probe_sha256"] == sha(HERE / "probe.mjs")
                 and probe.get("bridge_sha256") == sha(HERE / "probe.go"))
        con.execute("INSERT INTO metadata VALUES ('probe_fresh',?)", (str(fresh).lower(),))
        for f in probe["fixtures"]:
            con.execute("INSERT INTO probe_result VALUES (?,?,?,?,?,?)", (f["path"], f.get("parseError"), f.get("exportError"), f.get("alphaError"), packed(f["diagnostics"]), packed(f["report"])))
            for c in f["comparisons"]:
                con.execute("INSERT INTO probe_comparison VALUES (?,?,?,?,?,?,?)", (f["path"], c["capability"], len(c["source"]), None if c["target"] is None else len(c["target"]), c["equal"], packed(c["source"]), None if c["target"] is None else packed(c["target"])))
    backlog.populate(con, catalog)
    con.commit()
    if con.execute("PRAGMA integrity_check").fetchone()[0] != "ok" or con.execute("PRAGMA foreign_key_check").fetchall():
        raise ValueError("Database integrity check failed")
    con.close()


def logical_dump(path):
    with closing(sqlite3.connect(f"file:{path}?mode=ro", uri=True)) as con:
        return "\n".join(con.iterdump())


def refresh(check=False):
    descriptor, temporary = tempfile.mkstemp(suffix=".sqlite", dir=HERE)
    os.close(descriptor)
    try:
        build(temporary)
        if check:
            if not DATABASE.exists() or logical_dump(temporary) != logical_dump(DATABASE):
                raise ValueError("Capability database is stale. Review changes, then run manage.py refresh.")
            if not (HERE / "CAPABILITIES.md").exists() or (HERE / "CAPABILITIES.md").read_text() != report(temporary):
                raise ValueError("Capability report is stale. Run manage.py refresh.")
            if (HERE / "BACKLOG.md").read_text() != backlog.report(backlog.load(), backlog.read(backlog.META)):
                raise ValueError("Backlog report is stale. Run manage.py refresh.")
            print("Capability database matches catalog, source inventory, ledger, and probe receipt.")
        else:
            markdown = report(temporary)
            os.replace(temporary, DATABASE)
            (HERE / "CAPABILITIES.md").write_text(markdown)
            (HERE / "BACKLOG.md").write_text(backlog.report(backlog.load(), backlog.read(backlog.META)))
            print(f"Updated {DATABASE.relative_to(ROOT)}")
    finally:
        Path(temporary).unlink(missing_ok=True)


def query(sql):
    with closing(sqlite3.connect(f"file:{DATABASE}?mode=ro", uri=True)) as con:
        cursor = con.execute(sql)
        print("\t".join(c[0] for c in cursor.description))
        for row in cursor:
            print("\t".join("" if v is None else str(v) for v in row))


def report(path):
    with closing(sqlite3.connect(path)) as con:
        con.row_factory = sqlite3.Row
        meta = dict(con.execute("SELECT key,value FROM metadata"))
        oracle = json.loads(meta["oracle"])
        count = lambda table: con.execute(f"SELECT COUNT(*) FROM {table}").fetchone()[0]
        lines = ["# Capability inventory", "", "Generated by `python3 conformance/capabilities/manage.py refresh`.", "",
                 f"Review date: {meta['reviewed_at']}. AlphaTab: {oracle['version']} (`{oracle['sourceRevision']}`).", "",
                 f"The inventory contains {count('capability')} capability rows and {count('website_feature')} mapped format-documentation rows.",
                 f"The pinned source inventory contains {count('upstream_construct')} constructs from {count('source_file')} files.",
                 f"There are {count('missing_upstream_fixtures')} upstream fixtures without a byte-identical local fixture.", "",
                 "Support ratings describe the scope in each finding. Partial and unverified rows receive no completion credit.",
                 "A linked source construct has a capability association, not individual behavior proof.", "",
                 "| Stage | Supported | Partial | Missing | Unverified |", "| --- | ---: | ---: | ---: | ---: |"]
        for stage in ("import", "model", "export"):
            counts = dict(con.execute("SELECT status,capabilities FROM progress WHERE stage=?", (stage,)))
            lines.append(f"| {stage} | " + " | ".join(str(counts.get(s, 0)) for s in STATUSES[:4]) + " |")
        complete = con.execute("SELECT COUNT(*) FROM capability_status WHERE scope='guitar-pro' AND import_status='supported' AND model_status='supported' AND export_status='supported'").fetchone()[0]
        lines += ["", f"All three stages have a supported rating in {complete} rows. This is a checklist count, not a percentage of all musical behavior.", "",
                  "## Runtime probe", "", f"The receipt contains {count('probe_result')} input files. Probe freshness against the current source: `{meta.get('probe_fresh', 'unavailable')}`.",
                  "Raw consumer differences require review. Default-only cases do not prove feature support.", "",
                  "| Capability | Files with non-default source values | Files with differences | Blocked comparisons |",
                  "| --- | ---: | ---: | ---: |"]
        for row in con.execute("SELECT * FROM probe_coverage WHERE nondefault_fixtures>0 OR differing_fixtures>0 ORDER BY differing_fixtures DESC,title"):
            lines.append(f"| {row['title']} | {row['nondefault_fixtures']} | {row['differing_fixtures']} | {row['blocked_fixtures']} |")
        lines += ["", "## Reviewed capabilities", "", "Priority 1 affects core semantics or audit confidence. Priority 2 covers techniques and expressions. Priority 3 covers display details or scope extensions.", ""]
        for row in con.execute("SELECT * FROM capability_status ORDER BY scope DESC,domain,priority,id"):
            lines += [f"### {row['id']}: {row['title']}", "",
                      f"{row['domain']}. Priority {row['priority']}. Formats: {', '.join(json.loads(row['formats']))}. Scope: {row['scope']}.", "",
                      f"Import: **{row['import_status']}**. Model: **{row['model_status']}**. GP8 export: **{row['export_status']}**.", "",
                      row['finding'], "", "Completion criterion: " + row['acceptance'], ""]
            work = con.execute("SELECT id,title,issue_url FROM work_item WHERE capability_id=? ORDER BY priority,id", (row["id"],)).fetchall()
            if work:
                links = [f"[{w['title']}]({w['issue_url'] or ('work-items/' + w['id'] + '.json')})" for w in work]
                lines += ["Bounded work: " + "; ".join(links) + ".", ""]
        return "\n".join(lines)


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    sub = parser.add_subparsers(dest="command", required=True)
    sub.add_parser("refresh")
    sub.add_parser("check")
    sub.add_parser("summary")
    sub.add_parser("gaps")
    sub.add_parser("ready")
    q = sub.add_parser("query")
    q.add_argument("sql")
    args = parser.parse_args()
    if args.command in ("refresh", "check"):
        refresh(args.command == "check")
    elif args.command == "summary":
        query("SELECT * FROM progress ORDER BY stage,status")
    elif args.command == "ready":
        query("SELECT id,kind,priority,title,issue_url FROM ready_work ORDER BY priority,kind,id")
    elif args.command == "gaps":
        query("SELECT id,priority,title,import_status,model_status,export_status FROM gaps ORDER BY priority,domain,id")
    else:
        query(args.sql)


if __name__ == "__main__":
    main()
