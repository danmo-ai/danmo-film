package httpapi

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/danmo-ai/danmo-film/core/service/ingest"
)

func (a *App) registerProviders(r *gin.Engine) {
	g := r.Group("/api/providers")
	g.GET("/status", a.providersStatus)
	g.GET("/models", a.listModels)
	g.POST("/proxy/images/generations", a.proxy("image"))
	g.POST("/proxy/videos/generations", a.proxy("video"))
	g.POST("/proxy/videos/edits", a.proxy("video_edit"))
	g.GET("/proxy/tasks/:task_id", a.getTask)
}

func (a *App) providersStatus(c *gin.Context) {
	gen := a.generation()
	llm := a.llm()
	genHealth, err := gen.Health(c.Request.Context())
	if err != nil {
		genHealth = map[string]any{"ok": false, "error": err.Error()}
	}
	c.JSON(http.StatusOK, gin.H{
		"generation": gin.H{"provider_id": gen.ID(), "health": genHealth},
		"llm":        gin.H{"provider_id": llm.ID(), "available": llm.Available()},
	})
}

func (a *App) proxy(action string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Body map[string]any `json:"body"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			fail(c, 400, err)
			return
		}
		if req.Body == nil {
			req.Body = map[string]any{}
		}
		gen := a.generation()
		staged, err := ingest.RewriteBodyAssetRefs(c.Request.Context(), req.Body, gen, a.Assets)
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
			err = fmt.Errorf("unknown action")
		}
		if err != nil {
			fail(c, 502, err)
			return
		}
		c.JSON(http.StatusOK, out)
	}
}
