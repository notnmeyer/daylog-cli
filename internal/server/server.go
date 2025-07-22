package server

import (
	"embed"
	"log"
	"net/http"
	"sync"
)

//go:embed templates/show.html
var fs embed.FS

func showHandler(wg *sync.WaitGroup) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		defer wg.Done()

		data, err := fs.ReadFile("templates/show.html")
		if err != nil {
			http.Error(w, "could not read embedded file", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/html")
		w.Write(data)
	}
}

func Start(wg *sync.WaitGroup) {
	server := &http.Server{Addr: ":8000"}

	http.HandleFunc("/show", showHandler(wg))

	if err := server.ListenAndServe(); err != http.ErrServerClosed {
		log.Fatalf("ListenAndServe(): %v", err)
	}
}
