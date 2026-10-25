package sqlite

import (
	"database/sql"
	"encoding/json"
	"time"

	"github.com/danmo-ai/danmo-film/core/port"
)

type ProjectStore struct{ db *DB }

func NewProjectStore(db *DB) *ProjectStore { return &ProjectStore{db: db} }

func nowLocal() string { return time.Now().Format(time.RFC3339Nano) }

func parseState(raw string) map[string]any {
	var data map[string]any
	if json.Unmarshal([]byte(raw), &data) != nil || data == nil {
		return map[string]any{"version": 1, "shots": []any{}}
	}
	if _, ok := data["shots"].([]any); !ok {
		data["shots"] = []any{}
	}
	return data
}

func shotStats(state map[string]any) (shots, kf, seg int, hasFinal bool) {
	arr, _ := state["shots"].([]any)
	shots = len(arr)
	for _, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			continue
		}
		if str(m["keyframe_asset_id"]) != "" {
			kf++
		}
		if str(m["status"]) == "segment_ready" && str(m["segment_asset_id"]) != "" {
			seg++
		}
	}
	hasFinal = str(state["final_asset_id"]) != ""
	return
}

func str(v any) string {
	s, _ := v.(string)
	return s
}

func (s *ProjectStore) List(limit int) ([]port.ProjectSummary, error) {
	if limit < 1 {
		limit = 1
	}
	if limit > 200 {
		limit = 200
	}
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	rows, err := s.db.SQL.Query(`
SELECT id, title, state_json, created_at, updated_at
FROM long_video_projects ORDER BY updated_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []port.ProjectSummary
	for rows.Next() {
		var id, title, stateJSON, created, updated string
		if err := rows.Scan(&id, &title, &stateJSON, &created, &updated); err != nil {
			return nil, err
		}
		st := parseState(stateJSON)
		n, kf, seg, fin := shotStats(st)
		out = append(out, port.ProjectSummary{
			ID: id, Title: title, ShotCount: n, KeyframeCount: kf, SegmentCount: seg, HasFinal: fin,
			CreatedAt: created, UpdatedAt: updated,
		})
	}
	return out, rows.Err()
}

func (s *ProjectStore) Get(id string) (*port.ProjectDetail, error) {
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	row := s.db.SQL.QueryRow(`SELECT id, title, state_json, created_at, updated_at FROM long_video_projects WHERE id = ?`, id)
	var pid, title, stateJSON, created, updated string
	err := row.Scan(&pid, &title, &stateJSON, &created, &updated)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &port.ProjectDetail{ID: pid, Title: title, State: parseState(stateJSON), CreatedAt: created, UpdatedAt: updated}, nil
}

func (s *ProjectStore) Create(title string, state map[string]any) (*port.ProjectDetail, error) {
	if title == "" {
		title = "Long video"
	}
	if state == nil {
		state = map[string]any{}
	}
	if _, ok := state["version"]; !ok {
		state["version"] = 1
	}
	if _, ok := state["shots"]; !ok {
		state["shots"] = []any{}
	}
	pid := newID("lvp_", 10)
	now := nowLocal()
	raw, _ := json.Marshal(state)
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	_, err := s.db.SQL.Exec(`INSERT INTO long_video_projects (id, title, state_json, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`,
		pid, title, string(raw), now, now)
	if err != nil {
		return nil, err
	}
	return &port.ProjectDetail{ID: pid, Title: title, State: state, CreatedAt: now, UpdatedAt: now}, nil
}

func (s *ProjectStore) Update(id string, title *string, state map[string]any) (*port.ProjectDetail, error) {
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
					ResourceKind: "project", ResourceID: id,
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
	_, err = s.db.SQL.Exec(`UPDATE long_video_projects SET title = ?, state_json = ?, updated_at = ? WHERE id = ?`,
		newTitle, string(raw), now, id)
	if err != nil {
		return nil, err
	}
	return &port.ProjectDetail{ID: id, Title: newTitle, State: newState, CreatedAt: existing.CreatedAt, UpdatedAt: now}, nil
}

func (s *ProjectStore) Delete(id string) (bool, error) {
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	res, err := s.db.SQL.Exec(`DELETE FROM long_video_projects WHERE id = ?`, id)
	if err != nil {
		return false, err
	}
	n, _ := res.RowsAffected()
	return n > 0, nil
}
