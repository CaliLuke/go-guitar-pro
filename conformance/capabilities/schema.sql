PRAGMA foreign_keys = ON;
CREATE TABLE metadata (key TEXT PRIMARY KEY, value TEXT NOT NULL);
CREATE TABLE capability (
  id TEXT PRIMARY KEY,
  domain TEXT NOT NULL,
  title TEXT NOT NULL,
  scope TEXT NOT NULL CHECK (scope IN ('guitar-pro', 'excluded')),
  priority INTEGER NOT NULL CHECK (priority BETWEEN 1 AND 3),
  formats TEXT NOT NULL,
  finding TEXT NOT NULL,
  acceptance TEXT NOT NULL
);
CREATE TABLE assessment (
  capability_id TEXT NOT NULL REFERENCES capability(id),
  stage TEXT NOT NULL CHECK (stage IN ('import', 'model', 'export')),
  status TEXT NOT NULL CHECK (status IN ('supported', 'partial', 'missing', 'unverified', 'out-of-scope')),
  PRIMARY KEY (capability_id, stage)
);
CREATE TABLE evidence (
  capability_id TEXT NOT NULL REFERENCES capability(id),
  kind TEXT NOT NULL,
  reference TEXT NOT NULL,
  detail TEXT NOT NULL,
  PRIMARY KEY (capability_id, kind, reference)
);
CREATE TABLE upstream_construct (
  id TEXT PRIMARY KEY,
  path TEXT NOT NULL,
  line INTEGER NOT NULL,
  kind TEXT NOT NULL,
  name TEXT NOT NULL,
  declaration TEXT NOT NULL,
  legacy_disposition TEXT,
  review_status TEXT NOT NULL DEFAULT 'unverified',
  source_scope TEXT NOT NULL DEFAULT 'unclassified',
  source_formats TEXT NOT NULL DEFAULT '[]',
  scope_reason TEXT NOT NULL DEFAULT ''
);
CREATE TABLE construct_capability (
  construct_id TEXT NOT NULL REFERENCES upstream_construct(id),
  capability_id TEXT NOT NULL REFERENCES capability(id),
  PRIMARY KEY (construct_id, capability_id)
);
CREATE TABLE source_file (path TEXT PRIMARY KEY, sha256 TEXT NOT NULL);
CREATE TABLE upstream_review (
  construct_id TEXT PRIMARY KEY REFERENCES upstream_construct(id),
  source_scope TEXT NOT NULL,
  formats_json TEXT NOT NULL,
  disposition TEXT NOT NULL CHECK (disposition IN ('authored','derived','excluded')),
  reason TEXT NOT NULL,
  primary_capability TEXT REFERENCES capability(id),
  model_declaration TEXT
);
CREATE TABLE upstream_secondary_capability (
  construct_id TEXT NOT NULL REFERENCES upstream_review(construct_id),
  capability_id TEXT NOT NULL REFERENCES capability(id),
  PRIMARY KEY (construct_id, capability_id)
);
CREATE TABLE upstream_model_review (
  declaration TEXT PRIMARY KEY,
  construct_id TEXT NOT NULL REFERENCES upstream_construct(id),
  formats_json TEXT NOT NULL,
  disposition TEXT NOT NULL CHECK (disposition='authored'),
  reason TEXT NOT NULL,
  primary_capability TEXT NOT NULL REFERENCES capability(id)
);
CREATE TABLE obligation (
  kind TEXT NOT NULL,
  name TEXT NOT NULL,
  disposition TEXT NOT NULL,
  cases_json TEXT NOT NULL,
  PRIMARY KEY (kind, name)
);
CREATE TABLE matrix_case (id TEXT PRIMARY KEY, data_json TEXT NOT NULL);
CREATE TABLE fixture (
  path TEXT PRIMARY KEY,
  format TEXT NOT NULL,
  sha256 TEXT NOT NULL,
  data_json TEXT NOT NULL
);
CREATE TABLE corpus_diagnostic (
  fixture_path TEXT NOT NULL REFERENCES fixture(path),
  code TEXT NOT NULL,
  kind TEXT NOT NULL,
  count INTEGER NOT NULL,
  data_json TEXT NOT NULL
);
CREATE TABLE upstream_test (name TEXT PRIMARY KEY, data_json TEXT NOT NULL);
CREATE TABLE upstream_fixture (path TEXT PRIMARY KEY, blob_sha1 TEXT NOT NULL, local_matches_json TEXT NOT NULL);
CREATE VIEW missing_upstream_fixtures AS SELECT * FROM upstream_fixture WHERE local_matches_json='[]';
CREATE TABLE probe_result (
  fixture_path TEXT PRIMARY KEY,
  parse_error TEXT,
  export_error TEXT,
  source_consumer_error TEXT,
  target_consumer_error TEXT,
  diagnostics_json TEXT NOT NULL,
  export_report_json TEXT NOT NULL
);
CREATE TABLE probe_comparison (
  fixture_path TEXT NOT NULL REFERENCES probe_result(fixture_path),
  capability_id TEXT NOT NULL REFERENCES capability(id),
  source_count INTEGER,
  target_count INTEGER,
  equal_value INTEGER,
  comparison_status TEXT NOT NULL CHECK (comparison_status IN ('default-only','equal-nondefault','different','source-blocked','target-blocked')),
  source_json TEXT NOT NULL,
  target_json TEXT,
  PRIMARY KEY (fixture_path, capability_id)
);
CREATE TABLE support_claim (
  capability_id TEXT NOT NULL REFERENCES capability(id),
  stage TEXT NOT NULL CHECK (stage IN ('import','model','export')),
  case_id TEXT NOT NULL REFERENCES matrix_case(id),
  source TEXT NOT NULL,
  value TEXT NOT NULL,
  assertion_stage TEXT NOT NULL,
  obligation TEXT NOT NULL,
  serialization TEXT,
  report_assertion TEXT,
  report_not_applicable TEXT,
  independent_evidence TEXT,
  independent_limit TEXT,
  PRIMARY KEY (capability_id,stage)
);
CREATE TABLE issue (number INTEGER PRIMARY KEY, state TEXT NOT NULL, url TEXT NOT NULL, title TEXT NOT NULL, checked_at TEXT NOT NULL);
CREATE TABLE website_feature (
  id INTEGER PRIMARY KEY,
  url TEXT NOT NULL,
  format TEXT NOT NULL,
  domain TEXT NOT NULL,
  feature TEXT NOT NULL,
  model TEXT NOT NULL,
  reading TEXT NOT NULL,
  rendering TEXT NOT NULL,
  audio TEXT NOT NULL,
  alphatex TEXT NOT NULL
);
CREATE TABLE website_capability (
  website_id INTEGER NOT NULL REFERENCES website_feature(id),
  capability_id TEXT NOT NULL REFERENCES capability(id),
  PRIMARY KEY (website_id, capability_id)
);
CREATE VIEW website_comparison AS
SELECT w.format,w.domain,w.feature,w.model AS website_model,w.reading AS website_reading,
  c.id,c.import_status,c.model_status,c.export_status,c.finding,w.url
