package server

import (
	"log"
	"net/http"

	"github.com/ibrokethecloud/sim-cli/pkg/server/api"
	jsonstore "github.com/ibrokethecloud/sim-cli/pkg/server/store/json"
)

func Run(addr string, dataDir string) error {
	store, err := jsonstore.NewJSONStore(dataDir + "/data.json")

	if err != nil {
		return err
	}

	srv, err := api.NewServer(store, dataDir)
	if err != nil {
		return err
	}
	mux := http.NewServeMux()
	srv.RegisterRoutes(mux)

	log.Printf("Server listening on %s", addr)
	return http.ListenAndServe(addr, enableCors(mux))
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
