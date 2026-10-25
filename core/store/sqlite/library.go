package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"

	"github.com/danmo-ai/danmo-film/core/port"
)

type libraryTable struct {
	name       string
	kind       string
	entityPref string
}

var characterLibraryTable = libraryTable{name: "character_library", kind: "character", entityPref: "flc_"}
var sceneLibraryTable = libraryTable{name: "scene_library", kind: "scene", entityPref: "fls_"}

type CharacterLibraryStore struct {
	db *DB
}

func NewCharacterLibraryStore(db *DB) *CharacterLibraryStore {
	return &CharacterLibraryStore{db: db}
}

type SceneLibraryStore struct {
	db *DB
}

func NewSceneLibraryStore(db *DB) *SceneLibraryStore {
	return &SceneLibraryStore{db: db}
}

func (s *CharacterLibraryStore) List(limit int) ([]port.LibraryEntityVersion, error) {
	return listLibraryLatest(s.db, characterLibraryTable, limit)
}

func (s *CharacterLibraryStore) Get(entityID string, version int) (*port.LibraryEntityVersion, error) {
	return getLibraryEntity(s.db, characterLibraryTable, entityID, version)
}

func (s *CharacterLibraryStore) Create(definition map[string]any) (*port.LibraryEntityVersion, error) {
	return createLibraryEntity(s.db, characterLibraryTable, definition)
}

func (s *CharacterLibraryStore) NewVersion(entityID string, definition map[string]any) (*port.LibraryEntityVersion, error) {
	return newLibraryVersion(s.db, characterLibraryTable, entityID, definition)
}

func (s *CharacterLibraryStore) ListVersions(entityID string) ([]port.LibraryEntityVersion, error) {
	return listLibraryVersions(s.db, characterLibraryTable, entityID)
}

func (s *CharacterLibraryStore) Delete(entityID string) (bool, error) {
	return deleteLibraryEntity(s.db, characterLibraryTable, entityID)
}

func (s *SceneLibraryStore) List(limit int) ([]port.LibraryEntityVersion, error) {
	return listLibraryLatest(s.db, sceneLibraryTable, limit)
}

func (s *SceneLibraryStore) Get(entityID string, version int) (*port.LibraryEntityVersion, error) {
	return getLibraryEntity(s.db, sceneLibraryTable, entityID, version)
}

func (s *SceneLibraryStore) Create(definition map[string]any) (*port.LibraryEntityVersion, error) {
	return createLibraryEntity(s.db, sceneLibraryTable, definition)
}

func (s *SceneLibraryStore) NewVersion(entityID string, definition map[string]any) (*port.LibraryEntityVersion, error) {
	return newLibraryVersion(s.db, sceneLibraryTable, entityID, definition)
}

func (s *SceneLibraryStore) ListVersions(entityID string) ([]port.LibraryEntityVersion, error) {
	return listLibraryVersions(s.db, sceneLibraryTable, entityID)
}

func (s *SceneLibraryStore) Delete(entityID string) (bool, error) {
	return deleteLibraryEntity(s.db, sceneLibraryTable, entityID)
}

