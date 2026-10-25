package httpapi

import (
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/danmo-ai/danmo-film/core/port"
	"github.com/danmo-ai/danmo-film/core/service/assemble"
	"github.com/danmo-ai/danmo-film/core/service/ingest"
	"github.com/danmo-ai/danmo-film/core/store/sqlite"
)

func (a *App) registerAssets(r *gin.Engine) {
	g := r.Group("/api/assets")
	g.POST("", a.uploadAsset)
	g.POST("/from-provider", a.ingestFromProvider)
	g.GET("", a.listAssets)
	g.GET("/groups", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"items": []any{}}) })
	g.GET("/:asset_id", a.getAsset)
	g.GET("/:asset_id/file", a.getAssetFile)
	g.GET("/:asset_id/thumbnail", a.getAssetThumb)
	g.DELETE("/:asset_id", a.deleteAsset)
	g.POST("/:asset_id/extract-last-frame", a.extractLastFrame)
}

func (a *App) uploadAsset(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil {
		fail(c, 400, err)
		return
	}
	kind := strings.TrimSpace(c.PostForm("kind"))
	tmp, err := os.CreateTemp("", "film_upload_*"+filepath.Ext(fh.Filename))
	if err != nil {
		fail(c, 500, err)
		return
	}
	defer os.Remove(tmp.Name())
	src, err := fh.Open()
	if err != nil {
		fail(c, 400, err)
		return
	}
	defer src.Close()
	if _, err := io.Copy(tmp, src); err != nil {
		fail(c, 500, err)
		return
	}
	tmp.Close()
	mt := fh.Header.Get("Content-Type")
	if mt == "" {
		mt = mime.TypeByExtension(filepath.Ext(fh.Filename))
	}
	if mt == "" {
		mt = "application/octet-stream"
	}
	if kind == "" {
		kind = ingest.GuessKind(mt, "image")
	}
	row, err := a.Assets.CreateFromFile(tmp.Name(), port.CreateAssetOpts{
		Kind: kind, MimeType: mt, SourceProvider: "film-upload",
		Metadata: map[string]any{"filename": fh.Filename},
	})
	if err != nil {
		fail(c, 500, err)
		return
	}
	c.JSON(http.StatusOK, row)
}

func (a *App) ingestFromProvider(c *gin.Context) {
	var body struct {
		ProviderAssetID string         `json:"provider_asset_id"`
		Kind            string         `json:"kind"`
		MimeType        string         `json:"mime_type"`
		SourceTaskID    string         `json:"source_task_id"`
		Metadata        map[string]any `json:"metadata"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, 400, err)
		return
	}
	if body.Kind == "" {
		body.Kind = "video"
	}
	row, err := ingest.IngestProviderAsset(c.Request.Context(), a.generation(), a.Assets, body.ProviderAssetID, body.Kind, body.MimeType, body.SourceTaskID, body.Metadata)
	if err != nil {
		fail(c, 502, err)
		return
	}
	c.JSON(http.StatusOK, row)
}

func (a *App) listAssets(c *gin.Context) {
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
	items, err := a.Assets.List(c.Query("kind"), limit, offset)
	if err != nil {
		fail(c, 500, err)
		return
	}
	if items == nil {
		items = []*port.FilmAsset{}
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total": len(items)})
}

func (a *App) getAsset(c *gin.Context) {
	row, err := a.Assets.Get(c.Param("asset_id"))
	if err != nil {
		fail(c, 500, err)
		return
	}
	if row == nil {
		c.JSON(404, gin.H{"detail": "asset not found: " + c.Param("asset_id")})
		return
	}
	c.JSON(http.StatusOK, row)
}

func (a *App) getAssetFile(c *gin.Context) {
	id := c.Param("asset_id")
	if !sqlite.IsFilmAssetID(id) {
		c.JSON(404, gin.H{"detail": "Not a Film asset id. Ingest provider results via POST /api/assets/from-provider first."})
		return
	}
	path, err := a.Assets.ResolvePath(id)
	if err != nil {
		fail(c, 404, err)
		return
	}
	row, _ := a.Assets.Get(id)
	mt := "application/octet-stream"
	if row != nil && row.MimeType != "" {
		mt = row.MimeType
	}
	c.Header("Content-Type", mt)
	c.File(path)
}

func (a *App) getAssetThumb(c *gin.Context) {
	id := c.Param("asset_id")
	row, err := a.Assets.Get(id)
	if err != nil || row == nil {
		c.JSON(404, gin.H{"detail": "asset not found: " + id})
		return
	}
	if row.ThumbnailPath != "" {
		if _, err := os.Stat(row.ThumbnailPath); err == nil {
			c.Header("Content-Type", "image/webp")
			c.File(row.ThumbnailPath)
			return
		}
	}
	if row.Kind == "image" {
		a.getAssetFile(c)
		return
	}
	c.JSON(404, gin.H{"detail": "thumbnail not available"})
}

func (a *App) extractLastFrame(c *gin.Context) {
	id := c.Param("asset_id")
	row, err := a.Assets.Get(id)
	if err != nil {
		fail(c, 500, err)
		return
	}
	if row == nil {
		c.JSON(404, gin.H{"detail": "asset not found: " + id})
		return
	}
	if row.Kind != "video" {
		c.JSON(400, gin.H{"detail": "extract-last-frame requires a video asset"})
		return
	}
	videoPath, err := a.Assets.ResolvePath(id)
	if err != nil {
		fail(c, 404, err)
		return
	}
	tmp, err := os.CreateTemp("", "film_lastframe_*.png")
	if err != nil {
		fail(c, 500, err)
		return
	}
	tmpPath := tmp.Name()
	tmp.Close()
	defer os.Remove(tmpPath)
	if err := assemble.ExtractLastFrame(videoPath, tmpPath); err != nil {
		fail(c, 502, err)
		return
	}
	out, err := a.Assets.CreateFromFile(tmpPath, port.CreateAssetOpts{
		Kind: "image", MimeType: "image/png", SourceProvider: "film-extract-last-frame",
		Metadata: map[string]any{"source_video_asset_id": id},
	})
	if err != nil {
		fail(c, 500, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (a *App) deleteAsset(c *gin.Context) {
	id := c.Param("asset_id")
	if a.Series != nil {
		ref, where, err := a.Series.AssetReferenced(id)
		if err != nil {
			fail(c, 500, err)
			return
		}
		if ref {
			c.JSON(409, gin.H{"detail": "asset is referenced by " + where})
			return
		}
	}
	ok, err := a.Assets.Delete(id)
	if err != nil {
		fail(c, 500, err)
		return
	}
	if !ok {
		c.JSON(404, gin.H{"detail": "asset not found: " + id})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}
