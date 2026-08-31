package repository

import (
	"context"
	"database/sql"
	"fmt"
)

// SQLitePolicyRepository implements PolicyRepository using SQLite.
type SQLitePolicyRepository struct {
	db *sql.DB
}

// NewSQLitePolicyRepository creates a new SQLite-backed policy repository.
func NewSQLitePolicyRepository(db *sql.DB) *SQLitePolicyRepository {
	return &SQLitePolicyRepository{db: db}
}

func (r *SQLitePolicyRepository) Get(ctx context.Context, id string, version int) (PolicyMetadata, error) {
	var meta PolicyMetadata
	err := r.db.QueryRowContext(ctx, `
		SELECT id, version, name, description
		FROM policies WHERE id = ? AND version = ?`, id, version,
	).Scan(&meta.ID, &meta.Version, &meta.Name, &meta.Description)
	if err == sql.ErrNoRows {
		return PolicyMetadata{}, fmt.Errorf("policy %s v%d not found", id, version)
	}
	if err != nil {
		return PolicyMetadata{}, fmt.Errorf("get policy: %w", err)
	}
	return meta, nil
}

func (r *SQLitePolicyRepository) Save(ctx context.Context, meta PolicyMetadata) error {
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO policies (id, version, name, description)
		VALUES (?, ?, ?, ?)
		ON CONFLICT (id, version) DO UPDATE SET
			name = excluded.name,
			description = excluded.description`,
		meta.ID, meta.Version, meta.Name, meta.Description,
	)
	if err != nil {
		return fmt.Errorf("save policy: %w", err)
	}
	return nil
}

func (r *SQLitePolicyRepository) Exists(ctx context.Context, id string, version int) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM policies WHERE id = ? AND version = ?`, id, version,
	).Scan(&count)
	if err != nil {
		return false, fmt.Errorf("check policy existence: %w", err)
	}
	return count > 0, nil
}

func (r *SQLitePolicyRepository) List(ctx context.Context) ([]PolicyMetadata, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, version, name, description
		FROM policies ORDER BY id, version`)
	if err != nil {
		return nil, fmt.Errorf("list policies: %w", err)
	}
	defer rows.Close()

	var metas []PolicyMetadata
	for rows.Next() {
		var meta PolicyMetadata
		if err := rows.Scan(&meta.ID, &meta.Version, &meta.Name, &meta.Description); err != nil {
			return nil, fmt.Errorf("scan policy: %w", err)
		}
		metas = append(metas, meta)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("list iterations: %w", err)
	}

	return metas, nil
}