func deleteLibraryEntity(db *DB, tab libraryTable, entityID string) (bool, error) {
	if entityID == "" {
		return false, fmt.Errorf("entity_id required")
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	res, err := db.SQL.Exec(fmt.Sprintf(`DELETE FROM %s WHERE entity_id = ?`, tab.name), entityID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func listLibraryLatest(db *DB, tab libraryTable, limit int) ([]port.LibraryEntityVersion, error) {
	if limit < 1 {
		limit = 1
	}
	if limit > 500 {
		limit = 500
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	q := fmt.Sprintf(`
SELECT l.entity_id, l.version, l.definition_json, l.created_at
FROM %s l
INNER JOIN (
  SELECT entity_id, MAX(version) AS max_version FROM %s GROUP BY entity_id
) latest ON latest.entity_id = l.entity_id AND latest.max_version = l.version
ORDER BY l.created_at DESC
LIMIT ?`, tab.name, tab.name)
	rows, err := db.SQL.Query(q, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLibraryRows(rows, tab.kind)
}

func getLibraryEntity(db *DB, tab libraryTable, entityID string, version int) (*port.LibraryEntityVersion, error) {
	if entityID == "" {
		return nil, nil
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	if version <= 0 {
		row := db.SQL.QueryRow(fmt.Sprintf(`
SELECT entity_id, version, definition_json, created_at FROM %s
WHERE entity_id = ? ORDER BY version DESC LIMIT 1`, tab.name), entityID)
		return scanLibraryRow(row, tab.kind)
	}
	row := db.SQL.QueryRow(fmt.Sprintf(`
SELECT entity_id, version, definition_json, created_at FROM %s
WHERE entity_id = ? AND version = ?`, tab.name), entityID, version)
	return scanLibraryRow(row, tab.kind)
}

func createLibraryEntity(db *DB, tab libraryTable, definition map[string]any) (*port.LibraryEntityVersion, error) {
	if definition == nil {
		definition = map[string]any{}
	}
	entityID := str(definition["entity_id"])
	if entityID == "" {
		entityID = str(definition["id"])
	}
	if entityID == "" {
		entityID = newID(tab.entityPref, 10)
	}
	definition["entity_id"] = entityID
	now := nowISO()
	raw, _ := json.Marshal(definition)
	db.mu.Lock()
	defer db.mu.Unlock()
	_, err := db.SQL.Exec(fmt.Sprintf(`
INSERT INTO %s (entity_id, version, definition_json, created_at) VALUES (?, 1, ?, ?)`, tab.name),
		entityID, string(raw), now)
	if err != nil {
		return nil, err
	}
	return &port.LibraryEntityVersion{
		EntityID: entityID, Version: 1, Kind: tab.kind,
		Definition: definition, CreatedAt: now,
	}, nil
}

func newLibraryVersion(db *DB, tab libraryTable, entityID string, definition map[string]any) (*port.LibraryEntityVersion, error) {
	if entityID == "" {
		return nil, fmt.Errorf("entity_id required")
	}
	if definition == nil {
		definition = map[string]any{}
	}
	definition["entity_id"] = entityID
	db.mu.Lock()
	defer db.mu.Unlock()
	var maxVer sql.NullInt64
	err := db.SQL.QueryRow(fmt.Sprintf(`SELECT MAX(version) FROM %s WHERE entity_id = ?`, tab.name), entityID).Scan(&maxVer)
	if err != nil {
		return nil, err
	}
	if !maxVer.Valid {
		return nil, fmt.Errorf("library entity not found: %s", entityID)
	}
	next := int(maxVer.Int64) + 1
	now := nowISO()
	raw, _ := json.Marshal(definition)
	_, err = db.SQL.Exec(fmt.Sprintf(`
INSERT INTO %s (entity_id, version, definition_json, created_at) VALUES (?, ?, ?, ?)`, tab.name),
		entityID, next, string(raw), now)
	if err != nil {
		return nil, err
	}
	return &port.LibraryEntityVersion{
		EntityID: entityID, Version: next, Kind: tab.kind,
		Definition: definition, CreatedAt: now,
	}, nil
}

func listLibraryVersions(db *DB, tab libraryTable, entityID string) ([]port.LibraryEntityVersion, error) {
	if entityID == "" {
		return nil, fmt.Errorf("entity_id required")
	}
	db.mu.Lock()
	defer db.mu.Unlock()
	rows, err := db.SQL.Query(fmt.Sprintf(`
SELECT entity_id, version, definition_json, created_at FROM %s
WHERE entity_id = ? ORDER BY version DESC`, tab.name), entityID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLibraryRows(rows, tab.kind)
}

func scanLibraryRows(rows *sql.Rows, kind string) ([]port.LibraryEntityVersion, error) {
	var out []port.LibraryEntityVersion
	for rows.Next() {
		var entityID, defJSON, created string
		var ver int
		if err := rows.Scan(&entityID, &ver, &defJSON, &created); err != nil {
			return nil, err
		}
		out = append(out, decodeLibraryRow(entityID, ver, kind, defJSON, created))
	}
	return out, rows.Err()
}

func scanLibraryRow(row *sql.Row, kind string) (*port.LibraryEntityVersion, error) {
	var entityID, defJSON, created string
	var ver int
	err := row.Scan(&entityID, &ver, &defJSON, &created)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	v := decodeLibraryRow(entityID, ver, kind, defJSON, created)
	return &v, nil
}

func decodeLibraryRow(entityID string, ver int, kind, defJSON, created string) port.LibraryEntityVersion {
	var def map[string]any
	_ = json.Unmarshal([]byte(defJSON), &def)
	if def == nil {
		def = map[string]any{}
	}
	return port.LibraryEntityVersion{
		EntityID: entityID, Version: ver, Kind: kind,
		Definition: def, CreatedAt: created,
	}
}
