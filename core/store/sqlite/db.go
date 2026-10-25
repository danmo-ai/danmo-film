package sqlite

import (
	"database/sql"
	"fmt"
	"sync"

	_ "modernc.org/sqlite"
)

type DB struct {
	SQL  *sql.DB
	mu   sync.Mutex
	path string
}

func Open(path string) (*DB, error) {
	sqlDB, err := sql.Open("sqlite", path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	sqlDB.SetMaxOpenConns(1)
	d := &DB{SQL: sqlDB, path: path}
	if err := d.migrate(); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return d, nil
}

func (d *DB) Close() error { return d.SQL.Close() }

func (d *DB) migrate() error {
	_, err := d.SQL.Exec(`
CREATE TABLE IF NOT EXISTS film_assets (
    id TEXT PRIMARY KEY,
    kind TEXT NOT NULL,
    mime_type TEXT NOT NULL,
    file_path TEXT NOT NULL,
    thumbnail_path TEXT,
    source_provider TEXT,
    provider_ref TEXT,
    source_task_id TEXT,
    metadata TEXT,
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_film_assets_kind ON film_assets(kind);
CREATE INDEX IF NOT EXISTS idx_film_assets_created ON film_assets(created_at);
CREATE INDEX IF NOT EXISTS idx_film_assets_provider_ref ON film_assets(source_provider, provider_ref);

CREATE TABLE IF NOT EXISTS long_video_projects (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    state_json TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_long_video_projects_updated ON long_video_projects(updated_at);

CREATE TABLE IF NOT EXISTS long_video_project_activity (
    id TEXT PRIMARY KEY,
    project_id TEXT NOT NULL,
    category TEXT NOT NULL,
    event_type TEXT NOT NULL,
    phase TEXT NOT NULL DEFAULT '',
    status TEXT NOT NULL DEFAULT '',
    summary TEXT NOT NULL DEFAULT '',
    task_id TEXT,
    parse_run_id TEXT,
    shot_id TEXT NOT NULL DEFAULT '',
    detail_json TEXT NOT NULL DEFAULT '{}',
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_lvp_activity_project_created ON long_video_project_activity(project_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_lvp_activity_task ON long_video_project_activity(task_id);
CREATE INDEX IF NOT EXISTS idx_lvp_activity_parse_run ON long_video_project_activity(parse_run_id, created_at ASC);
CREATE INDEX IF NOT EXISTS idx_lvp_activity_category ON long_video_project_activity(project_id, category, created_at DESC);

CREATE TABLE IF NOT EXISTS film_series (
    id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    state_json TEXT NOT NULL,
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_film_series_updated ON film_series(updated_at);

CREATE TABLE IF NOT EXISTS film_series_episodes (
    series_id TEXT NOT NULL,
    episode_project_id TEXT NOT NULL,
    episode_number INTEGER NOT NULL,
    title TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    PRIMARY KEY (series_id, episode_project_id),
    UNIQUE (series_id, episode_number)
);
CREATE INDEX IF NOT EXISTS idx_film_series_episodes_series ON film_series_episodes(series_id, episode_number);
CREATE INDEX IF NOT EXISTS idx_film_series_episodes_project ON film_series_episodes(episode_project_id);

CREATE TABLE IF NOT EXISTS character_library (
    entity_id TEXT NOT NULL,
    version INTEGER NOT NULL,
    definition_json TEXT NOT NULL,
    created_at TEXT NOT NULL,
    PRIMARY KEY (entity_id, version)
);
CREATE INDEX IF NOT EXISTS idx_character_library_entity ON character_library(entity_id, version DESC);

CREATE TABLE IF NOT EXISTS scene_library (
    entity_id TEXT NOT NULL,
    version INTEGER NOT NULL,
    definition_json TEXT NOT NULL,
    created_at TEXT NOT NULL,
    PRIMARY KEY (entity_id, version)
);
CREATE INDEX IF NOT EXISTS idx_scene_library_entity ON scene_library(entity_id, version DESC);
`)
	return err
}
