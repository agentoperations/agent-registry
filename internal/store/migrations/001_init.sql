CREATE TABLE IF NOT EXISTS artifacts (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    kind            TEXT NOT NULL CHECK(kind IN ('agent', 'skill', 'mcp-server')),
    namespace       TEXT NOT NULL,
    name            TEXT NOT NULL,
    version         TEXT NOT NULL,
    title           TEXT NOT NULL,
    description     TEXT NOT NULL,
    status          TEXT NOT NULL DEFAULT 'draft'
                        CHECK(status IN ('draft','evaluated','approved','published','deprecated','archived')),
    is_latest       BOOLEAN NOT NULL DEFAULT 0,
    category        TEXT,
    tags            TEXT,
    license         TEXT,
    body            TEXT NOT NULL,
    content_hash    TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now')),
    updated_at      TEXT NOT NULL DEFAULT (datetime('now')),
    published_at    TEXT,
    UNIQUE(name, version)
);

CREATE INDEX IF NOT EXISTS idx_artifacts_kind ON artifacts(kind);
CREATE INDEX IF NOT EXISTS idx_artifacts_namespace ON artifacts(namespace);
CREATE INDEX IF NOT EXISTS idx_artifacts_name ON artifacts(name);
CREATE INDEX IF NOT EXISTS idx_artifacts_status ON artifacts(status);
CREATE INDEX IF NOT EXISTS idx_artifacts_kind_status ON artifacts(kind, status);
CREATE INDEX IF NOT EXISTS idx_artifacts_category ON artifacts(category);

CREATE VIRTUAL TABLE IF NOT EXISTS artifacts_fts USING fts5(
    name, title, description, tags, category,
    content=artifacts, content_rowid=id
);

CREATE TRIGGER IF NOT EXISTS artifacts_ai AFTER INSERT ON artifacts BEGIN
    INSERT INTO artifacts_fts(rowid, name, title, description, tags, category)
    VALUES (new.id, new.name, new.title, new.description, new.tags, new.category);
END;

CREATE TRIGGER IF NOT EXISTS artifacts_ad AFTER DELETE ON artifacts BEGIN
    INSERT INTO artifacts_fts(artifacts_fts, rowid, name, title, description, tags, category)
    VALUES ('delete', old.id, old.name, old.title, old.description, old.tags, old.category);
END;

CREATE TRIGGER IF NOT EXISTS artifacts_au AFTER UPDATE ON artifacts BEGIN
    INSERT INTO artifacts_fts(artifacts_fts, rowid, name, title, description, tags, category)
    VALUES ('delete', old.id, old.name, old.title, old.description, old.tags, old.category);
    INSERT INTO artifacts_fts(rowid, name, title, description, tags, category)
    VALUES (new.id, new.name, new.title, new.description, new.tags, new.category);
END;

CREATE TABLE IF NOT EXISTS eval_records (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    eval_id         TEXT NOT NULL UNIQUE,
    artifact_id     INTEGER NOT NULL REFERENCES artifacts(id) ON DELETE CASCADE,
    category        TEXT NOT NULL DEFAULT 'functional'
                        CHECK(category IN ('functional','safety','red-team','performance','custom')),
    provider_name   TEXT,
    provider_version TEXT,
    benchmark_name  TEXT NOT NULL,
    benchmark_version TEXT,
    evaluator_name  TEXT NOT NULL,
    evaluator_method TEXT NOT NULL CHECK(evaluator_method IN ('automated','human','hybrid','llm-as-judge')),
    overall_score   REAL NOT NULL CHECK(overall_score >= 0 AND overall_score <= 1),
    body            TEXT NOT NULL,
    evaluated_at    TEXT,
    created_at      TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_evals_artifact ON eval_records(artifact_id);
CREATE INDEX IF NOT EXISTS idx_evals_category ON eval_records(category);

CREATE TABLE IF NOT EXISTS trust_signals (
    artifact_id     INTEGER PRIMARY KEY REFERENCES artifacts(id) ON DELETE CASCADE,
    composite_score REAL NOT NULL CHECK(composite_score >= 0 AND composite_score <= 1),
    body            TEXT NOT NULL,
    calculated_at   TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE TABLE IF NOT EXISTS promotion_log (
    id              INTEGER PRIMARY KEY AUTOINCREMENT,
    artifact_id     INTEGER NOT NULL REFERENCES artifacts(id) ON DELETE CASCADE,
    from_status     TEXT NOT NULL,
    to_status       TEXT NOT NULL,
    comment         TEXT,
    promoted_at     TEXT NOT NULL DEFAULT (datetime('now'))
);

CREATE INDEX IF NOT EXISTS idx_promotion_log_artifact ON promotion_log(artifact_id);
