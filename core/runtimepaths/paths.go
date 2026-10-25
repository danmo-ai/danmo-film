package runtimepaths

import (
	"os"
	"path/filepath"
)

func UserDataDir() string {
	if v := os.Getenv("DANMO_FILM_USER_DATA_DIR"); v != "" {
		abs, err := filepath.Abs(v)
		if err == nil {
			return abs
		}
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".danmo-film")
}

func WorkspaceDir() string {
	if v := os.Getenv("DANMO_FILM_WORKSPACE"); v != "" {
		abs, err := filepath.Abs(v)
		if err == nil {
			return abs
		}
		return v
	}
	wd, err := os.Getwd()
	if err != nil {
		wd = "."
	}
	return filepath.Join(wd, "db")
}

func FilmDBPath() string {
	p := filepath.Join(WorkspaceDir(), "film.db")
	_ = os.MkdirAll(filepath.Dir(p), 0o755)
	return p
}

func FilmAssetsRoot() string {
	p := filepath.Join(WorkspaceDir(), "assets")
	_ = os.MkdirAll(p, 0o755)
	return p
}

func SettingsPath() string {
	return filepath.Join(UserDataDir(), "config", "settings.json")
}
