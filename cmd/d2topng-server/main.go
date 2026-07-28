// Command d2topng-server exposes the d2topng renderer as a small HTTP
// service: POST a D2 diagram, get a PNG back. Deliberately plain HTTP
// instead of MCP — any agent or script that can issue a POST can use it, no
// client-side protocol support required.
package main

import (
	"context"
	"errors"
	"fmt"
	"image/png"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/macsplit/d2topng/internal/render"
)

// maxSourceBytes bounds request bodies to keep the service resistant to
// trivial abuse without needing any external rate limiting.
const maxSourceBytes = 1 << 20 // 1 MiB of D2 source is already an enormous diagram

func main() {
	port := os.Getenv("PORT")
	if port == "" {
		port = "10000" // Render's own documented default
	}
	token := os.Getenv("D2TOPNG_API_TOKEN")
	if token == "" {
		log.Println("warning: D2TOPNG_API_TOKEN not set — /render is unauthenticated")
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", handleHealthz)
	mux.HandleFunc("POST /render", handleRender(token))

	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      30 * time.Second,
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("d2topng-server listening on :%s", port)
	if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatal(err)
	}
}

func handleHealthz(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}

// handleRender returns a handler for POST /render. Body is raw D2 source;
// optional ?scale=N query param (default 1) is passed straight through to
// render.Render. Responds with image/png on success, or a plain-text 400
// with D2's own compile diagnostics on invalid input.
func handleRender(token string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if token != "" && r.Header.Get("Authorization") != "Bearer "+token {
			http.Error(w, "unauthorized", http.StatusUnauthorized)
			return
		}

		scale := 1.0
		if s := r.URL.Query().Get("scale"); s != "" {
			v, err := strconv.ParseFloat(s, 64)
			if err != nil || v <= 0 {
				http.Error(w, "invalid scale parameter", http.StatusBadRequest)
				return
			}
			scale = v
		}

		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxSourceBytes))
		if err != nil {
			http.Error(w, "request body too large or unreadable", http.StatusBadRequest)
			return
		}
		if len(body) == 0 {
			http.Error(w, "empty request body: POST D2 source as the request body", http.StatusBadRequest)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), 20*time.Second)
		defer cancel()

		diagram, err := render.Compile(ctx, string(body))
		if err != nil {
			http.Error(w, fmt.Sprintf("d2 compile error:\n%s", err), http.StatusBadRequest)
			return
		}

		img, err := render.Render(diagram, scale)
		if err != nil {
			http.Error(w, fmt.Sprintf("render error: %s", err), http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "image/png")
		if err := png.Encode(w, img); err != nil {
			log.Printf("encode error: %v", err)
		}
	}
}
