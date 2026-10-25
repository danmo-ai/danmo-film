package httpapi

import (
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/danmo-ai/danmo-film/core/config"
	"github.com/danmo-ai/danmo-film/core/port"
	"github.com/danmo-ai/danmo-film/core/runtimepaths"
	"github.com/danmo-ai/danmo-film/core/store/sqlite"
)

type App struct {
	mu         sync.RWMutex
	Settings   config.Settings
	Assets     port.AssetStore
	Projects   port.ProjectStore
	Series     port.SeriesStore
	Characters port.CharacterLibraryStore
	Scenes     port.SceneLibraryStore
	Activity   port.ActivityStore
	Gen        port.GenerationProvider
	LLM        port.LLMProvider
}

func (a *App) generation() port.GenerationProvider {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.Gen
}

func (a *App) llm() port.LLMProvider {
	a.mu.RLock()
	defer a.mu.RUnlock()
	return a.LLM
}

func NewRouter(app *App) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(cors())

	r.GET("/api/system/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"ok": true, "product": "danmo-film",
			"user_data_dir": runtimepaths.UserDataDir(),
			"workspace_dir": runtimepaths.WorkspaceDir(),
			"go_version":    runtime.Version(),
		})
	})
	app.registerAssets(r)
	app.registerGeneration(r)
	app.registerProviders(r)
	app.registerProjects(r)
	app.registerSeries(r)
	app.registerLibrary(r)
	app.registerScriptParse(r)
	app.registerSettings(r)
	app.registerMakeSettingsProxy(r)

	dist := filepath.Join("out", "frontend", "dist")
	if st, err := os.Stat(dist); err == nil && st.IsDir() {
		r.Static("/assets", filepath.Join(dist, "assets"))
		r.NoRoute(func(c *gin.Context) {
			if strings.HasPrefix(c.Request.URL.Path, "/api/") {
				c.JSON(http.StatusNotFound, gin.H{"detail": "not found"})
				return
			}
			c.File(filepath.Join(dist, "index.html"))
		})
	}
	return r
}

func cors() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "*")
		c.Header("Access-Control-Allow-Headers", "*")
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	}
}

func fail(c *gin.Context, status int, err error) {
	c.JSON(status, gin.H{"detail": err.Error()})
}

func failStore(c *gin.Context, err error) {
	if isRevisionConflict(err) {
		fail(c, http.StatusConflict, err)
		return
	}
	fail(c, http.StatusInternalServerError, err)
}

func isRevisionConflict(err error) bool {
	var rev *sqlite.RevisionConflictError
	return errors.As(err, &rev)
}

func localeFrom(c *gin.Context, explicit string) string {
	if strings.TrimSpace(explicit) != "" {
		return strings.TrimSpace(explicit)
	}
	al := strings.ToLower(c.GetHeader("Accept-Language"))
	if strings.HasPrefix(al, "zh") {
		return "zh"
	}
	return "en"
}