FROM website_feature w JOIN website_capability m ON m.website_id=w.id
JOIN capability_status c ON c.id=m.capability_id;
CREATE VIEW capability_status AS
SELECT c.*,
  MAX(CASE WHEN a.stage='import' THEN a.status END) AS import_status,
  MAX(CASE WHEN a.stage='model' THEN a.status END) AS model_status,
  MAX(CASE WHEN a.stage='export' THEN a.status END) AS export_status
FROM capability c JOIN assessment a ON a.capability_id=c.id GROUP BY c.id;
CREATE VIEW progress AS
SELECT a.stage, a.status, COUNT(*) AS capabilities,
  ROUND(100.0*COUNT(*)/(SELECT COUNT(*) FROM capability WHERE scope='guitar-pro'), 1) AS percent_of_scope
FROM assessment a JOIN capability c ON c.id=a.capability_id
WHERE c.scope='guitar-pro' GROUP BY a.stage,a.status;
CREATE VIEW gaps AS
SELECT * FROM capability_status WHERE scope='guitar-pro'
AND (import_status<>'supported' OR model_status<>'supported' OR export_status<>'supported');
CREATE VIEW unreviewed_constructs AS
SELECT u.* FROM upstream_construct u WHERE NOT EXISTS
  (SELECT 1 FROM construct_capability m WHERE m.construct_id=u.id)
AND NOT EXISTS (SELECT 1 FROM upstream_review r WHERE r.construct_id=u.id);
CREATE VIEW authored_upstream_constructs AS
SELECT u.*,r.disposition,r.reason,r.primary_capability,r.model_declaration
FROM upstream_construct u JOIN upstream_review r ON r.construct_id=u.id
WHERE r.disposition='authored';
CREATE VIEW observed_differences AS
SELECT p.fixture_path,c.title,p.comparison_status,p.source_count,p.target_count,p.source_json,p.target_json
FROM probe_comparison p JOIN capability c ON c.id=p.capability_id WHERE comparison_status='different';
CREATE VIEW probe_coverage AS
SELECT c.id,c.title,COUNT(p.fixture_path) AS fixtures,
  SUM(CASE WHEN p.source_count>0 THEN 1 ELSE 0 END) AS nondefault_fixtures,
  SUM(CASE WHEN p.comparison_status='default-only' THEN 1 ELSE 0 END) AS default_only_fixtures,
  SUM(CASE WHEN p.comparison_status='different' THEN 1 ELSE 0 END) AS differing_fixtures,
  SUM(CASE WHEN p.comparison_status='source-blocked' OR r.source_consumer_error IS NOT NULL THEN 1 ELSE 0 END) AS source_blocked_fixtures,
  SUM(CASE WHEN p.comparison_status='target-blocked' OR r.target_consumer_error IS NOT NULL THEN 1 ELSE 0 END) AS target_blocked_fixtures,
  SUM(CASE WHEN p.comparison_status IN ('source-blocked','target-blocked') OR r.source_consumer_error IS NOT NULL OR r.target_consumer_error IS NOT NULL THEN 1 ELSE 0 END) AS blocked_fixtures
