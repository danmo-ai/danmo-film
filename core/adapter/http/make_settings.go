package httpapi

import (
	"bytes"
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	genadapt "github.com/danmo-ai/danmo-film/core/adapter/gen"
)

var errMakeSettingsUnavailable = errors.New("settings route requires danmo-make generation provider")

func (a *App) registerMakeSettingsProxy(r *gin.Engine) {
	g := r.Group("/api/settings")
	g.GET("/presets", a.settingsPresetsGet)
	g.POST("/presets", a.forwardMakeSettings)
	g.DELETE("/presets/*path", a.forwardMakeSettings)
	g.GET("/loras", a.forwardMakeSettings)
	g.GET("/loras/compatible/*path", a.settingsCompatibleLoras)
	g.GET("/controlnets/compatible/*path", a.settingsCompatibleControlNets)
	g.GET("/registry", a.settingsModelRegistry)
}

func (a *App) makeGen() *genadapt.MakeProvider {
	gen := a.generation()
	m, ok := gen.(*genadapt.MakeProvider)
	if !ok {
		return nil
	}
	return m
}

func upstreamPath(c *gin.Context) string {
	p := c.Request.URL.Path
	if q := c.Request.URL.RawQuery; q != "" {
		p += "?" + q
	}
	return p
}

func (a *App) forwardMakeSettings(c *gin.Context) {
	m := a.makeGen()
	if m == nil {
		fail(c, 501, errMakeSettingsUnavailable)
		return
	}
	var body io.Reader
	if c.Request.Body != nil && c.Request.ContentLength != 0 {
		raw, err := io.ReadAll(c.Request.Body)
		if err != nil {
			fail(c, 400, err)
			return
		}
		body = bytes.NewReader(raw)
	}
	status, b, err := m.Forward(c.Request.Context(), c.Request.Method, upstreamPath(c), body)
	if err != nil {
		fail(c, 502, err)
		return
	}
	c.Data(status, "application/json", b)
}

func (a *App) settingsPresetsGet(c *gin.Context) {
	m := a.makeGen()
	if m == nil {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	status, b, err := m.Forward(c.Request.Context(), http.MethodGet, "/api/settings/presets", nil)
	if err != nil || status >= 400 {
		c.JSON(http.StatusOK, gin.H{})
		return
	}
	c.Data(status, "application/json", b)
}

func (a *App) settingsCompatibleLoras(c *gin.Context) {
	m := a.makeGen()
	if m == nil {
		c.JSON(http.StatusOK, []any{})
		return
	}
	status, b, err := m.Forward(c.Request.Context(), http.MethodGet, upstreamPath(c), nil)
	if err != nil || status >= 400 {
		c.JSON(http.StatusOK, []any{})
		return
	}
	c.Data(status, "application/json", b)
}

func (a *App) settingsCompatibleControlNets(c *gin.Context) {
	a.settingsCompatibleLoras(c)
}

func (a *App) settingsModelRegistry(c *gin.Context) {
	reg, err := a.generation().GetRegistry(c.Request.Context())
	if err != nil {
		m := a.makeGen()
		if m != nil {
			status, b, fwdErr := m.Forward(c.Request.Context(), http.MethodGet, "/api/settings/registry", nil)
			if fwdErr == nil && status < 400 {
				c.Data(status, "application/json", b)
				return
			}
		}
		c.JSON(http.StatusOK, gin.H{"models": gin.H{}})
		return
	}
	c.JSON(http.StatusOK, reg)
}
