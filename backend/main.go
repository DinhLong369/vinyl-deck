package main

import (
	"log"
	"net/http"
	"os"

	"github.com/local/vinyl-api/internal/api"
	"github.com/local/vinyl-api/internal/store"
	"github.com/local/vinyl-api/internal/youtube"
)

func env(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}

func main() {
	addr := env("ADDR", ":8080")
	dataFile := env("DATA_FILE", "data/records.json")
	origin := env("CORS_ORIGIN", "*")

	st, err := store.Open(dataFile)
	if err != nil {
		log.Fatalf("open store: %v", err)
	}
	yt := youtube.NewClient()
	if st.Len() == 0 {
		api.Seed(st, yt)
	}

	srv := api.New(st, yt, origin)
	log.Printf("vinyl-api listening on %s (data: %s)", addr, dataFile)
	log.Fatal(http.ListenAndServe(addr, srv.Handler()))
}