FROM capability c
LEFT JOIN probe_comparison p ON c.id=p.capability_id
LEFT JOIN probe_result r ON r.fixture_path=p.fixture_path
GROUP BY c.id;
CREATE VIEW supported_without_claims AS
SELECT a.capability_id,a.stage FROM assessment a
LEFT JOIN support_claim s ON s.capability_id=a.capability_id AND s.stage=a.stage
WHERE a.status='supported' AND s.capability_id IS NULL;
CREATE TABLE work_item (
  id TEXT PRIMARY KEY,
  capability_id TEXT NOT NULL REFERENCES capability(id),
  title TEXT NOT NULL,
  kind TEXT NOT NULL CHECK (kind IN ('implementation','investigation')),
  priority INTEGER NOT NULL CHECK (priority BETWEEN 1 AND 3),
  status TEXT NOT NULL CHECK (status IN ('todo','in_progress','done')),
  owner TEXT,
  scope TEXT NOT NULL,
  formats_json TEXT NOT NULL,
  stages_json TEXT NOT NULL,
  acceptance_json TEXT NOT NULL,
  exclusions_json TEXT NOT NULL,
  reproduction_json TEXT NOT NULL,
  evidence_json TEXT NOT NULL,
  shared_files_json TEXT NOT NULL,
  resolution TEXT,
  verification_json TEXT NOT NULL,
  issue_number INTEGER UNIQUE,
  issue_url TEXT,
  issue_state TEXT
);
CREATE TABLE work_dependency (
  work_id TEXT NOT NULL REFERENCES work_item(id),
  depends_on TEXT NOT NULL REFERENCES work_item(id),
  PRIMARY KEY(work_id,depends_on), CHECK(work_id<>depends_on)
);
CREATE VIEW ready_work AS
SELECT w.* FROM work_item w WHERE w.status='todo'
AND (w.issue_state IS NULL OR w.issue_state='OPEN')
AND NOT EXISTS (SELECT 1 FROM work_dependency d JOIN work_item p ON p.id=d.depends_on
  WHERE d.work_id=w.id AND p.status<>'done')
AND NOT EXISTS (SELECT 1 FROM json_each(w.reproduction_json,'$.external_inputs') i
  WHERE json_extract(i.value,'$.state')='missing');
CREATE VIEW external_input_blockers AS
SELECT w.id,w.title,json_extract(i.value,'$.id') AS input_id,
  json_extract(i.value,'$.requirement') AS requirement,
  json_extract(i.value,'$.acquisition') AS acquisition
FROM work_item w JOIN json_each(w.reproduction_json,'$.external_inputs') i
WHERE w.status<>'done' AND json_extract(i.value,'$.state')='missing';
CREATE VIEW blocked_work AS
SELECT w.id,w.title,p.id AS dependency,p.title AS dependency_title,p.status
FROM work_item w JOIN work_dependency d ON d.work_id=w.id
JOIN work_item p ON p.id=d.depends_on WHERE w.status<>'done' AND p.status<>'done';
CREATE VIEW unticketed_work AS SELECT * FROM work_item WHERE issue_number IS NULL;
CREATE VIEW uncovered_gaps AS SELECT g.* FROM gaps g WHERE NOT EXISTS
  (SELECT 1 FROM work_item w WHERE w.capability_id=g.id);
CREATE VIEW backlog_state_drift AS SELECT * FROM work_item
WHERE (issue_state='CLOSED' AND status<>'done') OR (status='done' AND issue_state='OPEN');
CREATE VIEW work_conflicts AS
SELECT DISTINCT a.id AS work_id,b.id AS other_work_id,j.value AS shared_file
FROM work_item a JOIN work_item b ON a.id<b.id
JOIN json_each(a.shared_files_json) j JOIN json_each(b.shared_files_json) k ON j.value=k.value
WHERE a.status<>'done' AND b.status<>'done';
