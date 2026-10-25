package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	genadapt "github.com/danmo-ai/danmo-film/core/adapter/gen"
	"github.com/danmo-ai/danmo-film/core/service/ingest"
)

func (a *App) registerGeneration(r *gin.Engine) {
	r.POST("/api/images/generations", a.submit("image"))
	r.POST("/api/videos/generations", a.submit("video"))
	r.POST("/api/videos/edits", a.submit("video_edit"))
	r.GET("/api/models", a.listModels)
	r.GET("/api/registry", a.getRegistry)
	r.GET("/api/adapters", a.listAdapters)
	r.GET("/api/tasks/:task_id", a.getTask)
	r.GET("/api/tasks/:task_id/stream", a.streamTask)
}

func (a *App) submit(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body map[string]any
		if err := c.ShouldBindJSON(&body); err != nil {
			fail(c, 400, fmt.Errorf("JSON object required"))
			return
		}
		gen, body, err := a.generationForBody(body, action)
		if err != nil {
			fail(c, 400, err)
			return
		}
		staged, err := ingest.RewriteBodyAssetRefs(c.Request.Context(), body, gen, a.Assets)
		if err != nil {
			fail(c, 502, err)
			return
		}
		var out map[string]any
		switch action {
		case "image":
			out, err = gen.CreateImage(c.Request.Context(), staged)
		case "video":
			out, err = gen.CreateVideo(c.Request.Context(), staged)
		case "video_edit":
			out, err = gen.EditVideo(c.Request.Context(), staged)
		default:
			err = fmt.Errorf("unknown action: %s", action)
		}
		if err != nil {
			fail(c, 502, err)
			return
		}
		c.JSON(http.StatusOK, out)
	}
}

func (a *App) listModels(c *gin.Context) {
	media := c.Query("media")
	if pid := strings.TrimSpace(c.Query("provider")); pid != "" {
		if r, ok := a.generation().(*genadapt.Router); ok {
			items, err := r.ListModelsFor(c.Request.Context(), pid, media)
			if err != nil {
				fail(c, 502, err)
				return
			}
			c.JSON(http.StatusOK, gin.H{"items": items})
			return
		}
	}
	items, err := a.generation().ListModels(c.Request.Context(), media)
	if err != nil {
		fail(c, 502, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (a *App) getRegistry(c *gin.Context) {
	reg, err := a.generation().GetRegistry(c.Request.Context())
	if err != nil {
		// UI treats missing registry as unsupported models; avoid hard 502 when Make is offline.
		c.JSON(http.StatusOK, gin.H{"models": gin.H{}})
		return
	}
	c.JSON(http.StatusOK, reg)
}

func (a *App) listAdapters(c *gin.Context) {
	v, err := a.generation().ListAdapters(c.Request.Context(), c.Query("for_model"))
	if err != nil {
		fail(c, 502, err)
		return
	}
	c.JSON(http.StatusOK, v)
}

func (a *App) getTask(c *gin.Context) {
	gen := a.generation()
	taskID := c.Param("task_id")
	// Prefer router so provider_id:native_id task refs resolve correctly.
	task, err := gen.GetTask(c.Request.Context(), taskID)
	if err != nil {
		fail(c, 502, err)
		return
	}
	kind := "video"
	if media, _ := task["media"].(string); media == "image" {
		kind = "image"
	} else if strings.HasPrefix(strings.ToLower(fmt.Sprint(task["id"])), "img_") ||
		strings.Contains(strings.ToLower(fmt.Sprint(task["id"])), ":img_") {
		kind = "image"
	}
	c.JSON(http.StatusOK, ingest.IngestTaskAssetIDs(c.Request.Context(), gen, a.Assets, task, kind))
}

func (a *App) streamTask(c *gin.Context) {
	gen := a.generation()
	ch, err := gen.IterTaskSSE(c.Request.Context(), c.Param("task_id"))
	if err != nil {
		fail(c, 502, err)
		return
	}
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Status(http.StatusOK)
	for ev := range ch {
		payload := ev.Data
		if ev.Event == "result" || ev.Event == "done" {
			var obj map[string]any
			if json.Unmarshal([]byte(ev.Data), &obj) == nil {
				if _, ok := obj["asset_ids"]; ok {
					rewritten := ingest.IngestTaskAssetIDs(c.Request.Context(), gen, a.Assets, obj, "video")
					b, _ := json.Marshal(rewritten)
					payload = string(b)
				}
			}
		}
		fmt.Fprintf(c.Writer, "event: %s\ndata: %s\n\n", ev.Event, payload)
		c.Writer.Flush()
	}
}
