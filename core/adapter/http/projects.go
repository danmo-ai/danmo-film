package httpapi

import (
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/danmo-ai/danmo-film/core/port"
	"github.com/danmo-ai/danmo-film/core/service/assemble"
	"github.com/danmo-ai/danmo-film/core/service/ingest"
)

func (a *App) registerProjects(r *gin.Engine) {
	g := r.Group("/api/long-video")
	g.GET("/projects", a.listProjects)
	g.POST("/projects", a.createProject)
	g.GET("/projects/:project_id", a.getProject)
	g.PUT("/projects/:project_id", a.updateProject)
	g.DELETE("/projects/:project_id", a.deleteProject)
	g.GET("/projects/:project_id/activity", a.listActivity)
	g.GET("/projects/:project_id/activity/parse-runs/:parse_run_id", a.getParseRun)
	g.POST("/scene-grounding/depth-from-asset", func(c *gin.Context) {
		c.JSON(501, gin.H{"detail": "Scene-grounding depth encoding is not hosted in Danmo Film yet. Run depth on a generation provider and ingest the result into Film assets."})
	})
	g.POST("/assemble", a.assemble)
}

func (a *App) listProjects(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	items, err := a.Projects.List(limit)
	if err != nil {
		fail(c, 500, err)
		return
	}
	if items == nil {
		items = []port.ProjectSummary{}
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (a *App) createProject(c *gin.Context) {
	var body struct {
		Title string         `json:"title"`
		State map[string]any `json:"state"`
	}
	_ = c.ShouldBindJSON(&body)
	row, err := a.Projects.Create(body.Title, body.State)
	if err != nil {
		fail(c, 500, err)
		return
	}
	c.JSON(http.StatusCreated, row)
}

func (a *App) getProject(c *gin.Context) {
	row, err := a.Projects.Get(c.Param("project_id"))
	if err != nil {
		fail(c, 500, err)
		return
	}
	if row == nil {
		c.JSON(404, gin.H{"detail": "long video project not found"})
		return
	}
	c.JSON(http.StatusOK, row)
}

func (a *App) updateProject(c *gin.Context) {
	var body struct {
		Title *string        `json:"title"`
		State map[string]any `json:"state"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, 400, err)
		return
	}
	pid := c.Param("project_id")
	row, err := a.Projects.Update(pid, body.Title, body.State)
	if err != nil {
		failStore(c, err)
		return
	}
	if row == nil {
		c.JSON(404, gin.H{"detail": "long video project not found"})
		return
	}
	// Keep series episode row title in sync with project title (nav reads episode.title).
	if a.Series != nil {
		epTitle := strings.TrimSpace(row.Title)
		if epTitle == "" && body.State != nil {
			if t, ok := body.State["title"].(string); ok {
				epTitle = strings.TrimSpace(t)
			}
		}
		if epTitle != "" {
			if series, _, findErr := a.Series.FindSeriesByEpisode(pid); findErr == nil && series != nil {
				_, _ = a.Series.UpdateEpisodeTitle(series.ID, pid, epTitle)
			}
		}
	}
	c.JSON(http.StatusOK, row)
}

func (a *App) deleteProject(c *gin.Context) {
	pid := c.Param("project_id")
	if a.Series != nil {
		if series, _, err := a.Series.FindSeriesByEpisode(pid); err == nil && series != nil {
			_, _ = a.Series.RemoveEpisode(series.ID, pid)
		}
	}
	ok, err := a.Projects.Delete(pid)
	if err != nil {
		fail(c, 500, err)
		return
	}
	if !ok {
		c.JSON(404, gin.H{"detail": "long video project not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (a *App) listActivity(c *gin.Context) {
	pid := c.Param("project_id")
	if row, _ := a.Projects.Get(pid); row == nil {
		c.JSON(404, gin.H{"detail": "long video project not found"})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "200"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	f := port.ActivityFilter{
		Limit: limit, Offset: offset, Category: c.Query("category"), Phase: c.Query("phase"),
		EventType: c.Query("event_type"), ParseRunID: c.Query("parse_run_id"),
		TaskID: c.Query("task_id"), ShotID: c.Query("shot_id"),
	}
	items, err := a.Activity.List(pid, f)
	if err != nil {
		fail(c, 500, err)
		return
	}
	total, _ := a.Activity.Count(pid, f)
	if items == nil {
		items = []port.ActivityEvent{}
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": total})
}

func (a *App) getParseRun(c *gin.Context) {
	pid := c.Param("project_id")
	if row, _ := a.Projects.Get(pid); row == nil {
		c.JSON(404, gin.H{"detail": "long video project not found"})
		return
	}
	row, err := a.Activity.GetParseRun(pid, c.Param("parse_run_id"))
	if err != nil {
		fail(c, 500, err)
		return
	}
	if row == nil {
		c.JSON(404, gin.H{"detail": "parse run not found"})
		return
	}
	c.JSON(http.StatusOK, row)
}

func (a *App) assemble(c *gin.Context) {
	var body struct {
		SegmentAssetIDs []string `json:"segment_asset_ids"`
		OverlapFrames   int      `json:"overlap_frames"`
		FPS             float64  `json:"fps"`
		Title           string   `json:"title"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, 400, err)
		return
	}
	if len(body.SegmentAssetIDs) == 0 {
		c.JSON(400, gin.H{"detail": "segment_asset_ids is required"})
		return
	}
	if body.FPS == 0 {
		body.FPS = 16
	}
	tmp, err := os.MkdirTemp("", "film_assemble_*")
	if err != nil {
		fail(c, 500, err)
		return
	}
	defer os.RemoveAll(tmp)
	var paths []string
	for i, aid := range body.SegmentAssetIDs {
		dest := filepath.Join(tmp, fmt.Sprintf("seg_%03d.mp4", i))
		if err := ingest.MaterializeSegmentPath(c.Request.Context(), aid, dest, a.generation(), a.Assets); err != nil {
			fail(c, 502, err)
			return
		}
		paths = append(paths, dest)
	}
	out := filepath.Join(tmp, fmt.Sprintf("assembled_%s.mp4", time.Now().Format("20060102_150405")))
	if err := assemble.StitchSegmentPaths(paths, out, body.OverlapFrames, body.FPS); err != nil {
		fail(c, 502, err)
		return
	}
	row, err := a.Assets.CreateFromFile(out, port.CreateAssetOpts{
		Kind: "video", MimeType: "video/mp4", SourceProvider: "film-assemble",
		Metadata: map[string]any{"title": body.Title, "segment_count": len(body.SegmentAssetIDs), "segment_asset_ids": body.SegmentAssetIDs},
	})
	if err != nil {
		fail(c, 502, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"asset_id": row.ID, "title": body.Title, "segment_count": len(body.SegmentAssetIDs)})
}
