package sqlite

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/danmo-ai/danmo-film/core/port"
)

func IsFilmAssetID(id string) bool {
	return strings.HasPrefix(id, "fas_")
}

type AssetStore struct {
	db   *DB
	root string
}

func NewAssetStore(db *DB, filesRoot string) *AssetStore {
	_ = os.MkdirAll(filesRoot, 0o755)
	return &AssetStore{db: db, root: filesRoot}
}

func newID(prefix string, n int) string {
	b := make([]byte, n)
	_, _ = rand.Read(b)
	return prefix + hex.EncodeToString(b)
}

func nowISO() string {
	return time.Now().UTC().Truncate(time.Second).Format("2006-01-02T15:04:05Z")
}

func (s *AssetStore) CreateFromFile(src string, opts port.CreateAssetOpts) (*port.FilmAsset, error) {
	st, err := os.Stat(src)
	if err != nil || st.IsDir() {
		return nil, fmt.Errorf("Film asset source missing: %s", src)
	}
	aid := newID("fas_", 12)
	ext := filepath.Ext(src)
	if ext == "" {
		ext = ".bin"
	}
	dest := filepath.Join(s.root, aid+ext)
	if err := copyFile(src, dest); err != nil {
		return nil, err
	}
	created := nowISO()
	meta := opts.Metadata
	if meta == nil {
		meta = map[string]any{}
	}
	metaRaw, _ := json.Marshal(meta)
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	_, err = s.db.SQL.Exec(`
INSERT INTO film_assets (
  id, kind, mime_type, file_path, thumbnail_path,
  source_provider, provider_ref, source_task_id, metadata, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		aid, opts.Kind, opts.MimeType, dest, nil,
		nullIfEmpty(opts.SourceProvider), nullIfEmpty(opts.ProviderRef), nullIfEmpty(opts.SourceTaskID),
		string(metaRaw), created,
	)
	if err != nil {
		return nil, err
	}
	return s.getUnlocked(aid)
}

func (s *AssetStore) Get(id string) (*port.FilmAsset, error) {
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	return s.getUnlocked(id)
}

func (s *AssetStore) getUnlocked(id string) (*port.FilmAsset, error) {
	row := s.db.SQL.QueryRow(`SELECT * FROM film_assets WHERE id = ?`, id)
	a, err := scanAsset(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return a, err
}

func (s *AssetStore) ResolvePath(id string) (string, error) {
	row, err := s.Get(id)
	if err != nil {
		return "", err
	}
	if row == nil {
		return "", fmt.Errorf("Film asset not found: %s", id)
	}
	if _, err := os.Stat(row.FilePath); err != nil {
		return "", fmt.Errorf("Film asset file missing on disk: %s → %s", id, row.FilePath)
	}
	return row.FilePath, nil
}

func (s *AssetStore) List(kind string, limit, offset int) ([]*port.FilmAsset, error) {
	if limit < 1 {
		limit = 1
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	var rows *sql.Rows
	var err error
	if kind != "" {
		rows, err = s.db.SQL.Query(`SELECT * FROM film_assets WHERE kind = ? ORDER BY created_at DESC LIMIT ? OFFSET ?`, kind, limit, offset)
	} else {
		rows, err = s.db.SQL.Query(`SELECT * FROM film_assets ORDER BY created_at DESC LIMIT ? OFFSET ?`, limit, offset)
	}
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []*port.FilmAsset
	for rows.Next() {
		a, err := scanAssetRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (s *AssetStore) FindByProviderRef(providerID, ref string) (*port.FilmAsset, error) {
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	row := s.db.SQL.QueryRow(`
SELECT * FROM film_assets WHERE source_provider = ? AND provider_ref = ?
ORDER BY created_at DESC LIMIT 1`, providerID, ref)
	a, err := scanAsset(row)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return a, err
}

func (s *AssetStore) Delete(id string) (bool, error) {
	row, err := s.Get(id)
	if err != nil || row == nil {
		return false, err
	}
	s.db.mu.Lock()
	_, err = s.db.SQL.Exec(`DELETE FROM film_assets WHERE id = ?`, id)
	s.db.mu.Unlock()
	if err != nil {
		return false, err
	}
	_ = os.Remove(row.FilePath)
	if row.ThumbnailPath != "" {
		_ = os.Remove(row.ThumbnailPath)
	}
	return true, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanAsset(row scanner) (*port.FilmAsset, error) {
	var a port.FilmAsset
	var thumb, srcProv, provRef, srcTask, meta sql.NullString
	err := row.Scan(&a.ID, &a.Kind, &a.MimeType, &a.FilePath, &thumb, &srcProv, &provRef, &srcTask, &meta, &a.CreatedAt)
	if err != nil {
		return nil, err
	}
	a.ThumbnailPath = thumb.String
	a.SourceProvider = srcProv.String
	a.ProviderRef = provRef.String
	a.SourceTaskID = srcTask.String
	a.Metadata = map[string]any{}
	if meta.String != "" {
		_ = json.Unmarshal([]byte(meta.String), &a.Metadata)
	}
	a.ThumbnailURL = "/api/assets/" + a.ID + "/thumbnail"
	a.FileURL = "/api/assets/" + a.ID + "/file"
	return &a, nil
}

func scanAssetRows(rows *sql.Rows) (*port.FilmAsset, error) {
	return scanAsset(rows)
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
