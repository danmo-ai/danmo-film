package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/danmo-ai/danmo-film/core/port"
)

func (a *App) registerLibrary(r *gin.Engine) {
	ch := r.Group("/api/library/characters")
	ch.GET("", a.listCharacterLibrary)
	ch.POST("", a.createCharacterLibrary)
	ch.GET("/:entity_id", a.getCharacterLibrary)
	ch.GET("/:entity_id/versions", a.listCharacterLibraryVersions)
	ch.POST("/:entity_id/versions", a.newCharacterLibraryVersion)
	ch.DELETE("/:entity_id", a.deleteCharacterLibrary)

	sc := r.Group("/api/library/scenes")
	sc.GET("", a.listSceneLibrary)
	sc.POST("", a.createSceneLibrary)
	sc.GET("/:entity_id", a.getSceneLibrary)
	sc.GET("/:entity_id/versions", a.listSceneLibraryVersions)
	sc.POST("/:entity_id/versions", a.newSceneLibraryVersion)
	sc.DELETE("/:entity_id", a.deleteSceneLibrary)
}

func (a *App) listCharacterLibrary(c *gin.Context) {
	if a.Characters == nil {
		c.JSON(http.StatusOK, gin.H{"items": []any{}})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	items, err := a.Characters.List(limit)
	if err != nil {
		fail(c, 500, err)
		return
	}
	if items == nil {
		items = []port.LibraryEntityVersion{}
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (a *App) getCharacterLibrary(c *gin.Context) {
	if a.Characters == nil {
		fail(c, 501, errors.New("character library unavailable"))
		return
	}
	version, _ := strconv.Atoi(c.DefaultQuery("version", "0"))
	row, err := a.Characters.Get(c.Param("entity_id"), version)
	if err != nil {
		fail(c, 500, err)
		return
	}
	if row == nil {
		c.JSON(404, gin.H{"detail": "library character not found"})
		return
	}
	c.JSON(http.StatusOK, row)
}

func (a *App) createCharacterLibrary(c *gin.Context) {
	if a.Characters == nil {
		fail(c, 501, errors.New("character library unavailable"))
		return
	}
	var body struct {
		Definition map[string]any `json:"definition"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, 400, err)
		return
	}
	row, err := a.Characters.Create(body.Definition)
	if err != nil {
		fail(c, 500, err)
		return
	}
	c.JSON(http.StatusCreated, row)
}

func (a *App) listCharacterLibraryVersions(c *gin.Context) {
	if a.Characters == nil {
		fail(c, 501, errors.New("character library unavailable"))
		return
	}
	items, err := a.Characters.ListVersions(c.Param("entity_id"))
	if err != nil {
		fail(c, 500, err)
		return
	}
	if items == nil {
		items = []port.LibraryEntityVersion{}
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (a *App) newCharacterLibraryVersion(c *gin.Context) {
	if a.Characters == nil {
		fail(c, 501, errors.New("character library unavailable"))
		return
	}
	var body struct {
		Definition map[string]any `json:"definition"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, 400, err)
		return
	}
	row, err := a.Characters.NewVersion(c.Param("entity_id"), body.Definition)
	if err != nil {
		fail(c, 500, err)
		return
	}
	c.JSON(http.StatusCreated, row)
}

func (a *App) deleteCharacterLibrary(c *gin.Context) {
	if a.Characters == nil {
		fail(c, 501, errors.New("character library unavailable"))
		return
	}
	entityID := c.Param("entity_id")
	if a.Series != nil {
		if ref, where, err := a.Series.LibraryReferenced(entityID); err != nil {
			fail(c, 500, err)
			return
		} else if ref {
			c.JSON(409, gin.H{"detail": "library entity is referenced by " + where})
			return
		}
	}
	ok, err := a.Characters.Delete(entityID)
	if err != nil {
		fail(c, 500, err)
		return
	}
	if !ok {
		c.JSON(404, gin.H{"detail": "library character not found"})
		return
	}
	c.Status(http.StatusNoContent)
}

func (a *App) listSceneLibrary(c *gin.Context) {
	if a.Scenes == nil {
		c.JSON(http.StatusOK, gin.H{"items": []any{}})
		return
	}
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	items, err := a.Scenes.List(limit)
	if err != nil {
		fail(c, 500, err)
		return
	}
	if items == nil {
		items = []port.LibraryEntityVersion{}
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (a *App) getSceneLibrary(c *gin.Context) {
	if a.Scenes == nil {
		fail(c, 501, errors.New("scene library unavailable"))
		return
	}
	version, _ := strconv.Atoi(c.DefaultQuery("version", "0"))
	row, err := a.Scenes.Get(c.Param("entity_id"), version)
	if err != nil {
		fail(c, 500, err)
		return
	}
	if row == nil {
		c.JSON(404, gin.H{"detail": "library scene not found"})
		return
	}
	c.JSON(http.StatusOK, row)
}

func (a *App) createSceneLibrary(c *gin.Context) {
	if a.Scenes == nil {
		fail(c, 501, errors.New("scene library unavailable"))
		return
	}
	var body struct {
		Definition map[string]any `json:"definition"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, 400, err)
		return
	}
	row, err := a.Scenes.Create(body.Definition)
	if err != nil {
		fail(c, 500, err)
		return
	}
	c.JSON(http.StatusCreated, row)
}

func (a *App) listSceneLibraryVersions(c *gin.Context) {
	if a.Scenes == nil {
		fail(c, 501, errors.New("scene library unavailable"))
		return
	}
	items, err := a.Scenes.ListVersions(c.Param("entity_id"))
	if err != nil {
		fail(c, 500, err)
		return
	}
	if items == nil {
		items = []port.LibraryEntityVersion{}
	}
	c.JSON(http.StatusOK, gin.H{"items": items})
}

func (a *App) newSceneLibraryVersion(c *gin.Context) {
	if a.Scenes == nil {
		fail(c, 501, errors.New("scene library unavailable"))
		return
	}
	var body struct {
		Definition map[string]any `json:"definition"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		fail(c, 400, err)
		return
	}
	row, err := a.Scenes.NewVersion(c.Param("entity_id"), body.Definition)
	if err != nil {
		fail(c, 500, err)
		return
	}
	c.JSON(http.StatusCreated, row)
}

func (a *App) deleteSceneLibrary(c *gin.Context) {
	if a.Scenes == nil {
		fail(c, 501, errors.New("scene library unavailable"))
		return
	}
	entityID := c.Param("entity_id")
	if a.Series != nil {
		if ref, where, err := a.Series.LibraryReferenced(entityID); err != nil {
			fail(c, 500, err)
			return
		} else if ref {
			c.JSON(409, gin.H{"detail": "library entity is referenced by " + where})
			return
		}
	}
	ok, err := a.Scenes.Delete(entityID)
	if err != nil {
		fail(c, 500, err)
		return
	}
	if !ok {
		c.JSON(404, gin.H{"detail": "library scene not found"})
		return
	}
	c.Status(http.StatusNoContent)
}
