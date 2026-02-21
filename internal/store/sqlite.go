package store

import (
	"context"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/agentoperations/agent-registry/internal/model"
	_ "modernc.org/sqlite"
)

//go:embed migrations/001_init.sql
var migrationSQL string

type SQLiteStore struct {
	db *sql.DB
}

func NewSQLiteStore(dbPath string) (*SQLiteStore, error) {
	db, err := sql.Open("sqlite", dbPath+"?_pragma=journal_mode(wal)&_pragma=foreign_keys(1)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if _, err := db.Exec(migrationSQL); err != nil {
		return nil, fmt.Errorf("run migrations: %w", err)
	}
	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Close() error { return s.db.Close() }

func (s *SQLiteStore) InTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if err := fn(ctx); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *SQLiteStore) CreateArtifact(ctx context.Context, a *model.RegistryArtifact) (int64, error) {
	body, err := json.Marshal(a)
	if err != nil {
		return 0, fmt.Errorf("marshal artifact: %w", err)
	}
	hash := sha256.Sum256(body)
	tags := ""
	if a.Metadata != nil && len(a.Metadata.Tags) > 0 {
		tagsJSON, _ := json.Marshal(a.Metadata.Tags)
		tags = string(tagsJSON)
	}
	category := ""
	license := ""
	if a.Metadata != nil {
		category = a.Metadata.Category
		license = a.Metadata.License
	}
	now := time.Now().UTC().Format(time.RFC3339)
	a.Status = model.StatusDraft
	a.CreatedAt = now
	a.UpdatedAt = now

	res, err := s.db.ExecContext(ctx,
		`INSERT INTO artifacts (kind, namespace, name, version, title, description, status, category, tags, license, body, content_hash, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		string(a.Kind), a.Namespace(), a.Identity.Name, a.Identity.Version,
		a.Identity.Title, a.Identity.Description,
		string(model.StatusDraft), category, tags, license,
		string(body), hex.EncodeToString(hash[:]), now, now,
	)
	if err != nil {
		if strings.Contains(err.Error(), "UNIQUE constraint") {
			return 0, ErrAlreadyExists
		}
		return 0, fmt.Errorf("insert artifact: %w", err)
	}
	id, _ := res.LastInsertId()
	return id, nil
}

func (s *SQLiteStore) GetArtifact(ctx context.Context, kind model.Kind, name, version string) (*model.RegistryArtifact, int64, error) {
	var id int64
	var body string
	var status string
	var isLatest bool
	var createdAt, updatedAt string
	var publishedAt sql.NullString

	err := s.db.QueryRowContext(ctx,
		`SELECT id, body, status, is_latest, created_at, updated_at, published_at FROM artifacts WHERE kind = ? AND name = ? AND version = ?`,
		string(kind), name, version,
	).Scan(&id, &body, &status, &isLatest, &createdAt, &updatedAt, &publishedAt)
	if err == sql.ErrNoRows {
		return nil, 0, ErrNotFound
	}
	if err != nil {
		return nil, 0, fmt.Errorf("get artifact: %w", err)
	}

	var a model.RegistryArtifact
	if err := json.Unmarshal([]byte(body), &a); err != nil {
		return nil, 0, fmt.Errorf("unmarshal artifact: %w", err)
	}
	a.Status = model.Status(status)
	a.IsLatest = isLatest
	a.CreatedAt = createdAt
	a.UpdatedAt = updatedAt
	if publishedAt.Valid {
		a.PublishedAt = publishedAt.String
	}
	return &a, id, nil
}

func (s *SQLiteStore) GetLatestArtifact(ctx context.Context, kind model.Kind, name string) (*model.RegistryArtifact, int64, error) {
	var id int64
	var body string
	var status string
	var isLatest bool
	var createdAt, updatedAt string
	var publishedAt sql.NullString
	var version string

	err := s.db.QueryRowContext(ctx,
		`SELECT id, version, body, status, is_latest, created_at, updated_at, published_at
		 FROM artifacts WHERE kind = ? AND name = ?
		 ORDER BY created_at DESC LIMIT 1`,
		string(kind), name,
	).Scan(&id, &version, &body, &status, &isLatest, &createdAt, &updatedAt, &publishedAt)
	if err == sql.ErrNoRows {
		return nil, 0, ErrNotFound
	}
	if err != nil {
		return nil, 0, fmt.Errorf("get latest artifact: %w", err)
	}

	var a model.RegistryArtifact
	if err := json.Unmarshal([]byte(body), &a); err != nil {
		return nil, 0, fmt.Errorf("unmarshal artifact: %w", err)
	}
	a.Status = model.Status(status)
	a.IsLatest = isLatest
	a.CreatedAt = createdAt
	a.UpdatedAt = updatedAt
	if publishedAt.Valid {
		a.PublishedAt = publishedAt.String
	}
	return &a, id, nil
}

func (s *SQLiteStore) ListArtifacts(ctx context.Context, filter *model.ArtifactFilter, limit, offset int) ([]*model.RegistryArtifact, int, error) {
	where := []string{"1=1"}
	args := []interface{}{}

	if filter.Kind != "" {
		where = append(where, "kind = ?")
		args = append(args, string(filter.Kind))
	}
	if filter.Namespace != "" {
		where = append(where, "namespace = ?")
		args = append(args, filter.Namespace)
	}
	if filter.Status != "" {
		where = append(where, "status = ?")
		args = append(args, string(filter.Status))
	}
	if filter.Category != "" {
		where = append(where, "category = ?")
		args = append(args, filter.Category)
	}

	whereClause := strings.Join(where, " AND ")

	var total int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM artifacts WHERE "+whereClause, args...).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count artifacts: %w", err)
	}

	rows, err := s.db.QueryContext(ctx,
		"SELECT body, status, is_latest, created_at, updated_at, published_at FROM artifacts WHERE "+whereClause+" ORDER BY name, created_at DESC LIMIT ? OFFSET ?",
		append(args, limit, offset)...,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list artifacts: %w", err)
	}
	defer rows.Close()

	return scanArtifacts(rows, total)
}

func (s *SQLiteStore) ListVersions(ctx context.Context, kind model.Kind, name string) ([]*model.RegistryArtifact, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT body, status, is_latest, created_at, updated_at, published_at
		 FROM artifacts WHERE kind = ? AND name = ? ORDER BY created_at DESC`,
		string(kind), name,
	)
	if err != nil {
		return nil, fmt.Errorf("list versions: %w", err)
	}
	defer rows.Close()
	artifacts, _, err := scanArtifacts(rows, 0)
	return artifacts, err
}

