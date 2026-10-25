package httpapi

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/danmo-ai/danmo-film/core/port"
)

func (a *App) registerSeries(r *gin.Engine) {
	g := r.Group("/api/long-video")
	g.GET("/series", a.listSeries)
	g.POST("/series", a.createSeries)
	g.GET("/series/:series_id", a.getSeries)
	g.PUT("/series/:series_id", a.updateSeries)
	g.DELETE("/series/:series_id", a.deleteSeries)
	g.GET("/series/:series_id/episodes", a.listSeriesEpisodes)
	g.POST("/series/:series_id/episodes", a.createSeriesEpisode)
	g.PUT("/series/:series_id/episodes/:project_id", a.updateSeriesEpisode)
	g.DELETE("/series/:series_id/episodes/:project_id", a.unlinkSeriesEpisode)
	g.POST("/series/from-project/:project_id", a.convertProjectToSeries)
}

func (a *App) listSeries(c *gin.Context) {
	if a.Series == nil {
		c.JSON(http.StatusOK, gin.H{"items": []any{}})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	items, err := a.Series.List(limit)
	if err != nil {
		fail(c, 500, err)
		return
	}
	if items == nil {
		items = []port.SeriesSummary{}
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (a *App) createSeries(c *gin.Context) {
	if a.Series == nil {
		fail(c, 501, fmt.Errorf("series store unavailable"))
		return
	}
	var body struct {
		Title string         `json:"title"`
		State map[string]any `json:"state"`
	}
	_ = c.ShouldBindJSON(&body)
	row, err := a.Series.Create(body.Title, body.State)
	if err != nil {
		fail(c, 500, err)
		return
	}
	c.JSON(http.StatusCreated, row)
}

func (a *App) getSeries(c *gin.Context) {
	if a.Series == nil {
		fail(c, 501, fmt.Errorf("series store unavailable"))
		return
	}
	row, err := a.Series.Get(c.Param("series_id"))
	if err != nil {
		fail(c, 500, err)
		return
	}
	if row == nil {
		c.JSON(404, gin.H{"detail": "series not found"})
		return
	}
	c.JSON(http.StatusOK, row)
}

func (a *App) updateSeries(c *gin.Context) {
	if a.Series == nil {
		fail(c, 501, fmt.Errorf("series store unavailable"))
		return
	}
	var body struct {
		Title *string        `json:"title"`
		State map[string]any `json:"state"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, 400, err)
		return
	}
	row, err := a.Series.Update(c.Param("series_id"), body.Title, body.State)
	if err != nil {
		failStore(c, err)
		return
	}
	if row == nil {
		c.JSON(404, gin.H{"detail": "series not found"})
		return
	}
	c.JSON(http.StatusOK, row)
}

func (a *App) deleteSeries(c *gin.Context) {
	if a.Series == nil {
		fail(c, 501, fmt.Errorf("series store unavailable"))
		return
	}
	ok, err := a.Series.Delete(c.Param("series_id"))
	if err != nil {
		fail(c, 409, err)
		return
	}
	if !ok {
		c.JSON(404, gin.H{"detail": "series not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (a *App) listSeriesEpisodes(c *gin.Context) {
	if a.Series == nil {
		c.JSON(http.StatusOK, gin.H{"items": []any{}})
		return
	}
	sid := c.Param("series_id")
	if row, _ := a.Series.Get(sid); row == nil {
		c.JSON(404, gin.H{"detail": "series not found"})
		return
	}
	items, err := a.Series.ListEpisodes(sid)
	if err != nil {
		fail(c, 500, err)
		return
	}
	if items == nil {
		items = []port.SeriesEpisodeSummary{}
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (a *App) createSeriesEpisode(c *gin.Context) {
	if a.Series == nil || a.Projects == nil {
		fail(c, 501, fmt.Errorf("series store unavailable"))
		return
	}
	sid := c.Param("series_id")
	series, err := a.Series.Get(sid)
	if err != nil {
		fail(c, 500, err)
		return
	}
	if series == nil {
		c.JSON(404, gin.H{"detail": "series not found"})
		return
	}
	var body struct {
		Title         string         `json:"title"`
		EpisodeNumber int            `json:"episode_number"`
		State         map[string]any `json:"state"`
	}
	_ = c.ShouldBindJSON(&body)
	num := body.EpisodeNumber
	if num < 1 {
		num, err = a.Series.NextEpisodeNumber(sid)
		if err != nil {
			fail(c, 500, err)
			return
		}
	}
	title := body.Title
	if title == "" {
		title = fmt.Sprintf("第 %d 集", num)
	}
	state := body.State
	if state == nil {
		state = map[string]any{}
	}
	state["series_id"] = sid
	state["episode_number"] = num
	if _, ok := state["title"]; !ok {
		state["title"] = title
	}
	// New episode inherits series defaults but starts empty for script/shots.
	if chars, ok := series.State["characters"]; ok && state["characters"] == nil {
		state["characters"] = []any{}
		_ = chars // characters live on series; episode uses variants
	}
	if scenes, ok := series.State["scenes"]; ok && state["scenes"] == nil {
		state["scenes"] = []any{}
		_ = scenes
	}
	if style, ok := series.State["style_anchor"]; ok {
		if _, exists := state["style_anchor"]; !exists {
			state["style_anchor"] = style
		}
	}
	for _, key := range []string{"keyframe_model", "segment_video_model", "segment_duration_sec", "overlap_frames", "chain_mode", "output_size", "target_duration_sec"} {
		if v, ok := series.State[key]; ok {
			if _, exists := state[key]; !exists {
				state[key] = v
			}
		}
	}
	state["shots"] = []any{}
	delete(state, "script_text")
	delete(state, "chapter_text")
	delete(state, "brief")
	delete(state, "chapter_analysis")
	delete(state, "final_asset_id")
	if state["character_variants"] == nil {
		state["character_variants"] = map[string]any{}
	}
	if state["scene_variants"] == nil {
		state["scene_variants"] = map[string]any{}
	}

	project, err := a.Projects.Create(title, state)
	if err != nil {
		fail(c, 500, err)
		return
	}
	ep, err := a.Series.AddEpisode(sid, project.ID, title, num)
	if err != nil {
		_, _ = a.Projects.Delete(project.ID)
		fail(c, 500, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"episode": ep,
		"project": project,
	})
}

func (a *App) updateSeriesEpisode(c *gin.Context) {
	if a.Series == nil {
		fail(c, 501, fmt.Errorf("series store unavailable"))
		return
	}
	var body struct {
		Title string `json:"title"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, 400, err)
		return
	}
	ep, err := a.Series.UpdateEpisodeTitle(c.Param("series_id"), c.Param("project_id"), body.Title)
	if err != nil {
		fail(c, 500, err)
		return
	}
	if ep == nil {
		c.JSON(404, gin.H{"detail": "episode not found"})
		return
	}
	c.JSON(http.StatusOK, ep)
}

func (a *App) unlinkSeriesEpisode(c *gin.Context) {
	if a.Series == nil {
		fail(c, 501, fmt.Errorf("series store unavailable"))
		return
	}
	ok, err := a.Series.RemoveEpisode(c.Param("series_id"), c.Param("project_id"))
	if err != nil {
		fail(c, 500, err)
		return
	}
	if !ok {
		c.JSON(404, gin.H{"detail": "episode not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (a *App) convertProjectToSeries(c *gin.Context) {
	if a.Series == nil || a.Projects == nil {
		fail(c, 501, fmt.Errorf("series store unavailable"))
		return
	}
	pid := c.Param("project_id")
	project, err := a.Projects.Get(pid)
	if err != nil {
		fail(c, 500, err)
		return
	}
	if project == nil {
		c.JSON(404, gin.H{"detail": "long video project not found"})
		return
	}
	if existing, _, _ := a.Series.FindSeriesByEpisode(pid); existing != nil {
		c.JSON(409, gin.H{"detail": "project already belongs to a series", "series_id": existing.ID})
		return
	}
	var body struct {
		Title string `json:"title"`
	}
	_ = c.ShouldBindJSON(&body)
	title := body.Title
	if title == "" {
		title = project.Title
	}
	seriesState := map[string]any{
		"version":      1,
		"characters":   project.State["characters"],
		"scenes":       project.State["scenes"],
		"style_anchor": project.State["style_anchor"],
	}
	for _, key := range []string{"keyframe_model", "segment_video_model", "segment_duration_sec", "overlap_frames", "chain_mode", "output_size", "target_duration_sec"} {
		if v, ok := project.State[key]; ok {
			seriesState[key] = v
		}
	}
	series, err := a.Series.Create(title, seriesState)
	if err != nil {
		fail(c, 500, err)
		return
	}
	nextState := project.State
	if nextState == nil {
		nextState = map[string]any{}
	}
	nextState["series_id"] = series.ID
	nextState["episode_number"] = 1
	if nextState["character_variants"] == nil {
		nextState["character_variants"] = map[string]any{}
	}
	if nextState["scene_variants"] == nil {
		nextState["scene_variants"] = map[string]any{}
	}
	// Keep episode-local roster copies for backward compatibility; effective roster prefers series.
	updated, err := a.Projects.Update(pid, nil, nextState)
	if err != nil {
		_, _ = a.Series.Delete(series.ID)
		fail(c, 500, err)
		return
	}
	epTitle := project.Title
	if epTitle == "" {
		epTitle = "第 1 集"
	}
	ep, err := a.Series.AddEpisode(series.ID, pid, epTitle, 1)
	if err != nil {
		fail(c, 500, err)
		return
	}
	c.JSON(http.StatusCreated, gin.H{
		"series":  series,
		"episode": ep,
		"project": updated,
	})
}
