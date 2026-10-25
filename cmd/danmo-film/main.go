package main

import (
	"log"
	"os"

	llmadapt "github.com/danmo-ai/danmo-film/core/adapter/llm"
	genadapt "github.com/danmo-ai/danmo-film/core/adapter/gen"
	httpapi "github.com/danmo-ai/danmo-film/core/adapter/http"
	"github.com/danmo-ai/danmo-film/core/config"
	"github.com/danmo-ai/danmo-film/core/runtimepaths"
	"github.com/danmo-ai/danmo-film/core/store/sqlite"
)

func main() {
	settings, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}
	db, err := sqlite.Open(runtimepaths.FilmDBPath())
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	llm, err := llmadapt.Resolve(settings)
	if err != nil {
		log.Fatal(err)
	}
	gen, err := genadapt.Resolve(settings)
	if err != nil {
		log.Fatal(err)
	}

	app := &httpapi.App{
		Settings:   settings,
		Assets:     sqlite.NewAssetStore(db, runtimepaths.FilmAssetsRoot()),
		Projects:   sqlite.NewProjectStore(db),
		Series:     sqlite.NewSeriesStore(db),
		Characters: sqlite.NewCharacterLibraryStore(db),
		Scenes:     sqlite.NewSceneLibraryStore(db),
		Activity:   sqlite.NewActivityStore(db),
		Gen:        gen,
		LLM:        llm,
	}
	r := httpapi.NewRouter(app)
	host := first(os.Getenv("DQ_HTTP_HOST"), os.Getenv("DANMO_FILM_HOST"), "127.0.0.1")
	port := first(os.Getenv("DQ_BACKEND_PORT"), os.Getenv("DANMO_FILM_PORT"), "7803")
	addr := host + ":" + port
	log.Printf("Danmo Film API http://%s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}

func first(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
