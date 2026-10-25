package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/danmo-ai/danmo-film/core/port"
)

type SeriesStore struct{ db *DB }

func NewSeriesStore(db *DB) *SeriesStore { return &SeriesStore{db: db} }

func parseSeriesState(raw string) map[string]any {
	var data map[string]any
	if json.Unmarshal([]byte(raw), &data) != nil || data == nil {
		return map[string]any{"version": 1, "characters": []any{}, "scenes": []any{}}
	}
	if _, ok := data["characters"].([]any); !ok {
		data["characters"] = []any{}
	}
	if _, ok := data["scenes"].([]any); !ok {
		data["scenes"] = []any{}
	}
	return data
}

func (s *SeriesStore) List(limit int) ([]port.SeriesSummary, error) {
	if limit < 1 {
		limit = 1
	}
	if limit > 200 {
		limit = 200
	}
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	rows, err := s.db.SQL.Query(`
SELECT s.id, s.title, s.created_at, s.updated_at,
       (SELECT COUNT(*) FROM film_series_episodes e WHERE e.series_id = s.id) AS episode_count
FROM film_series s
ORDER BY s.updated_at DESC
LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []port.SeriesSummary
	for rows.Next() {
		var id, title, created, updated string
		var count int
		if err := rows.Scan(&id, &title, &created, &updated, &count); err != nil {
			return nil, err
		}
		out = append(out, port.SeriesSummary{
			ID: id, Title: title, EpisodeCount: count, CreatedAt: created, UpdatedAt: updated,
		})
	}
	return out, rows.Err()
}

func (s *SeriesStore) Get(id string) (*port.SeriesDetail, error) {
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	row := s.db.SQL.QueryRow(`SELECT id, title, state_json, created_at, updated_at FROM film_series WHERE id = ?`, id)
	var sid, title, stateJSON, created, updated string
	err := row.Scan(&sid, &title, &stateJSON, &created, &updated)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &port.SeriesDetail{ID: sid, Title: title, State: parseSeriesState(stateJSON), CreatedAt: created, UpdatedAt: updated}, nil
}

func (s *SeriesStore) Create(title string, state map[string]any) (*port.SeriesDetail, error) {
	if title == "" {
		title = "Untitled series"
	}
	if state == nil {
		state = map[string]any{}
	}
	if _, ok := state["version"]; !ok {
		state["version"] = 1
	}
	if _, ok := state["characters"]; !ok {
		state["characters"] = []any{}
	}
	if _, ok := state["scenes"]; !ok {
		state["scenes"] = []any{}
	}
	sid := newID("lvs_", 10)
	now := nowLocal()
	raw, _ := json.Marshal(state)
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	_, err := s.db.SQL.Exec(`INSERT INTO film_series (id, title, state_json, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		sid, title, string(raw), now, now)
	if err != nil {
		return nil, err
	}
	return &port.SeriesDetail{ID: sid, Title: title, State: state, CreatedAt: now, UpdatedAt: now}, nil
}

func (s *SeriesStore) Update(id string, title *string, state map[string]any) (*port.SeriesDetail, error) {
	existing, err := s.Get(id)
	if err != nil || existing == nil {
		return existing, err
	}
	now := nowLocal()
	newTitle := existing.Title
	if title != nil {
		newTitle = *title
	}
	newState := existing.State
	if state != nil {
		if clientRev, ok := clientSentRevision(state); ok {
			serverRev := stateRevision(existing.State)
			if clientRev != serverRev {
				return nil, &RevisionConflictError{
					ResourceKind: "series", ResourceID: id,
					ServerRevision: serverRev, ClientRevision: clientRev,
				}
			}
		}
		newState = state
		bumpRevision(newState)
	}
	raw, _ := json.Marshal(newState)
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	_, err = s.db.SQL.Exec(`UPDATE film_series SET title = ?, state_json = ?, updated_at = ? WHERE id = ?`,
		newTitle, string(raw), now, id)
	if err != nil {
		return nil, err
	}
	return &port.SeriesDetail{ID: id, Title: newTitle, State: newState, CreatedAt: existing.CreatedAt, UpdatedAt: now}, nil
}

func (s *SeriesStore) Delete(id string) (bool, error) {
	eps, err := s.ListEpisodes(id)
	if err != nil {
		return false, err
	}
	if len(eps) > 0 {
		return false, fmt.Errorf("series still has %d episode(s); remove episodes first", len(eps))
	}
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	res, err := s.db.SQL.Exec(`DELETE FROM film_series WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}

func (s *SeriesStore) ListEpisodes(seriesID string) ([]port.SeriesEpisodeSummary, error) {
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	rows, err := s.db.SQL.Query(`
SELECT e.series_id, e.episode_project_id, e.episode_number, e.title, e.created_at, e.updated_at,
       COALESCE(p.state_json, '{}'), COALESCE(p.updated_at, e.updated_at), COALESCE(p.title, '')
FROM film_series_episodes e
LEFT JOIN long_video_projects p ON p.id = e.episode_project_id
WHERE e.series_id = ?
ORDER BY e.episode_number ASC`, seriesID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []port.SeriesEpisodeSummary
	for rows.Next() {
		var seriesIDOut, projectID, epTitle, created, updated, stateJSON, projectUpdated, projectTitle string
		var num int
		if err := rows.Scan(&seriesIDOut, &projectID, &num, &epTitle, &created, &updated, &stateJSON, &projectUpdated, &projectTitle); err != nil {
			return nil, err
		}
		st := parseState(stateJSON)
		shots, kf, seg, fin := shotStats(st)
		// Prefer live project title (制作设置) over denormalized episode row.
		title := strings.TrimSpace(projectTitle)
		if title == "" {
			title = strings.TrimSpace(str(st["title"]))
		}
		if title == "" {
			title = strings.TrimSpace(epTitle)
		}
		if title == "" {
			title = fmt.Sprintf("Episode %d", num)
		}
		out = append(out, port.SeriesEpisodeSummary{
			SeriesID: seriesIDOut, EpisodeProjectID: projectID, EpisodeNumber: num, Title: title,
			ShotCount: shots, KeyframeCount: kf, SegmentCount: seg, HasFinal: fin,
			CreatedAt: created, UpdatedAt: projectUpdated,
		})
	}
	return out, rows.Err()
}

func (s *SeriesStore) NextEpisodeNumber(seriesID string) (int, error) {
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	var max sql.NullInt64
	err := s.db.SQL.QueryRow(`SELECT MAX(episode_number) FROM film_series_episodes WHERE series_id = ?`, seriesID).Scan(&max)
	if err != nil {
		return 0, err
	}
	if !max.Valid {
		return 1, nil
	}
	return int(max.Int64) + 1, nil
}

func (s *SeriesStore) AddEpisode(seriesID, projectID, title string, episodeNumber int) (*port.SeriesEpisodeSummary, error) {
	if seriesID == "" || projectID == "" {
		return nil, fmt.Errorf("series_id and episode_project_id required")
	}
	if episodeNumber < 1 {
		n, err := s.NextEpisodeNumber(seriesID)
		if err != nil {
			return nil, err
		}
		episodeNumber = n
	}
	now := nowLocal()
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	_, err := s.db.SQL.Exec(`
INSERT INTO film_series_episodes (series_id, episode_project_id, episode_number, title, created_at, updated_at)
VALUES (?, ?, ?, ?, ?, ?)`, seriesID, projectID, episodeNumber, title, now, now)
	if err != nil {
		return nil, err
	}
	_, _ = s.db.SQL.Exec(`UPDATE film_series SET updated_at = ? WHERE id = ?`, now, seriesID)
	return &port.SeriesEpisodeSummary{
		SeriesID: seriesID, EpisodeProjectID: projectID, EpisodeNumber: episodeNumber, Title: title,
		CreatedAt: now, UpdatedAt: now,
	}, nil
}

func (s *SeriesStore) UpdateEpisodeTitle(seriesID, projectID, title string) (*port.SeriesEpisodeSummary, error) {
	now := nowLocal()
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	res, err := s.db.SQL.Exec(`UPDATE film_series_episodes SET title = ?, updated_at = ? WHERE series_id = ? AND episode_project_id = ?`,
		title, now, seriesID, projectID)
	if err != nil {
		return nil, err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return nil, nil
	}
	_, _ = s.db.SQL.Exec(`UPDATE film_series SET updated_at = ? WHERE id = ?`, now, seriesID)
	var num int
	var created string
	err = s.db.SQL.QueryRow(`SELECT episode_number, created_at FROM film_series_episodes WHERE series_id = ? AND episode_project_id = ?`,
		seriesID, projectID).Scan(&num, &created)
	if err != nil {
		return nil, err
	}
	return &port.SeriesEpisodeSummary{
		SeriesID: seriesID, EpisodeProjectID: projectID, EpisodeNumber: num, Title: title,
		CreatedAt: created, UpdatedAt: now,
	}, nil
}

func (s *SeriesStore) RemoveEpisode(seriesID, projectID string) (bool, error) {
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	res, err := s.db.SQL.Exec(`DELETE FROM film_series_episodes WHERE series_id = ? AND episode_project_id = ?`, seriesID, projectID)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	if n > 0 {
		_, _ = s.db.SQL.Exec(`UPDATE film_series SET updated_at = ? WHERE id = ?`, nowLocal(), seriesID)
	}
	return n > 0, nil
}

func (s *SeriesStore) FindSeriesByEpisode(projectID string) (*port.SeriesDetail, *port.SeriesEpisodeSummary, error) {
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	row := s.db.SQL.QueryRow(`
SELECT e.series_id, e.episode_project_id, e.episode_number, e.title, e.created_at, e.updated_at,
       s.title, s.state_json, s.created_at, s.updated_at
FROM film_series_episodes e
JOIN film_series s ON s.id = e.series_id
WHERE e.episode_project_id = ?`, projectID)
	var seriesID, epProjectID, epTitle, epCreated, epUpdated string
	var seriesTitle, stateJSON, seriesCreated, seriesUpdated string
	var num int
	err := row.Scan(&seriesID, &epProjectID, &num, &epTitle, &epCreated, &epUpdated,
		&seriesTitle, &stateJSON, &seriesCreated, &seriesUpdated)
	if err == sql.ErrNoRows {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, err
	}
	return &port.SeriesDetail{
			ID: seriesID, Title: seriesTitle, State: parseSeriesState(stateJSON),
			CreatedAt: seriesCreated, UpdatedAt: seriesUpdated,
		}, &port.SeriesEpisodeSummary{
			SeriesID: seriesID, EpisodeProjectID: epProjectID, EpisodeNumber: num, Title: epTitle,
			CreatedAt: epCreated, UpdatedAt: epUpdated,
		}, nil
}

func jsonContainsAssetID(raw, assetID string) bool {
	if assetID == "" || raw == "" {
		return false
	}
	return strings.Contains(raw, assetID)
}

func jsonContainsLibraryEntityID(raw, entityID string) bool {
	if entityID == "" || raw == "" {
		return false
	}
	return strings.Contains(raw, entityID)
}

func (s *SeriesStore) LibraryReferenced(entityID string) (bool, string, error) {
	if entityID == "" {
		return false, "", nil
	}
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	rows, err := s.db.SQL.Query(`SELECT id, title, state_json FROM film_series`)
	if err != nil {
		return false, "", err
	}
	defer rows.Close()
	for rows.Next() {
		var id, title, stateJSON string
		if err := rows.Scan(&id, &title, &stateJSON); err != nil {
			return false, "", err
		}
		if jsonContainsLibraryEntityID(stateJSON, entityID) {
			return true, fmt.Sprintf("series %s (%s)", title, id), nil
		}
	}
	return false, "", rows.Err()
}

func (s *SeriesStore) AssetReferenced(assetID string) (bool, string, error) {
	if assetID == "" {
		return false, "", nil
	}
	s.db.mu.Lock()
	defer s.db.mu.Unlock()

	rows, err := s.db.SQL.Query(`SELECT id, title, state_json FROM film_series`)
	if err != nil {
		return false, "", err
	}
	defer rows.Close()
	for rows.Next() {
		var id, title, stateJSON string
		if err := rows.Scan(&id, &title, &stateJSON); err != nil {
			return false, "", err
		}
		if jsonContainsAssetID(stateJSON, assetID) {
			return true, fmt.Sprintf("series %s (%s)", title, id), nil
		}
	}
	if err := rows.Err(); err != nil {
		return false, "", err
	}

	prows, err := s.db.SQL.Query(`SELECT id, title, state_json FROM long_video_projects`)
	if err != nil {
		return false, "", err
	}
	defer prows.Close()
	for prows.Next() {
		var id, title, stateJSON string
		if err := prows.Scan(&id, &title, &stateJSON); err != nil {
			return false, "", err
		}
		if jsonContainsAssetID(stateJSON, assetID) {
			return true, fmt.Sprintf("project %s (%s)", title, id), nil
		}
	}
	return false, "", prows.Err()
}
