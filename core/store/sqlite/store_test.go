package sqlite

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/danmo-ai/danmo-film/core/port"
)

func TestAssetAndProjectRoundTrip(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "film.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	src := filepath.Join(dir, "a.txt")
	if err := os.WriteFile(src, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	assets := NewAssetStore(db, filepath.Join(dir, "assets"))
	row, err := assets.CreateFromFile(src, port.CreateAssetOpts{Kind: "image", MimeType: "text/plain", SourceProvider: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if !IsFilmAssetID(row.ID) {
		t.Fatal(row.ID)
	}
	got, err := assets.Get(row.ID)
	if err != nil || got == nil {
		t.Fatal(err)
	}
	ps := NewProjectStore(db)
	p, err := ps.Create("T", map[string]any{"shots": []any{}})
	if err != nil {
		t.Fatal(err)
	}
	list, err := ps.List(10)
	if err != nil || len(list) != 1 {
		t.Fatalf("%v %#v", err, list)
	}
	if p.Title != "T" {
		t.Fatal(p.Title)
	}
}

func TestSeriesEpisodeRoundTrip(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "film.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ps := NewProjectStore(db)
	ss := NewSeriesStore(db)

	series, err := ss.Create("短剧A", map[string]any{
		"characters": []any{map[string]any{"id": "c1", "name": "林婉"}},
		"scenes":     []any{},
	})
	if err != nil {
		t.Fatal(err)
	}
	ep1, err := ps.Create("第 1 集", map[string]any{"series_id": series.ID, "episode_number": 1, "shots": []any{}})
	if err != nil {
		t.Fatal(err)
	}
	link, err := ss.AddEpisode(series.ID, ep1.ID, "第 1 集", 1)
	if err != nil || link.EpisodeNumber != 1 {
		t.Fatalf("%v %#v", err, link)
	}
	n, err := ss.NextEpisodeNumber(series.ID)
	if err != nil || n != 2 {
		t.Fatalf("next=%d err=%v", n, err)
	}
	ep2, err := ps.Create("第 2 集", map[string]any{"series_id": series.ID, "episode_number": 2, "shots": []any{}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ss.AddEpisode(series.ID, ep2.ID, "第 2 集", 2); err != nil {
		t.Fatal(err)
	}
	eps, err := ss.ListEpisodes(series.ID)
	if err != nil || len(eps) != 2 {
		t.Fatalf("%v %#v", err, eps)
	}
	if eps[0].EpisodeNumber != 1 || eps[1].EpisodeNumber != 2 {
		t.Fatalf("order %#v", eps)
	}
	found, epMeta, err := ss.FindSeriesByEpisode(ep2.ID)
	if err != nil || found == nil || epMeta.EpisodeNumber != 2 {
		t.Fatalf("%v %#v %#v", err, found, epMeta)
	}
	if _, err := ss.Delete(series.ID); err == nil {
		t.Fatal("expected delete blocked while episodes exist")
	}
	if _, err := ss.RemoveEpisode(series.ID, ep1.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := ss.RemoveEpisode(series.ID, ep2.ID); err != nil {
		t.Fatal(err)
	}
	ok, err := ss.Delete(series.ID)
	if err != nil || !ok {
		t.Fatalf("delete empty series: %v %v", ok, err)
	}
}

func TestProjectRevisionConflict(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "film.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ps := NewProjectStore(db)
	p, err := ps.Create("T", map[string]any{"version": 1, "shots": []any{}})
	if err != nil {
		t.Fatal(err)
	}
	// Schema `version` alone must not trigger optimistic locking.
	if _, err := ps.Update(p.ID, nil, map[string]any{"version": 1, "shots": []any{}}); err != nil {
		t.Fatal(err)
	}
	got, _ := ps.Get(p.ID)
	if got.State["version"] != float64(1) && got.State["version"] != 1 {
		t.Fatalf("schema version must stay 1, got %v", got.State["version"])
	}
	if stateRevision(got.State) != 1 {
		t.Fatalf("expected revision 1 after first save, got %v", got.State["revision"])
	}
	// Matching revision succeeds and bumps.
	if _, err := ps.Update(p.ID, nil, map[string]any{"version": 1, "revision": 1, "shots": []any{}}); err != nil {
		t.Fatal(err)
	}
	got, _ = ps.Get(p.ID)
	if stateRevision(got.State) != 2 {
		t.Fatalf("revision %v", got.State["revision"])
	}
	// Stale revision conflicts.
	if _, err := ps.Update(p.ID, nil, map[string]any{"version": 1, "revision": 1, "shots": []any{}}); err == nil {
		t.Fatal("expected revision conflict")
	} else {
		var rev *RevisionConflictError
		if !errorsAsRevision(err, &rev) {
			t.Fatalf("unexpected err: %v", err)
		}
	}
}

func TestProjectSaveIgnoresSchemaVersionAsLock(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "film.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ps := NewProjectStore(db)
	p, err := ps.Create("T", map[string]any{"version": 1, "title": "a", "shots": []any{}})
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 3; i++ {
		if _, err := ps.Update(p.ID, nil, map[string]any{
			"version": 1,
			"title":   fmt.Sprintf("ep-%d", i),
			"shots":   []any{},
		}); err != nil {
			t.Fatalf("save %d without revision must succeed: %v", i, err)
		}
	}
}

func errorsAsRevision(err error, target **RevisionConflictError) bool {
	if err == nil {
		return false
	}
	if rev, ok := err.(*RevisionConflictError); ok {
		*target = rev
		return true
	}
	return false
}

func TestLibraryNewVersionRequiresEntity(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "film.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	lib := NewCharacterLibraryStore(db)
	if _, err := lib.NewVersion("flc_missing", map[string]any{"name": "x"}); err == nil {
		t.Fatal("expected error for unknown entity")
	}
}

func TestCharacterLibraryRoundTrip(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "film.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	lib := NewCharacterLibraryStore(db)
	row, err := lib.Create(map[string]any{"name": "Hero"})
	if err != nil || row.Version != 1 {
		t.Fatalf("%v %#v", err, row)
	}
	v2, err := lib.NewVersion(row.EntityID, map[string]any{"name": "Hero v2"})
	if err != nil || v2.Version != 2 {
		t.Fatalf("%v %#v", err, v2)
	}
	list, err := lib.List(10)
	if err != nil || len(list) != 1 || list[0].Version != 2 {
		t.Fatalf("%v %#v", err, list)
	}
}

func TestAssetReferenced(t *testing.T) {
	dir := t.TempDir()
	db, err := Open(filepath.Join(dir, "film.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	ss := NewSeriesStore(db)
	_, err = ss.Create("S", map[string]any{
		"characters": []any{map[string]any{
			"id": "c1", "looks": []any{map[string]any{"reference_asset_id": "fas_abc123"}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	ref, where, err := ss.AssetReferenced("fas_abc123")
	if err != nil || !ref || where == "" {
		t.Fatalf("ref=%v where=%q err=%v", ref, where, err)
	}
	ref, _, err = ss.AssetReferenced("fas_missing")
	if err != nil || ref {
		t.Fatalf("unexpected ref=%v err=%v", ref, err)
	}
}
