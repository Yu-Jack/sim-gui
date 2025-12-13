package server

import (
	"embed"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/Yu-Jack/sim-gui/pkg/server/api"
	jsonstore "github.com/Yu-Jack/sim-gui/pkg/server/store/json"
	"github.com/Yu-Jack/sim-gui/pkg/updater"
)

//go:embed all:static
var content embed.FS

func Run(addr string, dataDir string, dev bool) error {
	store, err := jsonstore.NewJSONStore(dataDir + "/data.json")

	if err != nil {
		return err
	}

	// Initialize update checker with 1 hour interval
	upd := updater.NewUpdater("Yu-Jack", "sim-gui", "main", 1*time.Hour)
	upd.Start()
	log.Println("Update checker started (checks every 1 hour)")

	srv, err := api.NewServer(store, dataDir, upd)
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	if !dev {
		if err := registerUIHandler(mux); err != nil {
			return err
		}
	}

	log.Printf("Server listening on http://localhost%s", addr)
	return http.ListenAndServe(addr, enableCors(mux))
}

func registerUIHandler(mux *http.ServeMux) error {
	// Serve UI
	assetsFS, err := fs.Sub(content, "static")
	if err != nil {
		return err
	}

	fileServer := http.FileServer(http.FS(assetsFS))

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api") {
			http.NotFound(w, r)
			return
		}

		// Check if the file exists in the assets
		path := strings.TrimPrefix(r.URL.Path, "/")
		if path == "" {
			path = "index.html"
		}

		_, err := assetsFS.Open(path)
		if os.IsNotExist(err) {
			// Serve index.html for SPA routing
			r.URL.Path = "/"
			fileServer.ServeHTTP(w, r)
			return
		}

		fileServer.ServeHTTP(w, r)
	})

	return nil
}

func enableCors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Allow all origins for development
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		// Handle preflight requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}