func (s *SQLiteStore) DeleteArtifact(ctx context.Context, kind model.Kind, name, version string) error {
	res, err := s.db.ExecContext(ctx,
		`DELETE FROM artifacts WHERE kind = ? AND name = ? AND version = ? AND status = 'draft'`,
		string(kind), name, version,
	)
	if err != nil {
		return fmt.Errorf("delete artifact: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *SQLiteStore) UpdateArtifactBody(ctx context.Context, kind model.Kind, name, version string, a *model.RegistryArtifact) error {
	body, err := json.Marshal(a)
	if err != nil {
		return fmt.Errorf("marshal artifact: %w", err)
	}
	now := time.Now().UTC().Format(time.RFC3339)
	tags := ""
	if a.Metadata != nil && len(a.Metadata.Tags) > 0 {
		tagsJSON, _ := json.Marshal(a.Metadata.Tags)
		tags = string(tagsJSON)
	}
	category := ""
	if a.Metadata != nil {
		category = a.Metadata.Category
	}

	res, err := s.db.ExecContext(ctx,
		`UPDATE artifacts SET body = ?, title = ?, description = ?, category = ?, tags = ?, updated_at = ?
		 WHERE kind = ? AND name = ? AND version = ? AND status = 'draft'`,
		string(body), a.Identity.Title, a.Identity.Description, category, tags, now,
		string(kind), name, version,
	)
	if err != nil {
		return fmt.Errorf("update artifact: %w", err)
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrImmutable
	}
	return nil
}

func (s *SQLiteStore) SetStatus(ctx context.Context, kind model.Kind, name, version string, status model.Status) error {
	now := time.Now().UTC().Format(time.RFC3339)
	publishedAt := sql.NullString{}
	if status == model.StatusPublished {
		publishedAt = sql.NullString{String: now, Valid: true}
	}

	_, err := s.db.ExecContext(ctx,
		`UPDATE artifacts SET status = ?, updated_at = ?, published_at = COALESCE(?, published_at)
		 WHERE kind = ? AND name = ? AND version = ?`,
		string(status), now, publishedAt,
		string(kind), name, version,
	)
	return err
}

func (s *SQLiteStore) UpdateLatestFlag(ctx context.Context, kind model.Kind, name string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE artifacts SET is_latest = 0 WHERE kind = ? AND name = ?`, string(kind), name)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		`UPDATE artifacts SET is_latest = 1
		 WHERE id = (SELECT id FROM artifacts WHERE kind = ? AND name = ? ORDER BY created_at DESC LIMIT 1)`,
		string(kind), name,
	)
	return err
}

// Eval records

func (s *SQLiteStore) CreateEvalRecord(ctx context.Context, artifactID int64, eval *model.EvalRecord) error {
	body, err := json.Marshal(eval)
	if err != nil {
		return fmt.Errorf("marshal eval: %w", err)
	}
	evaluatedAt := ""
	if eval.Context != nil {
		evaluatedAt = eval.Context.StartedAt
	}
	if evaluatedAt == "" {
		evaluatedAt = time.Now().UTC().Format(time.RFC3339)
	}
	providerName := ""
	providerVersion := ""
	if eval.Provider != nil {
		providerName = eval.Provider.Name
		providerVersion = eval.Provider.Version
	}

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO eval_records (eval_id, artifact_id, category, provider_name, provider_version,
		 benchmark_name, benchmark_version, evaluator_name, evaluator_method, overall_score, body, evaluated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		eval.ID, artifactID, string(eval.Category),
		providerName, providerVersion,
		eval.Benchmark.Name, eval.Benchmark.Version,
		eval.Evaluator.Name, eval.Evaluator.Type,
		eval.Results.OverallScore, string(body), evaluatedAt,
	)
	return err
}

func (s *SQLiteStore) ListEvalRecords(ctx context.Context, artifactID int64, filter *model.EvalFilter) ([]*model.EvalRecord, error) {
	where := []string{"artifact_id = ?"}
	args := []interface{}{artifactID}

	if filter != nil {
		if filter.Category != "" {
			where = append(where, "category = ?")
			args = append(args, string(filter.Category))
		}
		if filter.Benchmark != "" {
			where = append(where, "benchmark_name = ?")
			args = append(args, filter.Benchmark)
		}
		if filter.Provider != "" {
			where = append(where, "provider_name = ?")
			args = append(args, filter.Provider)
		}
	}

	rows, err := s.db.QueryContext(ctx,
		"SELECT body, created_at FROM eval_records WHERE "+strings.Join(where, " AND ")+" ORDER BY created_at DESC",
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("list eval records: %w", err)
	}
	defer rows.Close()

	var records []*model.EvalRecord
	for rows.Next() {
		var body, createdAt string
		if err := rows.Scan(&body, &createdAt); err != nil {
			return nil, err
		}
		var rec model.EvalRecord
		if err := json.Unmarshal([]byte(body), &rec); err != nil {
			return nil, err
		}
		rec.CreatedAt = createdAt
		records = append(records, &rec)
	}
	return records, nil
}

// Promotion log

func (s *SQLiteStore) LogPromotion(ctx context.Context, artifactID int64, from, to model.Status, comment string) error {
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO promotion_log (artifact_id, from_status, to_status, comment) VALUES (?, ?, ?, ?)",
		artifactID, string(from), string(to), comment,
	)
	return err
}

func (s *SQLiteStore) ListPromotions(ctx context.Context, artifactID int64) ([]model.PromotionEntry, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT from_status, to_status, comment, promoted_at FROM promotion_log WHERE artifact_id = ? ORDER BY promoted_at ASC",
		artifactID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []model.PromotionEntry
	for rows.Next() {
		var from, to, promoted string
		var comment sql.NullString
		if err := rows.Scan(&from, &to, &comment, &promoted); err != nil {
			return nil, err
		}
		e := model.PromotionEntry{
			From:      model.Status(from),
			To:        model.Status(to),
			Timestamp: promoted,
		}
		if comment.Valid {
			e.Comment = comment.String
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// Search

func (s *SQLiteStore) SearchArtifacts(ctx context.Context, query string, kinds []model.Kind, limit, offset int) ([]*model.RegistryArtifact, int, error) {
	kindFilter := ""
	args := []interface{}{query}
	if len(kinds) > 0 {
		placeholders := make([]string, len(kinds))
		for i, k := range kinds {
			placeholders[i] = "?"
			args = append(args, string(k))
		}
		kindFilter = " AND a.kind IN (" + strings.Join(placeholders, ",") + ")"
	}

	var total int
	countQuery := "SELECT COUNT(*) FROM artifacts_fts f JOIN artifacts a ON a.id = f.rowid WHERE artifacts_fts MATCH ?" + kindFilter
	if err := s.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count search: %w", err)
	}

	selectQuery := `SELECT a.body, a.status, a.is_latest, a.created_at, a.updated_at, a.published_at
		FROM artifacts_fts f JOIN artifacts a ON a.id = f.rowid
		WHERE artifacts_fts MATCH ?` + kindFilter + ` ORDER BY rank LIMIT ? OFFSET ?`
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, selectQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("search artifacts: %w", err)
	}
	defer rows.Close()

	return scanArtifacts(rows, total)
}

// Helpers

func scanArtifacts(rows *sql.Rows, total int) ([]*model.RegistryArtifact, int, error) {
	var artifacts []*model.RegistryArtifact
	for rows.Next() {
		var body, status, createdAt, updatedAt string
		var isLatest bool
		var publishedAt sql.NullString

		if err := rows.Scan(&body, &status, &isLatest, &createdAt, &updatedAt, &publishedAt); err != nil {
			return nil, 0, err
		}
		var a model.RegistryArtifact
		if err := json.Unmarshal([]byte(body), &a); err != nil {
			return nil, 0, err
		}
		a.Status = model.Status(status)
		a.IsLatest = isLatest
		a.CreatedAt = createdAt
		a.UpdatedAt = updatedAt
		if publishedAt.Valid {
			a.PublishedAt = publishedAt.String
		}
		artifacts = append(artifacts, &a)
	}
	return artifacts, total, nil
}
