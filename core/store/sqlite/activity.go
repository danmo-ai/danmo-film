package sqlite

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/danmo-ai/danmo-film/core/port"
)

type ActivityStore struct{ db *DB }

func NewActivityStore(db *DB) *ActivityStore { return &ActivityStore{db: db} }

func (s *ActivityStore) Append(ev port.ActivityEvent) (*port.ActivityEvent, error) {
	if strings.TrimSpace(ev.ProjectID) == "" {
		return nil, fmt.Errorf("project_id is required")
	}
	ev.ID = newID("lva_", 12)
	ev.CreatedAt = nowLocal()
	if ev.Detail == nil {
		ev.Detail = map[string]any{}
	}
	raw, _ := json.Marshal(ev.Detail)
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	_, err := s.db.SQL.Exec(`
INSERT INTO long_video_project_activity (
  id, project_id, category, event_type, phase, status, summary,
  task_id, parse_run_id, shot_id, detail_json, created_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		ev.ID, ev.ProjectID, ev.Category, ev.EventType, ev.Phase, ev.Status, ev.Summary,
		nullAny(ev.TaskID), nullAny(ev.ParseRunID), ev.ShotID, string(raw), ev.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &ev, nil
}

func (s *ActivityStore) List(projectID string, f port.ActivityFilter) ([]port.ActivityEvent, error) {
	limit, offset := f.Limit, f.Offset
	if limit < 1 {
		limit = 1
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}
	where := []string{"project_id = ?"}
	args := []any{projectID}
	if f.Category != "" {
		where = append(where, "category = ?")
		args = append(args, f.Category)
	}
	if f.Phase != "" {
		where = append(where, "phase = ?")
		args = append(args, f.Phase)
	}
	if f.EventType != "" {
		where = append(where, "event_type = ?")
		args = append(args, f.EventType)
	}
	if f.ParseRunID != "" {
		where = append(where, "parse_run_id = ?")
		args = append(args, f.ParseRunID)
	}
	if f.TaskID != "" {
		where = append(where, "task_id = ?")
		args = append(args, f.TaskID)
	}
	if f.ShotID != "" {
		where = append(where, "shot_id = ?")
		args = append(args, f.ShotID)
	}
	args = append(args, limit, offset)
	q := "SELECT * FROM long_video_project_activity WHERE " + strings.Join(where, " AND ") +
		" ORDER BY created_at DESC, id DESC LIMIT ? OFFSET ?"
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	rows, err := s.db.SQL.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []port.ActivityEvent
	for rows.Next() {
		ev, err := scanActivity(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, ev)
	}
	return out, rows.Err()
}

func (s *ActivityStore) Count(projectID string, f port.ActivityFilter) (int, error) {
	where := []string{"project_id = ?"}
	args := []any{projectID}
	if f.Category != "" {
		where = append(where, "category = ?")
		args = append(args, f.Category)
	}
	if f.EventType != "" {
		where = append(where, "event_type = ?")
		args = append(args, f.EventType)
	}
	if f.ParseRunID != "" {
		where = append(where, "parse_run_id = ?")
		args = append(args, f.ParseRunID)
	}
	s.db.mu.Lock()
	defer s.db.mu.Unlock()
	var n int
	err := s.db.SQL.QueryRow("SELECT COUNT(*) FROM long_video_project_activity WHERE "+strings.Join(where, " AND "), args...).Scan(&n)
	return n, err
}

func (s *ActivityStore) GetParseRun(projectID, parseRunID string) (map[string]any, error) {
	events, err := s.List(projectID, port.ActivityFilter{ParseRunID: parseRunID, Limit: 500})
	if err != nil || len(events) == 0 {
		return nil, err
	}
	sort.Slice(events, func(i, j int) bool { return events[i].CreatedAt < events[j].CreatedAt })
	var started, completed, failed *port.ActivityEvent
	var phases []map[string]any
	for i := range events {
		e := events[i]
		switch e.EventType {
		case "parse_started":
			started = &events[i]
		case "parse_completed":
			completed = &events[i]
		case "parse_failed":
			failed = &events[i]
		case "parse_phase":
			msg := ""
			if e.Detail != nil {
				msg, _ = e.Detail["message"].(string)
			}
			phases = append(phases, map[string]any{"phase": e.Phase, "message": msg, "at": e.CreatedAt})
		}
	}
	status := "running"
	if completed != nil {
		status = "completed"
	} else if failed != nil {
		status = "failed"
	}
	pick := started
	if completed != nil {
		pick = completed
	} else if failed != nil {
		pick = failed
	}
	startedAt := events[0].CreatedAt
	if started != nil {
		startedAt = started.CreatedAt
	}
	var completedAt any
	summary := ""
	detail := map[string]any{}
	if pick != nil {
		summary = pick.Summary
		detail = pick.Detail
	}
	if completed != nil {
		completedAt = completed.CreatedAt
	} else if failed != nil {
		completedAt = failed.CreatedAt
	}
	return map[string]any{
		"parse_run_id": parseRunID,
		"project_id":   projectID,
		"status":       status,
		"started_at":   startedAt,
		"completed_at": completedAt,
		"summary":      summary,
		"detail":       detail,
		"phases":       phases,
		"events":       events,
	}, nil
}

func scanActivity(rows *sql.Rows) (port.ActivityEvent, error) {
	var ev port.ActivityEvent
	var task, parse sql.NullString
	var detail string
	err := rows.Scan(&ev.ID, &ev.ProjectID, &ev.Category, &ev.EventType, &ev.Phase, &ev.Status, &ev.Summary, &task, &parse, &ev.ShotID, &detail, &ev.CreatedAt)
	if err != nil {
		return ev, err
	}
	if task.Valid {
		ev.TaskID = task.String
	}
	if parse.Valid {
		ev.ParseRunID = parse.String
	}
	ev.Detail = map[string]any{}
	_ = json.Unmarshal([]byte(detail), &ev.Detail)
	return ev, nil
}

func nullAny(v any) any {
	if v == nil {
		return nil
	}
	switch t := v.(type) {
	case string:
		if t == "" {
			return nil
		}
		return t
	default:
		return v
	}
}
